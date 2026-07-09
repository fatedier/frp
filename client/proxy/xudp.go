// Copyright 2025 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !frps

package proxy

import (
	"io"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/fatedier/golib/errors"
	fmux "github.com/hashicorp/yamux"
	"github.com/quic-go/quic-go"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/naming"
	"github.com/fatedier/frp/pkg/nathole"
	"github.com/fatedier/frp/pkg/proto/udp"
	"github.com/fatedier/frp/pkg/transport"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

func init() {
	RegisterProxyFactory(reflect.TypeFor[*v1.XUDPProxyConfig](), NewXUDPProxy)
}

// XUDPProxy is the provider side of a hole-punched UDP proxy. The NAT traversal
// path is identical to XTCPProxy; only the per-stream handler differs: instead of
// dialing a local TCP service and splicing streams, it bridges each tunnel stream
// to the local UDP service using the sudp/udp packet forwarder.
type XUDPProxy struct {
	*BaseProxy

	cfg     *v1.XUDPProxyConfig
	metrics *p2pMetrics
}

func NewXUDPProxy(baseProxy *BaseProxy, cfg v1.ProxyConfigurer) Proxy {
	unwrapped, ok := cfg.(*v1.XUDPProxyConfig)
	if !ok {
		return nil
	}
	return &XUDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
		metrics:   newP2PMetrics(naming.AddUserPrefix(baseProxy.clientCfg.User, unwrapped.Name), baseProxy.msgTransporter),
	}
}

func (pxy *XUDPProxy) InWorkConn(conn net.Conn, _ *msg.StartWorkConn) {
	xl := pxy.xl
	defer conn.Close()
	// Report P2P tunnel traffic/sessions to frps (it cannot measure them itself).
	pxy.metrics.startReporter(pxy.ctx)
	// readNatHoleSid is shared with the xtcp provider (same package).
	natHoleSidMsg, err := readNatHoleSid(conn, pxy.clientCfg.Transport.WireProtocol)
	if err != nil {
		xl.Errorf("xudp read from workConn error: %v", err)
		return
	}

	xl.Tracef("nathole prepare start")

	// Prepare NAT traversal options
	var opts nathole.PrepareOptions
	if pxy.cfg.NatTraversal != nil && pxy.cfg.NatTraversal.DisableAssistedAddrs {
		opts.DisableAssistedAddrs = true
	}

	prepareResult, err := nathole.Prepare([]string{pxy.clientCfg.NatHoleSTUNServer}, opts)
	if err != nil {
		xl.Warnf("nathole prepare error: %v", err)
		return
	}

	xl.Infof("nathole prepare success, nat type: %s, behavior: %s, addresses: %v, assistedAddresses: %v",
		prepareResult.NatType, prepareResult.Behavior, prepareResult.Addrs, prepareResult.AssistedAddrs)
	defer prepareResult.ListenConn.Close()

	// send NatHoleClient msg to server
	transactionID := nathole.NewTransactionID()
	natHoleClientMsg := &msg.NatHoleClient{
		TransactionID: transactionID,
		ProxyName:     naming.AddUserPrefix(pxy.clientCfg.User, pxy.cfg.Name),
		Sid:           natHoleSidMsg.Sid,
		MappedAddrs:   prepareResult.Addrs,
		AssistedAddrs: prepareResult.AssistedAddrs,
	}

	xl.Tracef("nathole exchange info start")
	natHoleRespMsg, err := nathole.ExchangeInfo(pxy.ctx, pxy.msgTransporter, transactionID, natHoleClientMsg, 5*time.Second)
	if err != nil {
		xl.Warnf("nathole exchange info error: %v", err)
		return
	}

	xl.Infof("get natHoleRespMsg, sid [%s], protocol [%s], candidate address %v, assisted address %v, detectBehavior: %+v",
		natHoleRespMsg.Sid, natHoleRespMsg.Protocol, natHoleRespMsg.CandidateAddrs,
		natHoleRespMsg.AssistedAddrs, natHoleRespMsg.DetectBehavior)

	listenConn := prepareResult.ListenConn
	newListenConn, raddr, err := nathole.MakeHole(pxy.ctx, listenConn, natHoleRespMsg, []byte(pxy.cfg.Secretkey))
	if err != nil {
		listenConn.Close()
		xl.Warnf("make hole error: %v", err)
		_ = pxy.msgTransporter.Send(&msg.NatHoleReport{
			Sid:     natHoleRespMsg.Sid,
			Success: false,
		})
		return
	}
	listenConn = newListenConn
	xl.Infof("establishing nat hole connection successful, sid [%s], remoteAddr [%s]", natHoleRespMsg.Sid, raddr)

	_ = pxy.msgTransporter.Send(&msg.NatHoleReport{
		Sid:     natHoleRespMsg.Sid,
		Success: true,
	})

	if natHoleRespMsg.Protocol == "kcp" {
		pxy.listenByKCP(listenConn, raddr)
		return
	}

	// default is quic
	pxy.listenByQUIC(listenConn, raddr)
}

func (pxy *XUDPProxy) listenByKCP(listenConn *net.UDPConn, raddr *net.UDPAddr) {
	xl := pxy.xl
	listenConn.Close()
	laddr, _ := net.ResolveUDPAddr("udp", listenConn.LocalAddr().String())
	lConn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		xl.Warnf("dial udp error: %v", err)
		return
	}
	defer lConn.Close()

	remote, err := netpkg.NewKCPConnFromUDP(lConn, true, raddr.String())
	if err != nil {
		xl.Warnf("create kcp connection from udp connection error: %v", err)
		return
	}

	fmuxCfg := fmux.DefaultConfig()
	fmuxCfg.KeepAliveInterval = 10 * time.Second
	fmuxCfg.MaxStreamWindowSize = 6 * 1024 * 1024
	fmuxCfg.LogOutput = io.Discard
	session, err := fmux.Server(remote, fmuxCfg)
	if err != nil {
		xl.Errorf("create mux session error: %v", err)
		return
	}
	defer session.Close()

	for {
		muxConn, err := session.Accept()
		if err != nil {
			xl.Errorf("accept connection error: %v", err)
			return
		}
		go pxy.handleUDPWorkConnection(pxy.metrics.track(muxConn))
	}
}

func (pxy *XUDPProxy) listenByQUIC(listenConn *net.UDPConn, _ *net.UDPAddr) {
	xl := pxy.xl
	defer listenConn.Close()

	tlsConfig, err := transport.NewServerTLSConfig("", "", "")
	if err != nil {
		xl.Warnf("create tls config error: %v", err)
		return
	}
	tlsConfig.NextProtos = []string{"frp"}
	quicListener, err := quic.Listen(listenConn, tlsConfig,
		&quic.Config{
			MaxIdleTimeout:     time.Duration(pxy.clientCfg.Transport.QUIC.MaxIdleTimeout) * time.Second,
			MaxIncomingStreams: int64(pxy.clientCfg.Transport.QUIC.MaxIncomingStreams),
			KeepAlivePeriod:    time.Duration(pxy.clientCfg.Transport.QUIC.KeepalivePeriod) * time.Second,
		},
	)
	if err != nil {
		xl.Warnf("dial quic error: %v", err)
		return
	}
	// only accept one connection from raddr
	c, err := quicListener.Accept(pxy.ctx)
	if err != nil {
		xl.Errorf("quic accept connection error: %v", err)
		return
	}
	for {
		stream, err := c.AcceptStream(pxy.ctx)
		if err != nil {
			xl.Debugf("quic accept stream error: %v", err)
			_ = c.CloseWithError(0, "")
			return
		}
		go pxy.handleUDPWorkConnection(pxy.metrics.track(netpkg.QuicStreamToNetConn(stream, c)))
	}
}

// handleUDPWorkConnection bridges one tunnel stream to the local UDP service.
// It blocks (draining the stream into readCh) until the stream dies, then closes
// the channels so all reader/sender/forwarder goroutines unwind. Every writer to
// readCh/sendCh either ranges the channel or uses PanicToError, so closing here
// is race-safe.
func (pxy *XUDPProxy) handleUDPWorkConnection(stream net.Conn) {
	xl := pxy.xl

	remote, recycleFn, err := pxy.wrapWorkConn(stream, []byte(pxy.cfg.Secretkey))
	if err != nil {
		xl.Errorf("wrap xudp work connection error: %v", err)
		stream.Close()
		return
	}
	if recycleFn != nil {
		defer recycleFn()
	}
	workConn := netpkg.WrapReadWriteCloserToConn(remote, stream)

	localAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(pxy.cfg.LocalIP, strconv.Itoa(pxy.cfg.LocalPort)))
	if err != nil {
		workConn.Close()
		xl.Errorf("resolve local udp addr [%s:%d] error: %v", pxy.cfg.LocalIP, pxy.cfg.LocalPort, err)
		return
	}

	payloadRW := msg.NewReadWriter(workConn, pxy.clientCfg.Transport.WireProtocol)
	readCh := make(chan *msg.UDPPacket, 1024)
	sendCh := make(chan msg.Message, 1024)

	var closeOnce sync.Once
	closeFn := func() {
		closeOnce.Do(func() {
			workConn.Close()
			close(readCh)
			close(sendCh)
		})
	}
	defer closeFn()

	// sendCh -> tunnel stream
	go func() {
		for rawMsg := range sendCh {
			if errRet := payloadRW.WriteMsg(rawMsg); errRet != nil {
				xl.Warnf("xudp work write error: %v", errRet)
				closeFn()
				return
			}
		}
	}()
	// keep the tunnel alive so the visitor's read deadline does not fire when idle
	go func() {
		for {
			time.Sleep(30 * time.Second)
			if errRet := errors.PanicToError(func() { sendCh <- &msg.Ping{} }); errRet != nil {
				return
			}
		}
	}()
	// bridge readCh/sendCh to the local UDP service (non-blocking: spawns its own goroutines)
	udp.Forwarder(localAddr, readCh, sendCh, int(pxy.clientCfg.UDPPacketSize), pxy.cfg.Transport.ProxyProtocolVersion)

	// tunnel stream -> readCh (blocks until the stream dies)
	for {
		var udpMsg msg.UDPPacket
		if errRet := payloadRW.ReadMsgInto(&udpMsg); errRet != nil {
			xl.Debugf("xudp read from workConn stopped: %v", errRet)
			return
		}
		if errRet := errors.PanicToError(func() {
			readCh <- &udpMsg
		}); errRet != nil {
			return
		}
	}
}

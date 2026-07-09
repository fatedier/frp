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

// Stream tags — MUST match client/visitor/xtcpxudp.go.
const (
	xtcpxudpStreamTagTCP byte = 0x01
	xtcpxudpStreamTagUDP byte = 0x02
)

func init() {
	RegisterProxyFactory(reflect.TypeFor[*v1.XTCPXUDPProxyConfig](), NewXTCPXUDPProxy)
}

// XTCPXUDPProxy is the provider side of the combined proxy. It punches ONE NAT
// hole (identical to xtcp/xudp) and then, per accepted tunnel stream, reads a
// 1-byte tag to route the stream to the local TCP service (0x01) or the local UDP
// service (0x02).
type XTCPXUDPProxy struct {
	*BaseProxy

	cfg     *v1.XTCPXUDPProxyConfig
	metrics *p2pMetrics
}

func NewXTCPXUDPProxy(baseProxy *BaseProxy, cfg v1.ProxyConfigurer) Proxy {
	unwrapped, ok := cfg.(*v1.XTCPXUDPProxyConfig)
	if !ok {
		return nil
	}
	return &XTCPXUDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
		metrics:   newP2PMetrics(naming.AddUserPrefix(baseProxy.clientCfg.User, unwrapped.Name), baseProxy.msgTransporter),
	}
}

func (pxy *XTCPXUDPProxy) InWorkConn(conn net.Conn, startWorkConnMsg *msg.StartWorkConn) {
	// A hole-punch trigger is tagged Protocol="nathole". Anything else is a
	// relay-fallback work conn carrying the exact same 1-byte-tagged stream as a
	// P2P tunnel stream, so it goes straight to the shared stream handler.
	if startWorkConnMsg.Protocol != msg.StartWorkConnProtocolNatHole {
		pxy.handleStream(conn, startWorkConnMsg)
		return
	}
	pxy.runNatHole(conn, startWorkConnMsg)
}

// runNatHole performs the NAT hole-punch handshake and then serves tagged tunnel
// streams (QUIC/KCP) to the local TCP/UDP services.
func (pxy *XTCPXUDPProxy) runNatHole(conn net.Conn, startWorkConnMsg *msg.StartWorkConn) {
	xl := pxy.xl
	defer conn.Close()
	// Report P2P tunnel traffic/sessions to frps (it cannot measure them itself).
	pxy.metrics.startReporter(pxy.ctx)
	// readNatHoleSid is shared with the xtcp provider (same package).
	natHoleSidMsg, err := readNatHoleSid(conn, pxy.clientCfg.Transport.WireProtocol)
	if err != nil {
		xl.Errorf("xtcpxudp read from workConn error: %v", err)
		return
	}

	xl.Tracef("nathole prepare start")
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
		_ = pxy.msgTransporter.Send(&msg.NatHoleReport{Sid: natHoleRespMsg.Sid, Success: false})
		return
	}
	listenConn = newListenConn
	xl.Infof("establishing nat hole connection successful, sid [%s], remoteAddr [%s]", natHoleRespMsg.Sid, raddr)

	_ = pxy.msgTransporter.Send(&msg.NatHoleReport{Sid: natHoleRespMsg.Sid, Success: true})

	if natHoleRespMsg.Protocol == "kcp" {
		pxy.listenByKCP(listenConn, raddr, startWorkConnMsg)
		return
	}
	pxy.listenByQUIC(listenConn, raddr, startWorkConnMsg)
}

func (pxy *XTCPXUDPProxy) listenByKCP(listenConn *net.UDPConn, raddr *net.UDPAddr, startWorkConnMsg *msg.StartWorkConn) {
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
		go pxy.handleStream(pxy.metrics.track(muxConn), startWorkConnMsg)
	}
}

func (pxy *XTCPXUDPProxy) listenByQUIC(listenConn *net.UDPConn, _ *net.UDPAddr, startWorkConnMsg *msg.StartWorkConn) {
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
		go pxy.handleStream(pxy.metrics.track(netpkg.QuicStreamToNetConn(stream, c)), startWorkConnMsg)
	}
}

// handleStream reads the 1-byte routing tag and dispatches the stream to the TCP
// or UDP local service.
func (pxy *XTCPXUDPProxy) handleStream(stream net.Conn, startWorkConnMsg *msg.StartWorkConn) {
	xl := pxy.xl
	tag := make([]byte, 1)
	if _, err := io.ReadFull(stream, tag); err != nil {
		xl.Debugf("read xtcpxudp stream tag error: %v", err)
		stream.Close()
		return
	}
	switch tag[0] {
	case xtcpxudpStreamTagTCP:
		// Reuse the standard TCP handler; it dials LocalIP:LocalPort and joins.
		pxy.HandleTCPWorkConnection(stream, startWorkConnMsg, []byte(pxy.cfg.Secretkey))
	case xtcpxudpStreamTagUDP:
		pxy.handleUDPWorkConnection(stream)
	default:
		xl.Warnf("unknown xtcpxudp stream tag: %d", tag[0])
		stream.Close()
	}
}

// handleUDPWorkConnection bridges one tagged UDP stream to the local UDP service
// (LocalPortUDP, defaulting to LocalPort). It blocks on the read loop until the
// stream dies, then closes the channels so the forwarder goroutines unwind.
func (pxy *XTCPXUDPProxy) handleUDPWorkConnection(stream net.Conn) {
	xl := pxy.xl

	remote, recycleFn, err := pxy.wrapWorkConn(stream, []byte(pxy.cfg.Secretkey))
	if err != nil {
		xl.Errorf("wrap xtcpxudp work connection error: %v", err)
		stream.Close()
		return
	}
	if recycleFn != nil {
		defer recycleFn()
	}
	workConn := netpkg.WrapReadWriteCloserToConn(remote, stream)

	udpPort := pxy.cfg.LocalPortUDP
	if udpPort == 0 {
		udpPort = pxy.cfg.LocalPort
	}
	localAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(pxy.cfg.LocalIP, strconv.Itoa(udpPort)))
	if err != nil {
		workConn.Close()
		xl.Errorf("resolve local udp addr [%s:%d] error: %v", pxy.cfg.LocalIP, udpPort, err)
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

	go func() {
		for rawMsg := range sendCh {
			if errRet := payloadRW.WriteMsg(rawMsg); errRet != nil {
				xl.Warnf("xtcpxudp work write error: %v", errRet)
				closeFn()
				return
			}
		}
	}()
	go func() {
		for {
			time.Sleep(30 * time.Second)
			if errRet := errors.PanicToError(func() { sendCh <- &msg.Ping{} }); errRet != nil {
				return
			}
		}
	}()
	udp.Forwarder(localAddr, readCh, sendCh, int(pxy.clientCfg.UDPPacketSize), pxy.cfg.Transport.ProxyProtocolVersion)

	for {
		var udpMsg msg.UDPPacket
		if errRet := payloadRW.ReadMsgInto(&udpMsg); errRet != nil {
			xl.Debugf("xtcpxudp read from workConn stopped: %v", errRet)
			return
		}
		if errRet := errors.PanicToError(func() {
			readCh <- &udpMsg
		}); errRet != nil {
			return
		}
	}
}

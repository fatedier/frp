// Copyright 2024 The frp Authors
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
	"context"
	"net"
	"reflect"
	"time"

	"github.com/quic-go/quic-go"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/naming"
	"github.com/fatedier/frp/pkg/nathole"
	"github.com/fatedier/frp/pkg/transport"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

func init() {
	RegisterProxyFactory(reflect.TypeFor[*v1.XUDPProxyConfig](), NewXUDPProxy)
}

type XUDPProxy struct {
	*BaseProxy

	cfg *v1.XUDPProxyConfig
}

func NewXUDPProxy(baseProxy *BaseProxy, cfg v1.ProxyConfigurer) Proxy {
	unwrapped, ok := cfg.(*v1.XUDPProxyConfig)
	if !ok {
		return nil
	}
	return &XUDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

func (pxy *XUDPProxy) InWorkConn(conn net.Conn, startWorkConnMsg *msg.StartWorkConn) {
	xl := pxy.xl
	defer conn.Close()

	var natHoleSidMsg msg.NatHoleSid
	err := msg.ReadMsgInto(conn, &natHoleSidMsg)
	if err != nil {
		xl.Errorf("xudp read from workConn error: %v", err)
		return
	}

	xl.Tracef("xudp nathole prepare start")

	// Build STUN server list: primary + additional configured servers
	stunServers := []string{pxy.clientCfg.NatHoleSTUNServer}
	if len(pxy.cfg.STUNServers) > 0 {
		stunServers = append(stunServers, pxy.cfg.STUNServers...)
	}

	// Multi-STUN discovery for port prediction
	var portDelta int
	var multiSTUNLocalAddr string
	if len(stunServers) > 1 {
		multiResult, err := nathole.DiscoverMultiSTUN(pxy.ctx, stunServers, "")
		if err != nil {
			xl.Warnf("xudp multi-stun discover error (non-fatal): %v", err)
		} else {
			portDelta = multiResult.PortDelta
			if multiResult.LocalAddr != nil {
				multiSTUNLocalAddr = multiResult.LocalAddr.String()
			}
			xl.Infof("xudp multi-stun discovery success, addresses: %v, portDelta: %d",
				multiResult.Addrs, multiResult.PortDelta)
		}
	}

	// Prepare NAT traversal options
	var opts nathole.PrepareOptions
	if pxy.cfg.NatTraversal != nil && pxy.cfg.NatTraversal.DisableAssistedAddrs {
		opts.DisableAssistedAddrs = true
	}
	// Reuse the multi-STUN local address for Prepare if available,
	// so portDelta corresponds to the actual punched socket.
	opts.LocalAddr = multiSTUNLocalAddr

	pxy.doStandardHolePunch(natHoleSidMsg, stunServers[:1], opts, startWorkConnMsg, portDelta)
}

func (pxy *XUDPProxy) doStandardHolePunch(
	natHoleSidMsg msg.NatHoleSid,
	stunServers []string,
	opts nathole.PrepareOptions,
	startWorkConnMsg *msg.StartWorkConn,
	portDelta int,
) {
	xl := pxy.xl

	prepareResult, err := nathole.Prepare(stunServers, opts)
	if err != nil {
		xl.Warnf("xudp nathole prepare error: %v", err)
		return
	}

	xl.Infof("xudp nathole prepare success, nat type: %s, behavior: %s, addresses: %v, assistedAddresses: %v",
		prepareResult.NatType, prepareResult.Behavior, prepareResult.Addrs, prepareResult.AssistedAddrs)

	// Send NatHoleClient msg to server for realm rendezvous
	transactionID := nathole.NewTransactionID()
	natHoleClientMsg := &msg.NatHoleClient{
		TransactionID: transactionID,
		ProxyName:     naming.AddUserPrefix(pxy.clientCfg.User, pxy.cfg.Name),
		Sid:           natHoleSidMsg.Sid,
		MappedAddrs:   prepareResult.Addrs,
		AssistedAddrs: prepareResult.AssistedAddrs,
	}

	xl.Tracef("xudp rendezvous exchange start")
	natHoleRespMsg, err := nathole.XUDPRendezvousExchange(
		pxy.ctx, pxy.msgTransporter, transactionID, natHoleClientMsg, 5*time.Second,
	)
	if err != nil {
		prepareResult.ListenConn.Close()
		xl.Warnf("xudp rendezvous exchange error: %v", err)
		return
	}

	xl.Infof("xudp get natHoleRespMsg, sid [%s], candidate address %v, assisted address %v, detectBehavior: %+v",
		natHoleRespMsg.Sid, natHoleRespMsg.CandidateAddrs,
		natHoleRespMsg.AssistedAddrs, natHoleRespMsg.DetectBehavior)

	// Use XUDPMakeHole with port prediction
	listenConn := prepareResult.ListenConn
	newListenConn, raddr, err := nathole.XUDPMakeHole(pxy.ctx, listenConn, natHoleRespMsg, []byte(pxy.cfg.Secretkey), portDelta)
	if err != nil {
		listenConn.Close()
		xl.Warnf("xudp make hole error: %v", err)
		_ = pxy.msgTransporter.Send(&msg.NatHoleReport{
			Sid:     natHoleRespMsg.Sid,
			Success: false,
		})
		return
	}
	listenConn = newListenConn
	xl.Infof("xudp establishing nat hole connection successful, sid [%s], remoteAddr [%s]", natHoleRespMsg.Sid, raddr)

	_ = pxy.msgTransporter.Send(&msg.NatHoleReport{
		Sid:     natHoleRespMsg.Sid,
		Success: true,
	})

	// Start decoupled QUIC listener - this runs in an isolated goroutine
	// that survives control channel drops. Ownership of listenConn transfers
	// to the goroutine (it will close it when done).
	go pxy.listenByQUICDecoupled(listenConn, raddr, startWorkConnMsg)
}

// listenByQUICDecoupled starts a QUIC listener in a decoupled goroutine.
// The goroutine uses an independent context so it survives if the primary
// frpc-frps control channel drops. It owns listenConn and closes it on exit.
func (pxy *XUDPProxy) listenByQUICDecoupled(listenConn *net.UDPConn, _ *net.UDPAddr, startWorkConnMsg *msg.StartWorkConn) {
	xl := pxy.xl

	// Create an independent context for the tunnel lifecycle
	tunnelCtx, tunnelCancel := context.WithCancel(context.Background())
	defer tunnelCancel()
	defer listenConn.Close()

	tlsConfig, err := transport.NewServerTLSConfig("", "", "")
	if err != nil {
		xl.Warnf("xudp create tls config error: %v", err)
		return
	}
	tlsConfig.NextProtos = []string{"frp-xudp"}

	quicListener, err := quic.Listen(listenConn, tlsConfig,
		&quic.Config{
			MaxIdleTimeout:     time.Duration(pxy.clientCfg.Transport.QUIC.MaxIdleTimeout) * time.Second,
			MaxIncomingStreams: int64(pxy.clientCfg.Transport.QUIC.MaxIncomingStreams),
			KeepAlivePeriod:    time.Duration(pxy.clientCfg.Transport.QUIC.KeepalivePeriod) * time.Second,
		},
	)
	if err != nil {
		xl.Warnf("xudp quic listen error: %v", err)
		return
	}
	defer quicListener.Close()

	xl.Infof("xudp decoupled QUIC tunnel started, waiting for connections")

	// Accept one connection from the visitor
	c, err := quicListener.Accept(tunnelCtx)
	if err != nil {
		xl.Errorf("xudp quic accept connection error: %v", err)
		return
	}

	xl.Infof("xudp QUIC connection accepted from visitor")

	// Handle streams in the decoupled goroutine
	for {
		stream, err := c.AcceptStream(tunnelCtx)
		if err != nil {
			xl.Debugf("xudp quic accept stream error: %v", err)
			_ = c.CloseWithError(0, "")
			return
		}
		go pxy.HandleTCPWorkConnection(netpkg.QuicStreamToNetConn(stream, c), startWorkConnMsg, []byte(pxy.cfg.Secretkey))
	}
}

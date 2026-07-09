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
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/fatedier/golib/errors"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/udp"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

func init() {
	RegisterProxyFactory(reflect.TypeFor[*v1.TCPUDPProxyConfig](), NewTCPUDPProxy)
}

// TCPUDPProxy is the client (provider) side of the merged public-port proxy. The
// server hands it work connections tagged with a protocol: TCP work conns (empty
// protocol) go to the standard TCP handler, UDP work conns ("udp") drive a single
// UDP forwarder to the local UDP service. It carries no visitor — the public port
// on frps is the entry point.
type TCPUDPProxy struct {
	*BaseProxy

	cfg *v1.TCPUDPProxyConfig

	localUDPAddr *net.UDPAddr
	readCh       chan *msg.UDPPacket
	// include msg.UDPPacket and msg.Ping
	sendCh   chan msg.Message
	workConn net.Conn
	closed   bool
}

func NewTCPUDPProxy(baseProxy *BaseProxy, cfg v1.ProxyConfigurer) Proxy {
	unwrapped, ok := cfg.(*v1.TCPUDPProxyConfig)
	if !ok {
		return nil
	}
	return &TCPUDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

func (pxy *TCPUDPProxy) Run() (err error) {
	udpPort := pxy.cfg.LocalPortUDP
	if udpPort == 0 {
		udpPort = pxy.cfg.LocalPort
	}
	pxy.localUDPAddr, err = net.ResolveUDPAddr("udp", net.JoinHostPort(pxy.cfg.LocalIP, strconv.Itoa(udpPort)))
	if err != nil {
		return
	}
	return
}

func (pxy *TCPUDPProxy) Close() {
	pxy.mu.Lock()
	defer pxy.mu.Unlock()

	if !pxy.closed {
		pxy.closed = true
		if pxy.workConn != nil {
			pxy.workConn.Close()
		}
		if pxy.readCh != nil {
			close(pxy.readCh)
		}
		if pxy.sendCh != nil {
			close(pxy.sendCh)
		}
	}
}

func (pxy *TCPUDPProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	if m.Protocol == "udp" {
		pxy.handleUDPWorkConn(conn)
		return
	}
	// Empty/other protocol is treated as TCP (backward-compatible default).
	pxy.HandleTCPWorkConnection(conn, m, pxy.encryptionKey)
}

// handleUDPWorkConn bridges the single UDP work connection to the local UDP
// service. It mirrors UDPProxy.InWorkConn in udp.go.
func (pxy *TCPUDPProxy) handleUDPWorkConn(conn net.Conn) {
	xl := pxy.xl
	xl.Infof("incoming a new work connection for tcp+udp udp side, %s", conn.RemoteAddr().String())
	// close resources related with old workConn
	pxy.Close()

	remote, _, err := pxy.wrapWorkConn(conn, pxy.encryptionKey)
	if err != nil {
		xl.Errorf("wrap work connection: %v", err)
		return
	}

	pxy.mu.Lock()
	pxy.workConn = netpkg.WrapReadWriteCloserToConn(remote, conn)
	payloadRW := msg.NewReadWriter(pxy.workConn, pxy.clientCfg.Transport.WireProtocol)
	pxy.readCh = make(chan *msg.UDPPacket, 1024)
	pxy.sendCh = make(chan msg.Message, 1024)
	pxy.closed = false
	pxy.mu.Unlock()

	workConnReaderFn := func(rw msg.ReadWriter, readCh chan *msg.UDPPacket) {
		for {
			var udpMsg msg.UDPPacket
			if errRet := rw.ReadMsgInto(&udpMsg); errRet != nil {
				xl.Warnf("read from workConn for udp error: %v", errRet)
				return
			}
			if errRet := errors.PanicToError(func() {
				readCh <- &udpMsg
			}); errRet != nil {
				xl.Infof("reader goroutine for udp work connection closed: %v", errRet)
				return
			}
		}
	}
	workConnSenderFn := func(rw msg.ReadWriter, sendCh chan msg.Message) {
		defer func() {
			xl.Infof("writer goroutine for udp work connection closed")
		}()
		var errRet error
		for rawMsg := range sendCh {
			if errRet = rw.WriteMsg(rawMsg); errRet != nil {
				xl.Errorf("udp work write error: %v", errRet)
				return
			}
		}
	}
	heartbeatFn := func(sendCh chan msg.Message) {
		var errRet error
		for {
			time.Sleep(time.Duration(30) * time.Second)
			if errRet = errors.PanicToError(func() {
				sendCh <- &msg.Ping{}
			}); errRet != nil {
				xl.Tracef("heartbeat goroutine for udp work connection closed")
				break
			}
		}
	}

	go workConnSenderFn(payloadRW, pxy.sendCh)
	go workConnReaderFn(payloadRW, pxy.readCh)
	go heartbeatFn(pxy.sendCh)

	udp.Forwarder(pxy.localUDPAddr, pxy.readCh, pxy.sendCh, int(pxy.clientCfg.UDPPacketSize), pxy.cfg.Transport.ProxyProtocolVersion)
}

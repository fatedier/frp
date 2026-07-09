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

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/udp"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

// Stream tags carried as the first byte of every relayed work connection so the
// provider can route it to the TCP or UDP local service. MUST match
// client/visitor/stcpsudp.go.
const (
	stcpsudpStreamTagTCP byte = 0x01
	stcpsudpStreamTagUDP byte = 0x02
)

func init() {
	RegisterProxyFactory(reflect.TypeFor[*v1.STCPSUDPProxyConfig](), NewSTCPSUDPProxy)
}

// STCPSUDPProxy is the provider side of the merged secret proxy. Each work
// connection arriving from the frps relay is prefixed with a 1-byte tag written by
// the visitor: 0x01 routes to the local TCP service (plain stcp relay), 0x02 to
// the local UDP service (plain sudp relay). The tag lives OUTSIDE the optional
// encryption layer, so encryption keys are handled exactly like stcp/sudp.
type STCPSUDPProxy struct {
	*BaseProxy

	cfg *v1.STCPSUDPProxyConfig

	localUDPAddr *net.UDPAddr
	closeCh      chan struct{}
}

func NewSTCPSUDPProxy(baseProxy *BaseProxy, cfg v1.ProxyConfigurer) Proxy {
	unwrapped, ok := cfg.(*v1.STCPSUDPProxyConfig)
	if !ok {
		return nil
	}
	return &STCPSUDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
		closeCh:   make(chan struct{}),
	}
}

func (pxy *STCPSUDPProxy) Run() (err error) {
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

func (pxy *STCPSUDPProxy) Close() {
	pxy.mu.Lock()
	defer pxy.mu.Unlock()
	select {
	case <-pxy.closeCh:
		return
	default:
		close(pxy.closeCh)
	}
}

func (pxy *STCPSUDPProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	xl := pxy.xl
	tag := make([]byte, 1)
	if _, err := io.ReadFull(conn, tag); err != nil {
		xl.Errorf("read stcp+sudp stream tag error: %v", err)
		conn.Close()
		return
	}
	switch tag[0] {
	case stcpsudpStreamTagTCP:
		// Reuse the standard TCP handler (dials LocalIP:LocalPort and joins).
		pxy.HandleTCPWorkConnection(conn, m, pxy.encryptionKey)
	case stcpsudpStreamTagUDP:
		pxy.handleUDPWorkConnection(conn)
	default:
		xl.Warnf("unknown stcp+sudp stream tag: %d", tag[0])
		conn.Close()
	}
}

// handleUDPWorkConnection bridges one tagged UDP work connection to the local UDP
// service. It mirrors SUDPProxy.InWorkConn in sudp.go (same encryption key,
// heartbeat, and forwarder wiring).
func (pxy *STCPSUDPProxy) handleUDPWorkConnection(conn net.Conn) {
	xl := pxy.xl
	xl.Infof("incoming a new work connection for stcp+sudp udp side, %s", conn.RemoteAddr().String())

	remote, _, err := pxy.wrapWorkConn(conn, pxy.encryptionKey)
	if err != nil {
		xl.Errorf("wrap work connection: %v", err)
		conn.Close()
		return
	}

	workConn := netpkg.WrapReadWriteCloserToConn(remote, conn)
	payloadConn := msg.NewConn(workConn, msg.NewReadWriter(workConn, pxy.clientCfg.Transport.WireProtocol))
	readCh := make(chan *msg.UDPPacket, 1024)
	sendCh := make(chan msg.Message, 1024)
	isClose := false

	mu := &sync.Mutex{}
	closeFn := func() {
		mu.Lock()
		defer mu.Unlock()
		if isClose {
			return
		}
		isClose = true
		if workConn != nil {
			workConn.Close()
		}
		close(readCh)
		close(sendCh)
	}

	workConnReaderFn := func(payloadConn *msg.Conn, readCh chan *msg.UDPPacket) {
		defer closeFn()
		for {
			select {
			case <-pxy.closeCh:
				return
			default:
			}

			var udpMsg msg.UDPPacket
			if errRet := payloadConn.ReadMsgInto(&udpMsg); errRet != nil {
				xl.Warnf("read from workConn for stcp+sudp udp error: %v", errRet)
				return
			}
			if errRet := errors.PanicToError(func() {
				readCh <- &udpMsg
			}); errRet != nil {
				xl.Warnf("reader goroutine for stcp+sudp udp work connection closed: %v", errRet)
				return
			}
		}
	}

	workConnSenderFn := func(payloadConn *msg.Conn, sendCh chan msg.Message) {
		defer func() {
			closeFn()
			xl.Infof("writer goroutine for stcp+sudp udp work connection closed")
		}()
		var errRet error
		for rawMsg := range sendCh {
			if errRet = payloadConn.WriteMsg(rawMsg); errRet != nil {
				xl.Errorf("stcp+sudp udp work write error: %v", errRet)
				return
			}
		}
	}

	heartbeatFn := func(sendCh chan msg.Message) {
		ticker := time.NewTicker(30 * time.Second)
		defer func() {
			ticker.Stop()
			closeFn()
		}()
		var errRet error
		for {
			select {
			case <-ticker.C:
				if errRet = errors.PanicToError(func() {
					sendCh <- &msg.Ping{}
				}); errRet != nil {
					xl.Warnf("heartbeat goroutine for stcp+sudp udp work connection closed")
					return
				}
			case <-pxy.closeCh:
				return
			}
		}
	}

	go workConnSenderFn(payloadConn, sendCh)
	go workConnReaderFn(payloadConn, readCh)
	go heartbeatFn(sendCh)

	udp.Forwarder(pxy.localUDPAddr, readCh, sendCh, int(pxy.clientCfg.UDPPacketSize), pxy.cfg.Transport.ProxyProtocolVersion)
}

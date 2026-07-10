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
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/fatedier/golib/errors"
	"github.com/sandertv/gophertunnel/minecraft"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/udp"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
)

func init() {
	RegisterProxyFactory(reflect.TypeFor[*v1.PEProxyConfig](), NewPEProxy)
}

// PEProxy exposes Minecraft: Bedrock Edition servers behind frpc. frps opens the
// public UDP port and tunnels datagrams here (it treats this like a "udp"
// proxy); frpc runs a local Bedrock router (gophertunnel) that terminates each
// connection, reads the hostname from the login (clientData ServerAddress), and
// re-originates to the matching local server in ForcedHosts — the equivalent of
// WaterdogPE forced_hosts. frp's udp forwarder gives each player a distinct
// loopback source socket, so the router sees them as separate RakNet sessions.
type PEProxy struct {
	*BaseProxy
	cfg *v1.PEProxyConfig

	router    *peRouter
	localAddr *net.UDPAddr

	readCh   chan *msg.UDPPacket
	sendCh   chan msg.Message
	workConn net.Conn
	closed   bool
}

func NewPEProxy(baseProxy *BaseProxy, cfg v1.ProxyConfigurer) Proxy {
	unwrapped, ok := cfg.(*v1.PEProxyConfig)
	if !ok {
		return nil
	}
	return &PEProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

func (pxy *PEProxy) Run() (err error) {
	xl := pxy.xl
	// Start the local Bedrock router on an ephemeral loopback UDP port. frp's
	// udp tunnel feeds each player to it through a distinct per-client socket
	// (see pkg/proto/udp.Forwarder), so gophertunnel distinguishes RakNet
	// sessions correctly.
	router, err := newPERouter(pxy.cfg.ForcedHosts, xl)
	if err != nil {
		return err
	}
	pxy.router = router
	pxy.localAddr = router.Addr()
	xl.Infof("pe proxy started bedrock router on %s (%d forced hosts)", pxy.localAddr, len(pxy.cfg.ForcedHosts))
	return nil
}

// closeWorkConn tears down only the per-work-connection resources; the router
// persists across work-conn reconnects.
func (pxy *PEProxy) closeWorkConn() {
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

func (pxy *PEProxy) Close() {
	pxy.closeWorkConn()
	if pxy.router != nil {
		pxy.router.Close()
	}
}

func (pxy *PEProxy) InWorkConn(conn net.Conn, _ *msg.StartWorkConn) {
	xl := pxy.xl
	xl.Infof("incoming a new work connection for pe proxy, %s", conn.RemoteAddr().String())
	// close resources related with old workConn (but keep the router alive)
	pxy.closeWorkConn()

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
				xl.Warnf("read from workConn for pe error: %v", errRet)
				return
			}
			if errRet := errors.PanicToError(func() {
				readCh <- &udpMsg
			}); errRet != nil {
				xl.Infof("reader goroutine for pe work connection closed: %v", errRet)
				return
			}
		}
	}
	workConnSenderFn := func(rw msg.ReadWriter, sendCh chan msg.Message) {
		defer xl.Infof("writer goroutine for pe work connection closed")
		var errRet error
		for rawMsg := range sendCh {
			if errRet = rw.WriteMsg(rawMsg); errRet != nil {
				xl.Errorf("pe work write error: %v", errRet)
				return
			}
		}
	}
	heartbeatFn := func(sendCh chan msg.Message) {
		var errRet error
		for {
			time.Sleep(30 * time.Second)
			if errRet = errors.PanicToError(func() {
				sendCh <- &msg.Ping{}
			}); errRet != nil {
				xl.Tracef("heartbeat goroutine for pe work connection closed")
				break
			}
		}
	}

	go workConnSenderFn(payloadRW, pxy.sendCh)
	go workConnReaderFn(payloadRW, pxy.readCh)
	go heartbeatFn(pxy.sendCh)

	// Bridge the frp udp tunnel to the local Bedrock router (empty proxy
	// protocol version = none).
	udp.Forwarder(pxy.localAddr, pxy.readCh, pxy.sendCh, int(pxy.clientCfg.UDPPacketSize), "")
}

// peRouter is the local Bedrock host-router (gophertunnel): it accepts RakNet
// connections, reads the login ServerAddress, and re-originates to the backend
// mapped in ForcedHosts.
type peRouter struct {
	listener *minecraft.Listener
	forced   map[string]string // lowercased host -> backend "ip:port"
	xl       *xlog.Logger
}

func newPERouter(forcedHosts map[string]string, xl *xlog.Logger) (*peRouter, error) {
	forced := make(map[string]string, len(forcedHosts))
	for h, b := range forcedHosts {
		forced[strings.ToLower(h)] = b
	}
	listener, err := minecraft.ListenConfig{
		AuthenticationDisabled: true,
	}.Listen("raknet", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	r := &peRouter{listener: listener, forced: forced, xl: xl}
	go r.acceptLoop()
	return r, nil
}

func (r *peRouter) Addr() *net.UDPAddr {
	if a, ok := r.listener.Addr().(*net.UDPAddr); ok {
		return a
	}
	a, _ := net.ResolveUDPAddr("udp", r.listener.Addr().String())
	return a
}

func (r *peRouter) Close() {
	_ = r.listener.Close()
}

func (r *peRouter) acceptLoop() {
	for {
		c, err := r.listener.Accept()
		if err != nil {
			return
		}
		go r.handle(c.(*minecraft.Conn))
	}
}

func (r *peRouter) resolve(serverAddress string) (string, bool) {
	host := serverAddress
	if i := strings.LastIndex(host, ":"); i != -1 {
		host = host[:i]
	}
	b, ok := r.forced[strings.ToLower(host)]
	return b, ok
}

func (r *peRouter) handle(conn *minecraft.Conn) {
	addr := conn.ClientData().ServerAddress
	backend, ok := r.resolve(addr)
	if !ok {
		r.xl.Warnf("pe: no forced host for [%s], disconnecting", addr)
		_ = r.listener.Disconnect(conn, "No route for this address.")
		return
	}

	serverConn, err := minecraft.Dialer{
		ClientData:   conn.ClientData(),
		IdentityData: conn.IdentityData(),
	}.Dial("raknet", backend)
	if err != nil {
		r.xl.Warnf("pe: dial backend [%s] failed: %v", backend, err)
		_ = r.listener.Disconnect(conn, "Backend unavailable.")
		return
	}
	r.xl.Infof("pe: routing player [%s] -> backend [%s]", addr, backend)

	// Spawn both sides concurrently (standard gophertunnel proxy handshake).
	var g sync.WaitGroup
	g.Add(2)
	go func() {
		defer g.Done()
		_ = conn.StartGame(serverConn.GameData())
	}()
	go func() {
		defer g.Done()
		_ = serverConn.DoSpawn()
	}()
	g.Wait()

	var once sync.Once
	closeBoth := func() {
		once.Do(func() {
			_ = r.listener.Disconnect(conn, "Connection closed.")
			_ = serverConn.Close()
		})
	}
	// client -> backend
	go func() {
		defer closeBoth()
		for {
			pk, err := conn.ReadPacket()
			if err != nil {
				return
			}
			if err := serverConn.WritePacket(pk); err != nil {
				return
			}
		}
	}()
	// backend -> client
	go func() {
		defer closeBoth()
		for {
			pk, err := serverConn.ReadPacket()
			if err != nil {
				return
			}
			if err := conn.WritePacket(pk); err != nil {
				return
			}
		}
	}()
}

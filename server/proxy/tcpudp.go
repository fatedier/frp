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

package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/fatedier/golib/errors"
	libio "github.com/fatedier/golib/io"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/udp"
	"github.com/fatedier/frp/pkg/util/limit"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/server/metrics"
)

func init() {
	RegisterProxyFactory(reflect.TypeFor[*v1.TCPUDPProxyConfig](), NewTCPUDPProxy)
}

// TCPUDPProxy exposes BOTH a TCP and a UDP service on the SAME public RemotePort
// as a single proxy. It opens one TCP listener (relaying each connection like a
// plain tcp proxy) and one UDP listener (relaying datagrams like a plain udp
// proxy). Each work connection it pulls from the client pool is tagged with its
// protocol (TCP work conns carry the empty default; UDP work conns carry "udp")
// so the single client-side proxy routes every work conn to the right handler.
type TCPUDPProxy struct {
	*BaseProxy
	cfg *v1.TCPUDPProxyConfig

	realBindPort int

	// UDP side, mirroring server/proxy/udp.go.
	udpConn      *net.UDPConn
	workConn     net.Conn
	sendCh       chan *msg.UDPPacket
	readCh       chan *msg.UDPPacket
	checkCloseCh chan int
	isClosed     bool
}

func NewTCPUDPProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.TCPUDPProxyConfig)
	if !ok {
		return nil
	}
	baseProxy.usedPortsNum = 1
	return &TCPUDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

func (pxy *TCPUDPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl

	// Acquire the shared port on the TCP manager first, then pin the SAME number
	// on the UDP manager so both protocols share one RemotePort.
	pxy.realBindPort, err = pxy.rc.TCPPortManager.Acquire(pxy.name, pxy.cfg.RemotePort)
	if err != nil {
		return "", fmt.Errorf("acquire tcp port error: %v", err)
	}
	defer func() {
		if err != nil {
			pxy.rc.TCPPortManager.Release(pxy.realBindPort)
		}
	}()

	if _, err = pxy.rc.UDPPortManager.Acquire(pxy.name, pxy.realBindPort); err != nil {
		return "", fmt.Errorf("acquire udp port %d error: %v", pxy.realBindPort, err)
	}
	defer func() {
		if err != nil {
			pxy.rc.UDPPortManager.Release(pxy.realBindPort)
		}
	}()

	// --- TCP listener ---
	listener, errRet := net.Listen("tcp", net.JoinHostPort(pxy.serverCfg.ProxyBindAddr, strconv.Itoa(pxy.realBindPort)))
	if errRet != nil {
		err = errRet
		return
	}
	pxy.listeners = append(pxy.listeners, listener)
	pxy.startCommonTCPListenersHandler()
	xl.Infof("tcp+udp proxy listen tcp port [%d]", pxy.realBindPort)

	// --- UDP listener ---
	udpAddr, errRet := net.ResolveUDPAddr("udp", net.JoinHostPort(pxy.serverCfg.ProxyBindAddr, strconv.Itoa(pxy.realBindPort)))
	if errRet != nil {
		err = errRet
		return
	}
	udpConn, errRet := net.ListenUDP("udp", udpAddr)
	if errRet != nil {
		err = errRet
		xl.Warnf("listen udp port error: %v", err)
		return
	}
	pxy.udpConn = udpConn
	pxy.sendCh = make(chan *msg.UDPPacket, 1024)
	pxy.readCh = make(chan *msg.UDPPacket, 1024)
	pxy.checkCloseCh = make(chan int)
	pxy.runUDPRelay()
	xl.Infof("tcp+udp proxy listen udp port [%d]", pxy.realBindPort)

	pxy.cfg.RemotePort = pxy.realBindPort
	remoteAddr = fmt.Sprintf(":%d", pxy.realBindPort)
	return
}

// runUDPRelay wires the UDP listener to a single client work connection and keeps
// replacing that work connection when it dies. It is a direct adaptation of
// server/proxy/udp.go, except it requests work connections tagged with "udp".
func (pxy *TCPUDPProxy) runUDPRelay() {
	xl := pxy.xl

	workConnReaderFn := func(payloadConn *msg.Conn) {
		for {
			var (
				rawMsg msg.Message
				errRet error
			)
			_ = payloadConn.SetReadDeadline(time.Now().Add(time.Duration(60) * time.Second))
			if rawMsg, errRet = payloadConn.ReadMsg(); errRet != nil {
				xl.Warnf("read from workConn for udp error: %v", errRet)
				_ = payloadConn.Close()
				_ = errors.PanicToError(func() {
					pxy.checkCloseCh <- 1
				})
				return
			}
			if err := payloadConn.SetReadDeadline(time.Time{}); err != nil {
				xl.Warnf("set read deadline error: %v", err)
			}
			switch m := rawMsg.(type) {
			case *msg.Ping:
				continue
			case *msg.UDPPacket:
				if errRet := errors.PanicToError(func() {
					pxy.readCh <- m
					metrics.Server.AddTrafficOut(
						pxy.GetName(),
						pxy.GetConfigurer().GetBaseConfig().Type,
						int64(len(m.Content)),
					)
				}); errRet != nil {
					_ = payloadConn.Close()
					xl.Infof("reader goroutine for udp work connection closed")
					return
				}
			}
		}
	}

	workConnSenderFn := func(payloadConn *msg.Conn, ctx context.Context) {
		var errRet error
		for {
			select {
			case udpMsg, ok := <-pxy.sendCh:
				if !ok {
					xl.Infof("sender goroutine for udp work connection closed")
					return
				}
				if errRet = payloadConn.WriteMsg(udpMsg); errRet != nil {
					xl.Infof("sender goroutine for udp work connection closed: %v", errRet)
					_ = payloadConn.Close()
					return
				}
				metrics.Server.AddTrafficIn(
					pxy.GetName(),
					pxy.GetConfigurer().GetBaseConfig().Type,
					int64(len(udpMsg.Content)),
				)
				continue
			case <-ctx.Done():
				xl.Infof("sender goroutine for udp work connection closed")
				return
			}
		}
	}

	go func() {
		// Sleep a while for waiting control send the NewProxyResp to client.
		time.Sleep(500 * time.Millisecond)
		for {
			workConn, err := pxy.GetWorkConnFromPoolWithProtocol(nil, nil, "udp")
			if err != nil {
				time.Sleep(1 * time.Second)
				select {
				case _, ok := <-pxy.checkCloseCh:
					if !ok {
						return
					}
				default:
				}
				continue
			}
			if pxy.workConn != nil {
				pxy.workConn.Close()
			}

			var rwc io.ReadWriteCloser = workConn
			if pxy.cfg.Transport.UseEncryption {
				rwc, err = libio.WithEncryption(rwc, pxy.encryptionKey)
				if err != nil {
					xl.Errorf("create encryption stream error: %v", err)
					workConn.Close()
					continue
				}
			}
			if pxy.cfg.Transport.UseCompression {
				rwc = libio.WithCompression(rwc)
			}
			if pxy.GetLimiter() != nil {
				rwc = libio.WrapReadWriteCloser(limit.NewReader(rwc, pxy.GetLimiter()), limit.NewWriter(rwc, pxy.GetLimiter()), func() error {
					return rwc.Close()
				})
			}

			pxy.workConn = netpkg.WrapReadWriteCloserToConn(rwc, workConn)
			payloadConn := msg.NewConn(pxy.workConn, msg.NewReadWriter(pxy.workConn, pxy.wireProtocol))
			ctx, cancel := context.WithCancel(context.Background())
			go workConnReaderFn(payloadConn)
			go workConnSenderFn(payloadConn, ctx)
			_, ok := <-pxy.checkCloseCh
			cancel()
			if !ok {
				return
			}
		}
	}()

	go func() {
		udp.ForwardUserConn(pxy.udpConn, pxy.readCh, pxy.sendCh, int(pxy.serverCfg.UDPPacketSize))
		pxy.Close()
	}()
}

func (pxy *TCPUDPProxy) Close() {
	pxy.mu.Lock()
	defer pxy.mu.Unlock()
	if !pxy.isClosed {
		pxy.isClosed = true

		pxy.BaseProxy.Close()
		if pxy.workConn != nil {
			pxy.workConn.Close()
		}
		if pxy.udpConn != nil {
			pxy.udpConn.Close()
		}
		close(pxy.checkCloseCh)
		close(pxy.readCh)
		close(pxy.sendCh)
	}
	pxy.rc.TCPPortManager.Release(pxy.realBindPort)
	pxy.rc.UDPPortManager.Release(pxy.realBindPort)
}

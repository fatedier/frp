// Copyright 2019 fatedier, fatedier@gmail.com
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
	RegisterProxyFactory(reflect.TypeFor[*v1.UDPProxyConfig](), NewUDPProxy)
}

type UDPProxy struct {
	*BaseProxy
	cfg *v1.UDPProxyConfig

	realBindPort int

	// udpConn is the listener of udp packages
	udpConn *net.UDPConn

	// there are always only one workConn at the same time
	// get another one if it closed
	workConn net.Conn

	// sendCh is used for sending packages to workConn
	sendCh chan *msg.UDPPacket

	// readCh is used for reading packages from workConn
	readCh chan *msg.UDPPacket

	// checkCloseCh is used for watching if workConn is closed
	checkCloseCh chan int

	isClosed bool
}

func NewUDPProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.UDPProxyConfig)
	if !ok {
		return nil
	}
	baseProxy.usedPortsNum = 1
	// QUIC datagram relay bypasses the work-connection stream, so it can't
	// honor the per-proxy stream wrappers (encryption/compression).
	baseProxy.udpDatagramWanted = unwrapped.QUICDatagrams &&
		!unwrapped.Transport.UseEncryption && !unwrapped.Transport.UseCompression
	return &UDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

func (pxy *UDPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl
	pxy.realBindPort, err = pxy.rc.UDPPortManager.Acquire(pxy.name, pxy.cfg.RemotePort)
	if err != nil {
		return "", fmt.Errorf("acquire port %d error: %v", pxy.cfg.RemotePort, err)
	}
	defer func() {
		if err != nil {
			pxy.rc.UDPPortManager.Release(pxy.realBindPort)
		}
	}()

	remoteAddr = fmt.Sprintf(":%d", pxy.realBindPort)
	pxy.cfg.RemotePort = pxy.realBindPort
	addr, errRet := net.ResolveUDPAddr("udp", net.JoinHostPort(pxy.serverCfg.ProxyBindAddr, strconv.Itoa(pxy.realBindPort)))
	if errRet != nil {
		err = errRet
		return
	}
	udpConn, errRet := net.ListenUDP("udp", addr)
	if errRet != nil {
		err = errRet
		xl.Warnf("listen udp port error: %v", err)
		return
	}
	xl.Infof("udp proxy listen port [%d]", pxy.cfg.RemotePort)

	pxy.udpConn = udpConn
	pxy.sendCh = make(chan *msg.UDPPacket, 1024)
	pxy.readCh = make(chan *msg.UDPPacket, 1024)
	pxy.checkCloseCh = make(chan int)

	// read message from workConn, if it returns any error, notify proxy to start a new workConn
	workConnReaderFn := func(payloadConn *msg.Conn) {
		for {
			var (
				rawMsg msg.Message
				errRet error
			)
			xl.Tracef("loop waiting message from udp workConn")
			// client will send heartbeat in workConn for keeping alive
			_ = payloadConn.SetReadDeadline(time.Now().Add(time.Duration(60) * time.Second))
			if rawMsg, errRet = payloadConn.ReadMsg(); errRet != nil {
				xl.Warnf("read from workConn for udp error: %v", errRet)
				_ = payloadConn.Close()
				// notify proxy to start a new work connection
				// ignore error here, it means the proxy is closed
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
				xl.Tracef("udp work conn get ping message")
				continue
			case *msg.UDPPacket:
				if errRet := errors.PanicToError(func() {
					xl.Tracef("get udp message from workConn, len: %d", len(m.Content))
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

	// send message to workConn; when dgMux is set, packets go out as
	// unreliable QUIC datagrams and only oversized ones use the stream
	workConnSenderFn := func(payloadConn *msg.Conn, dgMux *udp.DatagramMux, ctx context.Context) {
		var errRet error
		for {
			select {
			case udpMsg, ok := <-pxy.sendCh:
				if !ok {
					xl.Infof("sender goroutine for udp work connection closed")
					return
				}
				if dgMux != nil {
					switch dgErr := dgMux.Send(pxy.GetName(), udpMsg.RemoteAddr, udpMsg.Content); dgErr {
					case nil:
						// Datagrams bypass the limiter-wrapped work
						// connection, so account for them here: charging
						// after the send (like limit.Reader does) delays
						// subsequent packets to hold the configured rate,
						// and never double-charges the stream fallback.
						if limiter := pxy.GetLimiter(); limiter != nil {
							_ = limiter.WaitN(ctx, len(udpMsg.Content))
						}
						metrics.Server.AddTrafficIn(
							pxy.GetName(),
							pxy.GetConfigurer().GetBaseConfig().Type,
							int64(len(udpMsg.Content)),
						)
						continue
					case udp.ErrDatagramTooLarge:
						// fall through to the stream for this packet
					default:
						xl.Infof("sender goroutine for udp work connection closed: %v", dgErr)
						_ = payloadConn.Close()
						return
					}
				}
				if errRet = payloadConn.WriteMsg(udpMsg); errRet != nil {
					xl.Infof("sender goroutine for udp work connection closed: %v", errRet)
					_ = payloadConn.Close()
					return
				}
				xl.Tracef("send message to udp workConn, len: %d", len(udpMsg.Content))
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
			workConn, err := pxy.GetWorkConnFromPool(nil, nil)
			if err != nil {
				time.Sleep(1 * time.Second)
				// check if proxy is closed
				select {
				case _, ok := <-pxy.checkCloseCh:
					if !ok {
						return
					}
				default:
				}
				continue
			}
			// close the old workConn and replace it with a new one
			if pxy.workConn != nil {
				pxy.workConn.Close()
			}

			// QUIC datagram lane: mirror of the udpDatagram decision made
			// in GetWorkConnFromPool for this same work connection
			var dgMux *udp.DatagramMux
			if pxy.udpDatagramWanted {
				if qc, ok := netpkg.QuicConnFrom(workConn); ok && qc.ConnectionState().SupportsDatagrams {
					dgMux = udp.DatagramMuxFor(qc)
					dgMux.Register(pxy.GetName(), func(remoteAddr *net.UDPAddr, payload []byte) {
						// The bandwidth limiter can only police (drop)
						// here, not shape: this callback runs on the quic
						// connection's shared receive loop, which must
						// never block.
						if limiter := pxy.GetLimiter(); limiter != nil && !limiter.AllowN(time.Now(), len(payload)) {
							return
						}
						m := udp.NewUDPPacket(payload, nil, remoteAddr)
						_ = errors.PanicToError(func() {
							select {
							case pxy.readCh <- m:
								metrics.Server.AddTrafficOut(
									pxy.GetName(),
									pxy.GetConfigurer().GetBaseConfig().Type,
									int64(len(m.Content)),
								)
							default:
							}
						})
					})
					xl.Infof("udp proxy relaying via quic datagrams")
				}
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
			// Plain UDP payload follows the negotiated wire protocol for message framing.
			payloadConn := msg.NewConn(pxy.workConn, msg.NewReadWriter(pxy.workConn, pxy.wireProtocol))
			ctx, cancel := context.WithCancel(context.Background())
			go workConnReaderFn(payloadConn)
			go workConnSenderFn(payloadConn, dgMux, ctx)
			_, ok := <-pxy.checkCloseCh
			cancel()
			if dgMux != nil {
				dgMux.Unregister(pxy.GetName())
			}
			if !ok {
				return
			}
		}
	}()

	// Read from user connections and send wrapped udp message to sendCh (forwarded by workConn).
	// Client will transfor udp message to local udp service and waiting for response for a while.
	// Response will be wrapped to be forwarded by work connection to server.
	// Close readCh and sendCh at the end.
	go func() {
		udp.ForwardUserConn(udpConn, pxy.readCh, pxy.sendCh, int(pxy.serverCfg.UDPPacketSize))
		pxy.Close()
	}()
	return remoteAddr, nil
}

func (pxy *UDPProxy) Close() {
	pxy.mu.Lock()
	defer pxy.mu.Unlock()
	if !pxy.isClosed {
		pxy.isClosed = true

		pxy.BaseProxy.Close()
		if pxy.workConn != nil {
			pxy.workConn.Close()
		}
		pxy.udpConn.Close()

		// all channels only closed here
		close(pxy.checkCloseCh)
		close(pxy.readCh)
		close(pxy.sendCh)
	}
	pxy.rc.UDPPortManager.Release(pxy.realBindPort)
}

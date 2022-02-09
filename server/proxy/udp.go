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
	"strconv"
	"time"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/udp"
	frpNet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/server/metrics"

	"github.com/fatedier/golib/errors"
	frpIo "github.com/fatedier/golib/io"
)

type UDPProxy struct {
	*BaseProxy
	cfg *config.UDPProxyConf

	realPort int

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

func (pxy *UDPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl
	pxy.realPort, err = pxy.rc.UDPPortManager.Acquire(pxy.name, pxy.cfg.RemotePort)
	if err != nil {
		return "", fmt.Errorf("acquire port %d error: %v", pxy.cfg.RemotePort, err)
	}
	defer func() {
		if err != nil {
			pxy.rc.UDPPortManager.Release(pxy.realPort)
		}
	}()

	remoteAddr = fmt.Sprintf(":%d", pxy.realPort)
	pxy.cfg.RemotePort = pxy.realPort
	addr, errRet := net.ResolveUDPAddr("udp", net.JoinHostPort(pxy.serverCfg.ProxyBindAddr, strconv.Itoa(pxy.realPort)))
	if errRet != nil {
		err = errRet
		return
	}
	udpConn, errRet := net.ListenUDP("udp", addr)
	if errRet != nil {
		err = errRet
		xl.Warn("listen udp port error: %v", err)
		return
	}
	xl.Info("udp proxy listen port [%d]", pxy.cfg.RemotePort)

	pxy.udpConn = udpConn
	pxy.sendCh = make(chan *msg.UDPPacket, 1024)
	pxy.readCh = make(chan *msg.UDPPacket, 1024)
	pxy.checkCloseCh = make(chan int)

	// read message from workConn, if it returns any error, notify proxy to start a new workConn
	workConnReaderFn := func(conn net.Conn) {
		for {
			var (
				rawMsg msg.Message
				errRet error
			)
			xl.Trace("loop waiting message from udp workConn")
			// client will send heartbeat in workConn for keeping alive
			conn.SetReadDeadline(time.Now().Add(time.Duration(60) * time.Second))
			if rawMsg, errRet = msg.ReadMsg(conn); errRet != nil {
				xl.Warn("read from workConn for udp error: %v", errRet)
				conn.Close()
				// notify proxy to start a new work connection
				// ignore error here, it means the proxy is closed
				errors.PanicToError(func() {
					pxy.checkCloseCh <- 1
				})
				return
			}
			conn.SetReadDeadline(time.Time{})
			switch m := rawMsg.(type) {
			case *msg.Ping:
				xl.Trace("udp work conn get ping message")
				continue
			case *msg.UDPPacket:
				if errRet := errors.PanicToError(func() {
					xl.Trace("get udp message from workConn: %s", m.Content)
					pxy.readCh <- m
					metrics.Server.AddTrafficOut(
						pxy.GetName(),
						pxy.GetConf().GetBaseInfo().ProxyType,
						int64(len(m.Content)),
					)
				}); errRet != nil {
					conn.Close()
					xl.Info("reader goroutine for udp work connection closed")
					return
				}
			}
		}
	}

	// send message to workConn
	workConnSenderFn := func(conn net.Conn, ctx context.Context) {
		var errRet error
		for {
			select {
			case udpMsg, ok := <-pxy.sendCh:
				if !ok {
					xl.Info("sender goroutine for udp work connection closed")
					return
				}
				if errRet = msg.WriteMsg(conn, udpMsg); errRet != nil {
					xl.Info("sender goroutine for udp work connection closed: %v", errRet)
					conn.Close()
					return
				}
				xl.Trace("send message to udp workConn: %s", udpMsg.Content)
				metrics.Server.AddTrafficIn(
					pxy.GetName(),
					pxy.GetConf().GetBaseInfo().ProxyType,
					int64(len(udpMsg.Content)),
				)
				continue
			case <-ctx.Done():
				xl.Info("sender goroutine for udp work connection closed")
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

			var rwc io.ReadWriteCloser = workConn
			if pxy.cfg.UseEncryption {
				rwc, err = frpIo.WithEncryption(rwc, []byte(pxy.serverCfg.Token))
				if err != nil {
					xl.Error("create encryption stream error: %v", err)
					workConn.Close()
					continue
				}
			}
			if pxy.cfg.UseCompression {
				rwc = frpIo.WithCompression(rwc)
			}

			pxy.workConn = frpNet.WrapReadWriteCloserToConn(rwc, workConn)
			ctx, cancel := context.WithCancel(context.Background())
			go workConnReaderFn(pxy.workConn)
			go workConnSenderFn(pxy.workConn, ctx)
			_, ok := <-pxy.checkCloseCh
			cancel()
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

func (pxy *UDPProxy) GetConf() config.ProxyConf {
	return pxy.cfg
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
	pxy.rc.UDPPortManager.Release(pxy.realPort)
}

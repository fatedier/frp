// Copyright 2023 The frp Authors
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
	libio "github.com/fatedier/golib/io"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/udp"
	"github.com/fatedier/frp/pkg/util/limit"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

func init() {
	RegisterProxyFactory(reflect.TypeOf(&v1.SUDPProxyConfig{}), NewSUDPProxy)
}

type SUDPProxy struct {
	*BaseProxy

	cfg *v1.SUDPProxyConfig

	localAddr *net.UDPAddr

	closeCh chan struct{}
}

func NewSUDPProxy(baseProxy *BaseProxy, cfg v1.ProxyConfigurer) Proxy {
	unwrapped, ok := cfg.(*v1.SUDPProxyConfig)
	if !ok {
		return nil
	}
	return &SUDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
		closeCh:   make(chan struct{}),
	}
}

func (pxy *SUDPProxy) Run() (err error) {
	pxy.localAddr, err = net.ResolveUDPAddr("udp", net.JoinHostPort(pxy.cfg.LocalIP, strconv.Itoa(pxy.cfg.LocalPort)))
	if err != nil {
		return
	}
	return
}

func (pxy *SUDPProxy) Close() {
	pxy.mu.Lock()
	defer pxy.mu.Unlock()
	select {
	case <-pxy.closeCh:
		return
	default:
		close(pxy.closeCh)
	}
}

func (pxy *SUDPProxy) InWorkConn(conn net.Conn, _ *msg.StartWorkConn) {
	xl := pxy.xl
	xl.Infof("incoming a new work connection for sudp proxy, %s", conn.RemoteAddr().String())

	var rwc io.ReadWriteCloser = conn
	var err error
	if pxy.limiter != nil {
		rwc = libio.WrapReadWriteCloser(limit.NewReader(conn, pxy.limiter), limit.NewWriter(conn, pxy.limiter), func() error {
			return conn.Close()
		})
	}
	if pxy.cfg.Transport.UseEncryption {
		rwc, err = libio.WithEncryption(rwc, []byte(pxy.clientCfg.Auth.Token))
		if err != nil {
			conn.Close()
			xl.Errorf("create encryption stream error: %v", err)
			return
		}
	}
	if pxy.cfg.Transport.UseCompression {
		rwc = libio.WithCompression(rwc)
	}
	conn = netpkg.WrapReadWriteCloserToConn(rwc, conn)

	workConn := conn
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

	// udp service <- frpc <- frps <- frpc visitor <- user
	workConnReaderFn := func(conn net.Conn, readCh chan *msg.UDPPacket) {
		defer closeFn()

		for {
			// first to check sudp proxy is closed or not
			select {
			case <-pxy.closeCh:
				xl.Tracef("frpc sudp proxy is closed")
				return
			default:
			}

			var udpMsg msg.UDPPacket
			if errRet := msg.ReadMsgInto(conn, &udpMsg); errRet != nil {
				xl.Warnf("read from workConn for sudp error: %v", errRet)
				return
			}

			if errRet := errors.PanicToError(func() {
				readCh <- &udpMsg
			}); errRet != nil {
				xl.Warnf("reader goroutine for sudp work connection closed: %v", errRet)
				return
			}
		}
	}

	// udp service -> frpc -> frps -> frpc visitor -> user
	workConnSenderFn := func(conn net.Conn, sendCh chan msg.Message) {
		defer func() {
			closeFn()
			xl.Infof("writer goroutine for sudp work connection closed")
		}()

		var errRet error
		for rawMsg := range sendCh {
			switch m := rawMsg.(type) {
			case *msg.UDPPacket:
				xl.Tracef("frpc send udp package to frpc visitor, [udp local: %v, remote: %v], [tcp work conn local: %v, remote: %v]",
					m.LocalAddr.String(), m.RemoteAddr.String(), conn.LocalAddr().String(), conn.RemoteAddr().String())
			case *msg.Ping:
				xl.Tracef("frpc send ping message to frpc visitor")
			}

			if errRet = msg.WriteMsg(conn, rawMsg); errRet != nil {
				xl.Errorf("sudp work write error: %v", errRet)
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
					xl.Warnf("heartbeat goroutine for sudp work connection closed")
					return
				}
			case <-pxy.closeCh:
				xl.Tracef("frpc sudp proxy is closed")
				return
			}
		}
	}

	go workConnSenderFn(workConn, sendCh)
	go workConnReaderFn(workConn, readCh)
	go heartbeatFn(sendCh)

	udp.Forwarder(pxy.localAddr, readCh, sendCh, int(pxy.clientCfg.UDPPacketSize))
}

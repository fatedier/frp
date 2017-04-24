// Copyright 2017 fatedier, fatedier@gmail.com
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

package client

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/models/proto/tcp"
	"github.com/fatedier/frp/models/proto/udp"
	"github.com/fatedier/frp/utils/errors"
	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"
)

// Proxy defines how to work for different proxy type.
type Proxy interface {
	Run() error

	// InWorkConn accept work connections registered to server.
	InWorkConn(conn frpNet.Conn)
	Close()
	log.Logger
}

func NewProxy(ctl *Control, pxyConf config.ProxyConf) (pxy Proxy) {
	baseProxy := BaseProxy{
		ctl:    ctl,
		Logger: log.NewPrefixLogger(pxyConf.GetName()),
	}
	switch cfg := pxyConf.(type) {
	case *config.TcpProxyConf:
		pxy = &TcpProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}
	case *config.UdpProxyConf:
		pxy = &UdpProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}
	case *config.HttpProxyConf:
		pxy = &HttpProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}
	case *config.HttpsProxyConf:
		pxy = &HttpsProxy{
			BaseProxy: baseProxy,
			cfg:       cfg,
		}
	}
	return
}

type BaseProxy struct {
	ctl    *Control
	closed bool
	mu     sync.RWMutex
	log.Logger
}

// TCP
type TcpProxy struct {
	BaseProxy

	cfg *config.TcpProxyConf
}

func (pxy *TcpProxy) Run() (err error) {
	return
}

func (pxy *TcpProxy) Close() {
}

func (pxy *TcpProxy) InWorkConn(conn frpNet.Conn) {
	defer conn.Close()
	HandleTcpWorkConnection(&pxy.cfg.LocalSvrConf, &pxy.cfg.BaseProxyConf, conn)
}

// HTTP
type HttpProxy struct {
	BaseProxy

	cfg *config.HttpProxyConf
}

func (pxy *HttpProxy) Run() (err error) {
	return
}

func (pxy *HttpProxy) Close() {
}

func (pxy *HttpProxy) InWorkConn(conn frpNet.Conn) {
	defer conn.Close()
	HandleTcpWorkConnection(&pxy.cfg.LocalSvrConf, &pxy.cfg.BaseProxyConf, conn)
}

// HTTPS
type HttpsProxy struct {
	BaseProxy

	cfg *config.HttpsProxyConf
}

func (pxy *HttpsProxy) Run() (err error) {
	return
}

func (pxy *HttpsProxy) Close() {
}

func (pxy *HttpsProxy) InWorkConn(conn frpNet.Conn) {
	defer conn.Close()
	HandleTcpWorkConnection(&pxy.cfg.LocalSvrConf, &pxy.cfg.BaseProxyConf, conn)
}

// UDP
type UdpProxy struct {
	BaseProxy

	cfg *config.UdpProxyConf

	localAddr *net.UDPAddr
	readCh    chan *msg.UdpPacket
	sendCh    chan *msg.UdpPacket
	workConn  frpNet.Conn
}

func (pxy *UdpProxy) Run() (err error) {
	pxy.localAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", pxy.cfg.LocalIp, pxy.cfg.LocalPort))
	if err != nil {
		return
	}
	return
}

func (pxy *UdpProxy) Close() {
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

func (pxy *UdpProxy) InWorkConn(conn frpNet.Conn) {
	pxy.Info("incoming a new work connection for udp proxy")
	// close resources releated with old workConn
	pxy.Close()

	pxy.mu.Lock()
	pxy.workConn = conn
	pxy.readCh = make(chan *msg.UdpPacket, 64)
	pxy.sendCh = make(chan *msg.UdpPacket, 64)
	pxy.closed = false
	pxy.mu.Unlock()

	workConnReaderFn := func(conn net.Conn) {
		for {
			var udpMsg msg.UdpPacket
			if errRet := msg.ReadMsgInto(conn, &udpMsg); errRet != nil {
				pxy.Warn("read from workConn for udp error: %v", errRet)
				return
			}
			if errRet := errors.PanicToError(func() {
				pxy.Trace("get udp package from workConn: %s", udpMsg.Content)
				pxy.readCh <- &udpMsg
			}); errRet != nil {
				pxy.Info("reader goroutine for udp work connection closed: %v", errRet)
				return
			}
		}
	}
	workConnSenderFn := func(conn net.Conn) {
		var errRet error
		for udpMsg := range pxy.sendCh {
			pxy.Trace("send udp package to workConn: %s", udpMsg.Content)
			if errRet = msg.WriteMsg(conn, udpMsg); errRet != nil {
				pxy.Info("sender goroutine for udp work connection closed")
				return
			}
		}
	}

	go workConnSenderFn(pxy.workConn)
	go workConnReaderFn(pxy.workConn)
	udp.Forwarder(pxy.localAddr, pxy.readCh, pxy.sendCh)
}

// Common handler for tcp work connections.
func HandleTcpWorkConnection(localInfo *config.LocalSvrConf, baseInfo *config.BaseProxyConf, workConn frpNet.Conn) {
	localConn, err := frpNet.ConnectTcpServer(fmt.Sprintf("%s:%d", localInfo.LocalIp, localInfo.LocalPort))
	if err != nil {
		workConn.Error("connect to local service [%s:%d] error: %v", localInfo.LocalIp, localInfo.LocalPort, err)
		return
	}

	var remote io.ReadWriteCloser
	remote = workConn
	if baseInfo.UseEncryption {
		remote, err = tcp.WithEncryption(remote, []byte(config.ClientCommonCfg.PrivilegeToken))
		if err != nil {
			workConn.Error("create encryption stream error: %v", err)
			return
		}
	}
	if baseInfo.UseCompression {
		remote = tcp.WithCompression(remote)
	}
	workConn.Debug("join connections, localConn(l[%s] r[%s]) workConn(l[%s] r[%s])", localConn.LocalAddr().String(),
		localConn.RemoteAddr().String(), workConn.LocalAddr().String(), workConn.RemoteAddr().String())
	tcp.Join(localConn, remote)
	workConn.Debug("join connections closed")
}

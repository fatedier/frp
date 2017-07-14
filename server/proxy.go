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

package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/models/proto/udp"
	"github.com/fatedier/frp/utils/errors"
	frpIo "github.com/fatedier/frp/utils/io"
	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/vhost"
)

type Proxy interface {
	Run() error
	GetControl() *Control
	GetName() string
	GetConf() config.ProxyConf
	GetWorkConnFromPool() (workConn frpNet.Conn, err error)
	Close()
	log.Logger
}

type BaseProxy struct {
	name      string
	ctl       *Control
	listeners []frpNet.Listener
	mu        sync.RWMutex
	log.Logger
}

func (pxy *BaseProxy) GetName() string {
	return pxy.name
}

func (pxy *BaseProxy) GetControl() *Control {
	return pxy.ctl
}

func (pxy *BaseProxy) Close() {
	pxy.Info("proxy closing")
	for _, l := range pxy.listeners {
		l.Close()
	}
}

func (pxy *BaseProxy) GetWorkConnFromPool() (workConn frpNet.Conn, err error) {
	ctl := pxy.GetControl()
	// try all connections from the pool
	for i := 0; i < ctl.poolCount+1; i++ {
		if workConn, err = ctl.GetWorkConn(); err != nil {
			pxy.Warn("failed to get work connection: %v", err)
			return
		}
		pxy.Info("get a new work connection: [%s]", workConn.RemoteAddr().String())
		workConn.AddLogPrefix(pxy.GetName())

		err := msg.WriteMsg(workConn, &msg.StartWorkConn{
			ProxyName: pxy.GetName(),
		})
		if err != nil {
			workConn.Warn("failed to send message to work connection from pool: %v, times: %d", err, i)
			workConn.Close()
		} else {
			break
		}
	}

	if err != nil {
		pxy.Error("try to get work connection failed in the end")
		return
	}
	return
}

// startListenHandler start a goroutine handler for each listener.
// p: p will just be passed to handler(Proxy, frpNet.Conn).
// handler: each proxy type can set different handler function to deal with connections accepted from listeners.
func (pxy *BaseProxy) startListenHandler(p Proxy, handler func(Proxy, frpNet.Conn)) {
	for _, listener := range pxy.listeners {
		go func(l frpNet.Listener) {
			for {
				// block
				// if listener is closed, err returned
				c, err := l.Accept()
				if err != nil {
					pxy.Info("listener is closed")
					return
				}
				pxy.Debug("get a user connection [%s]", c.RemoteAddr().String())
				go handler(p, c)
			}
		}(listener)
	}
}

func NewProxy(ctl *Control, pxyConf config.ProxyConf) (pxy Proxy, err error) {
	basePxy := BaseProxy{
		name:      pxyConf.GetName(),
		ctl:       ctl,
		listeners: make([]frpNet.Listener, 0),
		Logger:    log.NewPrefixLogger(ctl.runId),
	}
	switch cfg := pxyConf.(type) {
	case *config.TcpProxyConf:
		pxy = &TcpProxy{
			BaseProxy: basePxy,
			cfg:       cfg,
		}
	case *config.HttpProxyConf:
		pxy = &HttpProxy{
			BaseProxy: basePxy,
			cfg:       cfg,
		}
	case *config.HttpsProxyConf:
		pxy = &HttpsProxy{
			BaseProxy: basePxy,
			cfg:       cfg,
		}
	case *config.UdpProxyConf:
		pxy = &UdpProxy{
			BaseProxy: basePxy,
			cfg:       cfg,
		}
	case *config.StcpProxyConf:
		pxy = &StcpProxy{
			BaseProxy: basePxy,
			cfg:       cfg,
		}
	default:
		return pxy, fmt.Errorf("proxy type not support")
	}
	pxy.AddLogPrefix(pxy.GetName())
	return
}

type TcpProxy struct {
	BaseProxy
	cfg *config.TcpProxyConf
}

func (pxy *TcpProxy) Run() error {
	listener, err := frpNet.ListenTcp(config.ServerCommonCfg.ProxyBindAddr, pxy.cfg.RemotePort)
	if err != nil {
		return err
	}
	listener.AddLogPrefix(pxy.name)
	pxy.listeners = append(pxy.listeners, listener)
	pxy.Info("tcp proxy listen port [%d]", pxy.cfg.RemotePort)

	pxy.startListenHandler(pxy, HandleUserTcpConnection)
	return nil
}

func (pxy *TcpProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *TcpProxy) Close() {
	pxy.BaseProxy.Close()
}

type HttpProxy struct {
	BaseProxy
	cfg *config.HttpProxyConf
}

func (pxy *HttpProxy) Run() (err error) {
	routeConfig := &vhost.VhostRouteConfig{
		RewriteHost: pxy.cfg.HostHeaderRewrite,
		Username:    pxy.cfg.HttpUser,
		Password:    pxy.cfg.HttpPwd,
	}

	locations := pxy.cfg.Locations
	if len(locations) == 0 {
		locations = []string{""}
	}
	for _, domain := range pxy.cfg.CustomDomains {
		routeConfig.Domain = domain
		for _, location := range locations {
			routeConfig.Location = location
			l, err := pxy.ctl.svr.VhostHttpMuxer.Listen(routeConfig)
			if err != nil {
				return err
			}
			l.AddLogPrefix(pxy.name)
			pxy.Info("http proxy listen for host [%s] location [%s]", routeConfig.Domain, routeConfig.Location)
			pxy.listeners = append(pxy.listeners, l)
		}
	}

	if pxy.cfg.SubDomain != "" {
		routeConfig.Domain = pxy.cfg.SubDomain + "." + config.ServerCommonCfg.SubDomainHost
		for _, location := range locations {
			routeConfig.Location = location
			l, err := pxy.ctl.svr.VhostHttpMuxer.Listen(routeConfig)
			if err != nil {
				return err
			}
			l.AddLogPrefix(pxy.name)
			pxy.Info("http proxy listen for host [%s] location [%s]", routeConfig.Domain, routeConfig.Location)
			pxy.listeners = append(pxy.listeners, l)
		}
	}

	pxy.startListenHandler(pxy, HandleUserTcpConnection)
	return
}

func (pxy *HttpProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *HttpProxy) Close() {
	pxy.BaseProxy.Close()
}

type HttpsProxy struct {
	BaseProxy
	cfg *config.HttpsProxyConf
}

func (pxy *HttpsProxy) Run() (err error) {
	routeConfig := &vhost.VhostRouteConfig{}

	for _, domain := range pxy.cfg.CustomDomains {
		routeConfig.Domain = domain
		l, err := pxy.ctl.svr.VhostHttpsMuxer.Listen(routeConfig)
		if err != nil {
			return err
		}
		l.AddLogPrefix(pxy.name)
		pxy.Info("https proxy listen for host [%s]", routeConfig.Domain)
		pxy.listeners = append(pxy.listeners, l)
	}

	if pxy.cfg.SubDomain != "" {
		routeConfig.Domain = pxy.cfg.SubDomain + "." + config.ServerCommonCfg.SubDomainHost
		l, err := pxy.ctl.svr.VhostHttpsMuxer.Listen(routeConfig)
		if err != nil {
			return err
		}
		l.AddLogPrefix(pxy.name)
		pxy.Info("https proxy listen for host [%s]", routeConfig.Domain)
		pxy.listeners = append(pxy.listeners, l)
	}

	pxy.startListenHandler(pxy, HandleUserTcpConnection)
	return
}

func (pxy *HttpsProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *HttpsProxy) Close() {
	pxy.BaseProxy.Close()
}

type StcpProxy struct {
	BaseProxy
	cfg *config.StcpProxyConf
}

func (pxy *StcpProxy) Run() error {
	listener, err := pxy.ctl.svr.vistorManager.Listen(pxy.GetName(), pxy.cfg.Sk)
	if err != nil {
		return err
	}
	listener.AddLogPrefix(pxy.name)
	pxy.listeners = append(pxy.listeners, listener)
	pxy.Info("stcp proxy custom listen success")

	pxy.startListenHandler(pxy, HandleUserTcpConnection)
	return nil
}

func (pxy *StcpProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *StcpProxy) Close() {
	pxy.BaseProxy.Close()
	pxy.ctl.svr.vistorManager.CloseListener(pxy.GetName())
}

type UdpProxy struct {
	BaseProxy
	cfg *config.UdpProxyConf

	// udpConn is the listener of udp packages
	udpConn *net.UDPConn

	// there are always only one workConn at the same time
	// get another one if it closed
	workConn net.Conn

	// sendCh is used for sending packages to workConn
	sendCh chan *msg.UdpPacket

	// readCh is used for reading packages from workConn
	readCh chan *msg.UdpPacket

	// checkCloseCh is used for watching if workConn is closed
	checkCloseCh chan int

	isClosed bool
}

func (pxy *UdpProxy) Run() (err error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.ServerCommonCfg.ProxyBindAddr, pxy.cfg.RemotePort))
	if err != nil {
		return err
	}
	udpConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		pxy.Warn("listen udp port error: %v", err)
		return err
	}
	pxy.Info("udp proxy listen port [%d]", pxy.cfg.RemotePort)

	pxy.udpConn = udpConn
	pxy.sendCh = make(chan *msg.UdpPacket, 1024)
	pxy.readCh = make(chan *msg.UdpPacket, 1024)
	pxy.checkCloseCh = make(chan int)

	// read message from workConn, if it returns any error, notify proxy to start a new workConn
	workConnReaderFn := func(conn net.Conn) {
		for {
			var (
				rawMsg msg.Message
				errRet error
			)
			pxy.Trace("loop waiting message from udp workConn")
			// client will send heartbeat in workConn for keeping alive
			conn.SetReadDeadline(time.Now().Add(time.Duration(60) * time.Second))
			if rawMsg, errRet = msg.ReadMsg(conn); errRet != nil {
				pxy.Warn("read from workConn for udp error: %v", errRet)
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
				pxy.Trace("udp work conn get ping message")
				continue
			case *msg.UdpPacket:
				if errRet := errors.PanicToError(func() {
					pxy.Trace("get udp message from workConn: %s", m.Content)
					pxy.readCh <- m
					StatsAddTrafficOut(pxy.GetName(), int64(len(m.Content)))
				}); errRet != nil {
					conn.Close()
					pxy.Info("reader goroutine for udp work connection closed")
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
					pxy.Info("sender goroutine for udp work connection closed")
					return
				}
				if errRet = msg.WriteMsg(conn, udpMsg); errRet != nil {
					pxy.Info("sender goroutine for udp work connection closed: %v", errRet)
					conn.Close()
					return
				} else {
					pxy.Trace("send message to udp workConn: %s", udpMsg.Content)
					StatsAddTrafficIn(pxy.GetName(), int64(len(udpMsg.Content)))
					continue
				}
			case <-ctx.Done():
				pxy.Info("sender goroutine for udp work connection closed")
				return
			}
		}
	}

	go func() {
		// Sleep a while for waiting control send the NewProxyResp to client.
		time.Sleep(500 * time.Millisecond)
		for {
			workConn, err := pxy.GetWorkConnFromPool()
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
			// close the old workConn and replac it with a new one
			if pxy.workConn != nil {
				pxy.workConn.Close()
			}
			pxy.workConn = workConn
			ctx, cancel := context.WithCancel(context.Background())
			go workConnReaderFn(workConn)
			go workConnSenderFn(workConn, ctx)
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
		udp.ForwardUserConn(udpConn, pxy.readCh, pxy.sendCh)
		pxy.Close()
	}()
	return nil
}

func (pxy *UdpProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *UdpProxy) Close() {
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
}

// HandleUserTcpConnection is used for incoming tcp user connections.
// It can be used for tcp, http, https type.
func HandleUserTcpConnection(pxy Proxy, userConn frpNet.Conn) {
	defer userConn.Close()

	// try all connections from the pool
	workConn, err := pxy.GetWorkConnFromPool()
	if err != nil {
		return
	}
	defer workConn.Close()

	var local io.ReadWriteCloser = workConn
	cfg := pxy.GetConf().GetBaseInfo()
	if cfg.UseEncryption {
		local, err = frpIo.WithEncryption(local, []byte(config.ServerCommonCfg.PrivilegeToken))
		if err != nil {
			pxy.Error("create encryption stream error: %v", err)
			return
		}
	}
	if cfg.UseCompression {
		local = frpIo.WithCompression(local)
	}
	pxy.Debug("join connections, workConn(l[%s] r[%s]) userConn(l[%s] r[%s])", workConn.LocalAddr().String(),
		workConn.RemoteAddr().String(), userConn.LocalAddr().String(), userConn.RemoteAddr().String())

	StatsOpenConnection(pxy.GetName())
	inCount, outCount := frpIo.Join(local, userConn)
	StatsCloseConnection(pxy.GetName())
	StatsAddTrafficIn(pxy.GetName(), inCount)
	StatsAddTrafficOut(pxy.GetName(), outCount)
	pxy.Debug("join connections closed")
}

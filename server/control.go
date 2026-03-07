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
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/auth"
	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	pkgerr "github.com/fatedier/frp/pkg/errors"
	"github.com/fatedier/frp/pkg/msg"
	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/pkg/transport"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/frp/pkg/util/wait"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/server/controller"
	"github.com/fatedier/frp/server/metrics"
	"github.com/fatedier/frp/server/proxy"
	"github.com/fatedier/frp/server/registry"
)

type ControlManager struct {
	// controls indexed by run id
	ctlsByRunID map[string]*Control

	mu sync.RWMutex
}

func NewControlManager() *ControlManager {
	return &ControlManager{
		ctlsByRunID: make(map[string]*Control),
	}
}

func (cm *ControlManager) Add(runID string, ctl *Control) (old *Control) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var ok bool
	old, ok = cm.ctlsByRunID[runID]
	if ok {
		old.Replaced(ctl)
	}
	cm.ctlsByRunID[runID] = ctl
	return
}

// we should make sure if it's the same control to prevent delete a new one
func (cm *ControlManager) Del(runID string, ctl *Control) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if c, ok := cm.ctlsByRunID[runID]; ok && c == ctl {
		delete(cm.ctlsByRunID, runID)
	}
}

func (cm *ControlManager) GetByID(runID string) (ctl *Control, ok bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	ctl, ok = cm.ctlsByRunID[runID]
	return
}

func (cm *ControlManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for _, ctl := range cm.ctlsByRunID {
		ctl.Close()
	}
	cm.ctlsByRunID = make(map[string]*Control)
	return nil
}

// SessionContext encapsulates the input parameters for creating a new Control.
type SessionContext struct {
	// all resource managers and controllers
	RC *controller.ResourceController
	// proxy manager
	PxyManager *proxy.Manager
	// plugin manager
	PluginManager *plugin.Manager
	// verifies authentication based on selected method
	AuthVerifier auth.Verifier
	// key used for connection encryption
	EncryptionKey []byte
	// control connection
	Conn net.Conn
	// indicates whether the connection is encrypted
	ConnEncrypted bool
	// login message
	LoginMsg *msg.Login
	// server configuration
	ServerCfg *v1.ServerConfig
	// client registry
	ClientRegistry *registry.ClientRegistry
}

type Control struct {
	// session context
	sessionCtx *SessionContext

	// other components can use this to communicate with client
	msgTransporter transport.MessageTransporter

	// msgDispatcher is a wrapper for control connection.
	// It provides a channel for sending messages, and you can register handlers to process messages based on their respective types.
	msgDispatcher *msg.Dispatcher

	// work connections
	workConnCh chan net.Conn

	// proxies in one client
	proxies map[string]proxy.Proxy

	// pool count
	poolCount int

	// ports used, for limitations
	portsUsedNum int

	// last time got the Ping message
	lastPing atomic.Value

	// A new run id will be generated when a new client login.
	// If run id got from login message has same run id, it means it's the same client, so we can
	// replace old controller instantly.
	runID string

	mu sync.RWMutex

	xl     *xlog.Logger
	ctx    context.Context
	doneCh chan struct{}
}

func NewControl(ctx context.Context, sessionCtx *SessionContext) (*Control, error) {
	poolCount := min(sessionCtx.LoginMsg.PoolCount, int(sessionCtx.ServerCfg.Transport.MaxPoolCount))
	ctl := &Control{
		sessionCtx:   sessionCtx,
		workConnCh:   make(chan net.Conn, poolCount+10),
		proxies:      make(map[string]proxy.Proxy),
		poolCount:    poolCount,
		portsUsedNum: 0,
		runID:        sessionCtx.LoginMsg.RunID,
		xl:           xlog.FromContextSafe(ctx),
		ctx:          ctx,
		doneCh:       make(chan struct{}),
	}
	ctl.lastPing.Store(time.Now())

	if sessionCtx.ConnEncrypted {
		cryptoRW, err := netpkg.NewCryptoReadWriter(sessionCtx.Conn, sessionCtx.EncryptionKey)
		if err != nil {
			return nil, err
		}
		ctl.msgDispatcher = msg.NewDispatcher(cryptoRW)
	} else {
		ctl.msgDispatcher = msg.NewDispatcher(sessionCtx.Conn)
	}
	ctl.registerMsgHandlers()
	ctl.msgTransporter = transport.NewMessageTransporter(ctl.msgDispatcher)
	return ctl, nil
}

// Start send a login success message to client and start working.
func (ctl *Control) Start() {
	loginRespMsg := &msg.LoginResp{
		Version: version.Full(),
		RunID:   ctl.runID,
		Error:   "",
	}
	_ = msg.WriteMsg(ctl.sessionCtx.Conn, loginRespMsg)

	go func() {
		for i := 0; i < ctl.poolCount; i++ {
			// ignore error here, that means that this control is closed
			_ = ctl.msgDispatcher.Send(&msg.ReqWorkConn{})
		}
	}()
	go ctl.worker()
}

func (ctl *Control) Close() error {
	ctl.sessionCtx.Conn.Close()
	return nil
}

func (ctl *Control) Replaced(newCtl *Control) {
	xl := ctl.xl
	xl.Infof("replaced by client [%s]", newCtl.runID)
	ctl.runID = ""
	ctl.sessionCtx.Conn.Close()
}

func (ctl *Control) RegisterWorkConn(conn net.Conn) error {
	xl := ctl.xl
	defer func() {
		if err := recover(); err != nil {
			xl.Errorf("panic error: %v", err)
			xl.Errorf(string(debug.Stack()))
		}
	}()

	select {
	case ctl.workConnCh <- conn:
		xl.Debugf("new work connection registered")
		return nil
	default:
		xl.Debugf("work connection pool is full, discarding")
		return fmt.Errorf("work connection pool is full, discarding")
	}
}

// When frps get one user connection, we get one work connection from the pool and return it.
// If no workConn available in the pool, send message to frpc to get one or more
// and wait until it is available.
// return an error if wait timeout
func (ctl *Control) GetWorkConn() (workConn net.Conn, err error) {
	xl := ctl.xl
	defer func() {
		if err := recover(); err != nil {
			xl.Errorf("panic error: %v", err)
			xl.Errorf(string(debug.Stack()))
		}
	}()

	var ok bool
	// get a work connection from the pool
	select {
	case workConn, ok = <-ctl.workConnCh:
		if !ok {
			err = pkgerr.ErrCtlClosed
			return
		}
		xl.Debugf("get work connection from pool")
	default:
		// no work connections available in the poll, send message to frpc to get more
		if err := ctl.msgDispatcher.Send(&msg.ReqWorkConn{}); err != nil {
			return nil, fmt.Errorf("control is already closed")
		}

		select {
		case workConn, ok = <-ctl.workConnCh:
			if !ok {
				err = pkgerr.ErrCtlClosed
				xl.Warnf("no work connections available, %v", err)
				return
			}

		case <-time.After(time.Duration(ctl.sessionCtx.ServerCfg.UserConnTimeout) * time.Second):
			err = fmt.Errorf("timeout trying to get work connection")
			xl.Warnf("%v", err)
			return
		}
	}

	// When we get a work connection from pool, replace it with a new one.
	_ = ctl.msgDispatcher.Send(&msg.ReqWorkConn{})
	return
}

func (ctl *Control) heartbeatWorker() {
	if ctl.sessionCtx.ServerCfg.Transport.HeartbeatTimeout <= 0 {
		return
	}

	xl := ctl.xl
	go wait.Until(func() {
		if time.Since(ctl.lastPing.Load().(time.Time)) > time.Duration(ctl.sessionCtx.ServerCfg.Transport.HeartbeatTimeout)*time.Second {
			xl.Warnf("heartbeat timeout")
			ctl.sessionCtx.Conn.Close()
			return
		}
	}, time.Second, ctl.doneCh)
}

// block until Control closed
func (ctl *Control) WaitClosed() {
	<-ctl.doneCh
}

func (ctl *Control) loginUserInfo() plugin.UserInfo {
	return plugin.UserInfo{
		User:  ctl.sessionCtx.LoginMsg.User,
		Metas: ctl.sessionCtx.LoginMsg.Metas,
		RunID: ctl.sessionCtx.LoginMsg.RunID,
	}
}

func (ctl *Control) closeProxy(pxy proxy.Proxy) {
	pxy.Close()
	ctl.sessionCtx.PxyManager.Del(pxy.GetName())
	metrics.Server.CloseProxy(pxy.GetName(), pxy.GetConfigurer().GetBaseConfig().Type)

	notifyContent := &plugin.CloseProxyContent{
		User: ctl.loginUserInfo(),
		CloseProxy: msg.CloseProxy{
			ProxyName: pxy.GetName(),
		},
	}
	go func() {
		_ = ctl.sessionCtx.PluginManager.CloseProxy(notifyContent)
	}()
}

func (ctl *Control) worker() {
	xl := ctl.xl

	go ctl.heartbeatWorker()
	go ctl.msgDispatcher.Run()

	<-ctl.msgDispatcher.Done()
	ctl.sessionCtx.Conn.Close()

	ctl.mu.Lock()
	close(ctl.workConnCh)
	for workConn := range ctl.workConnCh {
		workConn.Close()
	}
	proxies := ctl.proxies
	ctl.proxies = make(map[string]proxy.Proxy)
	ctl.mu.Unlock()

	for _, pxy := range proxies {
		ctl.closeProxy(pxy)
	}

	metrics.Server.CloseClient()
	ctl.sessionCtx.ClientRegistry.MarkOfflineByRunID(ctl.runID)
	xl.Infof("client exit success")
	close(ctl.doneCh)
}

func (ctl *Control) registerMsgHandlers() {
	ctl.msgDispatcher.RegisterHandler(&msg.NewProxy{}, ctl.handleNewProxy)
	ctl.msgDispatcher.RegisterHandler(&msg.Ping{}, ctl.handlePing)
	ctl.msgDispatcher.RegisterHandler(&msg.NatHoleVisitor{}, msg.AsyncHandler(ctl.handleNatHoleVisitor))
	ctl.msgDispatcher.RegisterHandler(&msg.NatHoleClient{}, msg.AsyncHandler(ctl.handleNatHoleClient))
	ctl.msgDispatcher.RegisterHandler(&msg.NatHoleReport{}, msg.AsyncHandler(ctl.handleNatHoleReport))
	ctl.msgDispatcher.RegisterHandler(&msg.CloseProxy{}, ctl.handleCloseProxy)
}

func (ctl *Control) handleNewProxy(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.NewProxy)

	content := &plugin.NewProxyContent{
		User:     ctl.loginUserInfo(),
		NewProxy: *inMsg,
	}
	var remoteAddr string
	retContent, err := ctl.sessionCtx.PluginManager.NewProxy(content)
	if err == nil {
		inMsg = &retContent.NewProxy
		remoteAddr, err = ctl.RegisterProxy(inMsg)
	}

	// register proxy in this control
	resp := &msg.NewProxyResp{
		ProxyName: inMsg.ProxyName,
	}
	if err != nil {
		xl.Warnf("new proxy [%s] type [%s] error: %v", inMsg.ProxyName, inMsg.ProxyType, err)
		resp.Error = util.GenerateResponseErrorString(fmt.Sprintf("new proxy [%s] error", inMsg.ProxyName),
			err, lo.FromPtr(ctl.sessionCtx.ServerCfg.DetailedErrorsToClient))
	} else {
		resp.RemoteAddr = remoteAddr
		xl.Infof("new proxy [%s] type [%s] success", inMsg.ProxyName, inMsg.ProxyType)
		clientID := ctl.sessionCtx.LoginMsg.ClientID
		if clientID == "" {
			clientID = ctl.sessionCtx.LoginMsg.RunID
		}
		metrics.Server.NewProxy(inMsg.ProxyName, inMsg.ProxyType, ctl.sessionCtx.LoginMsg.User, clientID)
	}
	_ = ctl.msgDispatcher.Send(resp)
}

func (ctl *Control) handlePing(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.Ping)

	content := &plugin.PingContent{
		User: ctl.loginUserInfo(),
		Ping: *inMsg,
	}
	retContent, err := ctl.sessionCtx.PluginManager.Ping(content)
	if err == nil {
		inMsg = &retContent.Ping
		err = ctl.sessionCtx.AuthVerifier.VerifyPing(inMsg)
	}
	if err != nil {
		xl.Warnf("received invalid ping: %v", err)
		_ = ctl.msgDispatcher.Send(&msg.Pong{
			Error: util.GenerateResponseErrorString("invalid ping", err, lo.FromPtr(ctl.sessionCtx.ServerCfg.DetailedErrorsToClient)),
		})
		return
	}
	ctl.lastPing.Store(time.Now())
	xl.Debugf("receive heartbeat")
	_ = ctl.msgDispatcher.Send(&msg.Pong{})
}

func (ctl *Control) handleNatHoleVisitor(m msg.Message) {
	inMsg := m.(*msg.NatHoleVisitor)
	ctl.sessionCtx.RC.NatHoleController.HandleVisitor(inMsg, ctl.msgTransporter, ctl.sessionCtx.LoginMsg.User)
}

func (ctl *Control) handleNatHoleClient(m msg.Message) {
	inMsg := m.(*msg.NatHoleClient)
	ctl.sessionCtx.RC.NatHoleController.HandleClient(inMsg, ctl.msgTransporter)
}

func (ctl *Control) handleNatHoleReport(m msg.Message) {
	inMsg := m.(*msg.NatHoleReport)
	ctl.sessionCtx.RC.NatHoleController.HandleReport(inMsg)
}

func (ctl *Control) handleCloseProxy(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.CloseProxy)
	_ = ctl.CloseProxy(inMsg)
	xl.Infof("close proxy [%s] success", inMsg.ProxyName)
}

func (ctl *Control) RegisterProxy(pxyMsg *msg.NewProxy) (remoteAddr string, err error) {
	var pxyConf v1.ProxyConfigurer
	// Load configures from NewProxy message and validate.
	pxyConf, err = config.NewProxyConfigurerFromMsg(pxyMsg, ctl.sessionCtx.ServerCfg)
	if err != nil {
		return
	}

	// User info
	userInfo := plugin.UserInfo{
		User:  ctl.sessionCtx.LoginMsg.User,
		Metas: ctl.sessionCtx.LoginMsg.Metas,
		RunID: ctl.runID,
	}

	// NewProxy will return an interface Proxy.
	// In fact, it creates different proxies based on the proxy type. We just call run() here.
	pxy, err := proxy.NewProxy(ctl.ctx, &proxy.Options{
		UserInfo:           userInfo,
		LoginMsg:           ctl.sessionCtx.LoginMsg,
		PoolCount:          ctl.poolCount,
		ResourceController: ctl.sessionCtx.RC,
		GetWorkConnFn:      ctl.GetWorkConn,
		Configurer:         pxyConf,
		ServerCfg:          ctl.sessionCtx.ServerCfg,
		EncryptionKey:      ctl.sessionCtx.EncryptionKey,
	})
	if err != nil {
		return remoteAddr, err
	}

	// Check ports used number in each client
	if ctl.sessionCtx.ServerCfg.MaxPortsPerClient > 0 {
		ctl.mu.Lock()
		if ctl.portsUsedNum+pxy.GetUsedPortsNum() > int(ctl.sessionCtx.ServerCfg.MaxPortsPerClient) {
			ctl.mu.Unlock()
			err = fmt.Errorf("exceed the max_ports_per_client")
			return
		}
		ctl.portsUsedNum += pxy.GetUsedPortsNum()
		ctl.mu.Unlock()

		defer func() {
			if err != nil {
				ctl.mu.Lock()
				ctl.portsUsedNum -= pxy.GetUsedPortsNum()
				ctl.mu.Unlock()
			}
		}()
	}

	if ctl.sessionCtx.PxyManager.Exist(pxyMsg.ProxyName) {
		err = fmt.Errorf("proxy [%s] already exists", pxyMsg.ProxyName)
		return
	}

	remoteAddr, err = pxy.Run()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			pxy.Close()
		}
	}()

	err = ctl.sessionCtx.PxyManager.Add(pxyMsg.ProxyName, pxy)
	if err != nil {
		return
	}

	ctl.mu.Lock()
	ctl.proxies[pxy.GetName()] = pxy
	ctl.mu.Unlock()
	return
}

func (ctl *Control) CloseProxy(closeMsg *msg.CloseProxy) (err error) {
	ctl.mu.Lock()
	pxy, ok := ctl.proxies[closeMsg.ProxyName]
	if !ok {
		ctl.mu.Unlock()
		return
	}

	if ctl.sessionCtx.ServerCfg.MaxPortsPerClient > 0 {
		ctl.portsUsedNum -= pxy.GetUsedPortsNum()
	}
	delete(ctl.proxies, closeMsg.ProxyName)
	ctl.mu.Unlock()

	ctl.closeProxy(pxy)
	return
}

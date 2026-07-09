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
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/client/visitor"
	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/naming"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/wait"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/pkg/vnet"
)

type SessionContext struct {
	// The client common configuration.
	Common *v1.ClientCommonConfig

	// Unique ID obtained from frps.
	// It should be attached to the login message when reconnecting.
	RunID string
	// Underlying control connection. Once conn is closed, the msgDispatcher and the entire Control will exit.
	Conn *msg.Conn
	// Auth runtime used for login, heartbeats, and encryption.
	Auth *auth.ClientAuth
	// Connector is used to create message connections to frps.
	Connector MessageConnector
	// Virtual net controller
	VnetController *vnet.Controller
}

type Control struct {
	// service context
	ctx context.Context
	xl  *xlog.Logger

	// session context
	sessionCtx *SessionContext

	// manage all proxies
	pm *proxy.Manager

	// manage all visitors
	vm *visitor.Manager

	doneCh chan struct{}

	// of time.Time, last time got the Pong message
	lastPong atomic.Value

	// The role of msgTransporter is similar to HTTP2.
	// It allows multiple messages to be sent simultaneously on the same control connection.
	// The server's response messages will be dispatched to the corresponding waiting goroutines based on the laneKey and message type.
	msgTransporter transport.MessageTransporter

	// msgDispatcher is a wrapper for control connection.
	// It provides a channel for sending messages, and you can register handlers to process messages based on their respective types.
	msgDispatcher *msg.Dispatcher
}

// NewControl wires a control connection. The proxy manager (pm) is per-control
// and is torn down when the connection drops (relay proxies must re-register).
// The visitor manager (vm) is owned by the Service and shared across reconnects,
// so P2P visitor tunnels survive frps going down; NewControl only borrows it and
// (re)binds it to this control in Run.
func NewControl(ctx context.Context, sessionCtx *SessionContext, vm *visitor.Manager) (*Control, error) {
	// new xlog instance
	ctl := &Control{
		ctx:        ctx,
		xl:         xlog.FromContextSafe(ctx),
		sessionCtx: sessionCtx,
		vm:         vm,
		doneCh:     make(chan struct{}),
	}
	ctl.lastPong.Store(time.Now())

	ctl.msgDispatcher = msg.NewDispatcher(sessionCtx.Conn)
	ctl.registerMsgHandlers()
	ctl.msgTransporter = transport.NewMessageTransporter(ctl.msgDispatcher)

	ctl.pm = proxy.NewManager(ctl.ctx, sessionCtx.Common, sessionCtx.Auth.EncryptionKey(), ctl.msgTransporter, sessionCtx.VnetController)
	return ctl, nil
}

func (ctl *Control) Run(proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) {
	go ctl.worker()

	// start all proxies
	ctl.pm.UpdateAll(proxyCfgs)

	// Bind the shared visitor manager to this (re)connected control, then apply
	// visitor configs. Existing visitors persist across reconnects; only their
	// session context is refreshed here.
	ctl.vm.UpdateSession(ctl.sessionCtx.RunID, ctl.connectServer, ctl.msgTransporter)
	ctl.vm.UpdateAll(visitorCfgs)
}

func (ctl *Control) SetInWorkConnCallback(cb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool) {
	ctl.pm.SetInWorkConnCallback(cb)
}

func (ctl *Control) handleReqWorkConn(_ msg.Message) {
	xl := ctl.xl
	workConn, err := ctl.connectServer()
	if err != nil {
		xl.Warnf("start new connection to server error: %v", err)
		return
	}

	m := &msg.NewWorkConn{
		RunID: ctl.sessionCtx.RunID,
	}
	if err = ctl.sessionCtx.Auth.Setter.SetNewWorkConn(m); err != nil {
		xl.Warnf("error during NewWorkConn authentication: %v", err)
		workConn.Close()
		return
	}
	if err = workConn.WriteMsg(m); err != nil {
		xl.Warnf("work connection write to server error: %v", err)
		workConn.Close()
		return
	}

	var startMsg msg.StartWorkConn
	if err = workConn.ReadMsgInto(&startMsg); err != nil {
		xl.Tracef("work connection closed before response StartWorkConn message: %v", err)
		workConn.Close()
		return
	}
	if startMsg.Error != "" {
		xl.Errorf("StartWorkConn contains error: %s", startMsg.Error)
		workConn.Close()
		return
	}

	startMsg.ProxyName = naming.StripUserPrefix(ctl.sessionCtx.Common.User, startMsg.ProxyName)

	// dispatch this work connection to related proxy
	ctl.pm.HandleWorkConn(startMsg.ProxyName, workConn, &startMsg)
}

func (ctl *Control) handleNewProxyResp(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.NewProxyResp)
	// Server will return NewProxyResp message to each NewProxy message.
	// Start a new proxy handler if no error got
	proxyName := naming.StripUserPrefix(ctl.sessionCtx.Common.User, inMsg.ProxyName)
	err := ctl.pm.StartProxy(proxyName, inMsg.RemoteAddr, inMsg.Error)
	if err != nil {
		xl.Warnf("[%s] start error: %v", proxyName, err)
	} else {
		xl.Infof("[%s] start proxy success", proxyName)
	}
}

func (ctl *Control) handleNatHoleResp(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.NatHoleResp)

	// Dispatch the NatHoleResp message to the related proxy.
	ok := ctl.msgTransporter.DispatchWithType(inMsg, msg.TypeNameNatHoleResp, inMsg.TransactionID)
	if !ok {
		xl.Tracef("dispatch NatHoleResp message to related proxy error")
	}
}

func (ctl *Control) handlePong(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.Pong)

	if inMsg.Error != "" {
		xl.Errorf("pong message contains error: %s", inMsg.Error)
		ctl.closeSession()
		return
	}
	ctl.lastPong.Store(time.Now())
	xl.Debugf("receive heartbeat from server")
}

// closeSession closes the control connection.
func (ctl *Control) closeSession() {
	ctl.sessionCtx.Conn.Close()
	ctl.sessionCtx.Connector.Close()
}

func (ctl *Control) Close() error {
	return ctl.GracefulClose(0)
}

func (ctl *Control) GracefulClose(d time.Duration) error {
	// Only the per-control proxy manager is closed here. The visitor manager is
	// owned by the Service and must outlive this control so P2P tunnels survive.
	ctl.pm.Close()

	time.Sleep(d)

	ctl.closeSession()
	return nil
}

// Done returns a channel that will be closed after all resources are released
func (ctl *Control) Done() <-chan struct{} {
	return ctl.doneCh
}

// connectServer return a new connection to frps
func (ctl *Control) connectServer() (*msg.Conn, error) {
	return ctl.sessionCtx.Connector.Connect()
}

func (ctl *Control) registerMsgHandlers() {
	ctl.msgDispatcher.RegisterHandler(&msg.ReqWorkConn{}, msg.AsyncHandler(ctl.handleReqWorkConn))
	ctl.msgDispatcher.RegisterHandler(&msg.NewProxyResp{}, ctl.handleNewProxyResp)
	ctl.msgDispatcher.RegisterHandler(&msg.NatHoleResp{}, ctl.handleNatHoleResp)
	ctl.msgDispatcher.RegisterHandler(&msg.Pong{}, ctl.handlePong)
}

// heartbeatWorker sends heartbeat to server and check heartbeat timeout.
func (ctl *Control) heartbeatWorker() {
	xl := ctl.xl

	if ctl.sessionCtx.Common.Transport.HeartbeatInterval > 0 {
		// Send heartbeat to server.
		sendHeartBeat := func() (bool, error) {
			xl.Debugf("send heartbeat to server")
			pingMsg := &msg.Ping{}
			if err := ctl.sessionCtx.Auth.Setter.SetPing(pingMsg); err != nil {
				xl.Warnf("error during ping authentication: %v, skip sending ping message", err)
				return false, err
			}
			_ = ctl.msgDispatcher.Send(pingMsg)
			return false, nil
		}

		go wait.BackoffUntil(sendHeartBeat,
			wait.NewFastBackoffManager(wait.FastBackoffOptions{
				Duration:           time.Duration(ctl.sessionCtx.Common.Transport.HeartbeatInterval) * time.Second,
				InitDurationIfFail: time.Second,
				Factor:             2.0,
				Jitter:             0.1,
				MaxDuration:        time.Duration(ctl.sessionCtx.Common.Transport.HeartbeatInterval) * time.Second,
			}),
			true, ctl.doneCh,
		)
	}

	// Check heartbeat timeout.
	if ctl.sessionCtx.Common.Transport.HeartbeatInterval > 0 && ctl.sessionCtx.Common.Transport.HeartbeatTimeout > 0 {
		go wait.Until(func() {
			if time.Since(ctl.lastPong.Load().(time.Time)) > time.Duration(ctl.sessionCtx.Common.Transport.HeartbeatTimeout)*time.Second {
				xl.Warnf("heartbeat timeout")
				ctl.closeSession()
				return
			}
		}, time.Second, ctl.doneCh)
	}
}

func (ctl *Control) worker() {
	xl := ctl.xl
	go ctl.heartbeatWorker()
	go ctl.msgDispatcher.Run()

	<-ctl.msgDispatcher.Done()
	xl.Debugf("control message dispatcher exited")
	ctl.closeSession()

	// Tear down only the per-control proxy manager. The Service-owned visitor
	// manager (and its live P2P tunnels) stays up across the lost connection.
	ctl.pm.Close()
	close(ctl.doneCh)
}

func (ctl *Control) UpdateAllConfigurer(proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error {
	ctl.vm.UpdateAll(visitorCfgs)
	ctl.pm.UpdateAll(proxyCfgs)
	return nil
}

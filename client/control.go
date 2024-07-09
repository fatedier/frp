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

	"github.com/samber/lo"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/client/visitor"
	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/transport"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/wait"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type SessionContext struct {
	// The client common configuration.
	Common *v1.ClientCommonConfig

	// Unique ID obtained from frps.
	// It should be attached to the login message when reconnecting.
	RunID string
	// Underlying control connection. Once conn is closed, the msgDispatcher and the entire Control will exit.
	Conn net.Conn
	// Indicates whether the connection is encrypted.
	ConnEncrypted bool
	// Sets authentication based on selected method
	AuthSetter auth.Setter
	// Connector is used to create new connections, which could be real TCP connections or virtual streams.
	Connector Connector
}

type Control struct {
	svr *Service

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

func NewControl(ctx context.Context, sessionCtx *SessionContext, svr *Service) (*Control, error) {
	// new xlog instance
	ctl := &Control{
		svr:        svr,
		ctx:        ctx,
		xl:         xlog.FromContextSafe(ctx),
		sessionCtx: sessionCtx,
		doneCh:     make(chan struct{}),
	}
	ctl.lastPong.Store(time.Now())

	if sessionCtx.ConnEncrypted {
		cryptoRW, err := netpkg.NewCryptoReadWriter(sessionCtx.Conn, []byte(sessionCtx.Common.Auth.Token))
		if err != nil {
			return nil, err
		}
		ctl.msgDispatcher = msg.NewDispatcher(cryptoRW)
	} else {
		ctl.msgDispatcher = msg.NewDispatcher(sessionCtx.Conn)
	}
	ctl.registerMsgHandlers()
	ctl.msgTransporter = transport.NewMessageTransporter(ctl.msgDispatcher.SendChannel())

	ctl.pm = proxy.NewManager(ctl.ctx, sessionCtx.Common, ctl.msgTransporter)
	ctl.vm = visitor.NewManager(ctl.ctx, sessionCtx.RunID, sessionCtx.Common, ctl.connectServer, ctl.msgTransporter)
	return ctl, nil
}

func (ctl *Control) Run(proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) {
	go ctl.worker()

	// start all proxies
	ctl.pm.UpdateAll(proxyCfgs)

	// start all visitors
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
	if err = ctl.sessionCtx.AuthSetter.SetNewWorkConn(m); err != nil {
		xl.Warnf("error during NewWorkConn authentication: %v", err)
		workConn.Close()
		return
	}
	if err = msg.WriteMsg(workConn, m); err != nil {
		xl.Warnf("work connection write to server error: %v", err)
		workConn.Close()
		return
	}

	var startMsg msg.StartWorkConn
	if err = msg.ReadMsgInto(workConn, &startMsg); err != nil {
		xl.Tracef("work connection closed before response StartWorkConn message: %v", err)
		workConn.Close()
		return
	}
	if startMsg.Error != "" {
		xl.Errorf("StartWorkConn contains error: %s", startMsg.Error)
		workConn.Close()
		return
	}

	// dispatch this work connection to related proxy
	ctl.pm.HandleWorkConn(startMsg.ProxyName, workConn, &startMsg)
}

func (ctl *Control) handleNewProxyResp(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.NewProxyResp)
	// Server will return NewProxyResp message to each NewProxy message.
	// Start a new proxy handler if no error got
	err := ctl.pm.StartProxy(inMsg.ProxyName, inMsg.RemoteAddr, inMsg.Error)
	if err != nil {
		xl.Warnf("[%s] start error: %v", inMsg.ProxyName, err)
	} else {
		xl.Infof("[%s] start proxy success", inMsg.ProxyName)
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

func (ctl *Control) handleCustom(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.ClientProxyClose)
	xl.Infof("client get close message: %s", inMsg)
	ctl.svr.GracefulClose(0)
}

func (ctl *Control) handlePong(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.Pong)

	if inMsg.AuthErr != "" {
		xl.Errorf("Pong message contains auth error: %s", inMsg.AuthErr)
		ctl.svr.GracefulClose(0)
		return
	}

	if inMsg.Error != "" {
		xl.Errorf("Pong message contains error: %s", inMsg.Error)
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
	ctl.pm.Close()
	ctl.vm.Close()

	time.Sleep(d)

	ctl.closeSession()
	return nil
}

// Done returns a channel that will be closed after all resources are released
func (ctl *Control) Done() <-chan struct{} {
	return ctl.doneCh
}

// connectServer return a new connection to frps
func (ctl *Control) connectServer() (net.Conn, error) {
	return ctl.sessionCtx.Connector.Connect()
}

func (ctl *Control) registerMsgHandlers() {
	ctl.msgDispatcher.RegisterHandler(&msg.ReqWorkConn{}, msg.AsyncHandler(ctl.handleReqWorkConn))
	ctl.msgDispatcher.RegisterHandler(&msg.NewProxyResp{}, ctl.handleNewProxyResp)
	ctl.msgDispatcher.RegisterHandler(&msg.NatHoleResp{}, ctl.handleNatHoleResp)
	ctl.msgDispatcher.RegisterHandler(&msg.Pong{}, ctl.handlePong)
	ctl.msgDispatcher.RegisterHandler(&msg.ClientProxyClose{}, ctl.handleCustom)
}

// headerWorker sends heartbeat to server and check heartbeat timeout.
func (ctl *Control) heartbeatWorker() {
	xl := ctl.xl

	// TODO(fatedier): Change default value of HeartbeatInterval to -1 if tcpmux is enabled.
	// Users can still enable heartbeat feature by setting HeartbeatInterval to a positive value.
	if ctl.sessionCtx.Common.Transport.HeartbeatInterval > 0 {
		// send heartbeat to server
		sendHeartBeat := func() (bool, error) {
			xl.Debugf("send heartbeat to server")
			pingMsg := &msg.Ping{}
			if err := ctl.sessionCtx.AuthSetter.SetPing(pingMsg); err != nil {
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

	// Check heartbeat timeout only if TCPMux is not enabled and users don't disable heartbeat feature.
	if ctl.sessionCtx.Common.Transport.HeartbeatInterval > 0 && ctl.sessionCtx.Common.Transport.HeartbeatTimeout > 0 &&
		!lo.FromPtr(ctl.sessionCtx.Common.Transport.TCPMux) {

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
	go ctl.heartbeatWorker()
	go ctl.msgDispatcher.Run()

	<-ctl.msgDispatcher.Done()
	ctl.closeSession()

	ctl.pm.Close()
	ctl.vm.Close()
	close(ctl.doneCh)
}

func (ctl *Control) UpdateAllConfigurer(proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error {
	ctl.vm.UpdateAll(visitorCfgs)
	ctl.pm.UpdateAll(proxyCfgs)
	return nil
}

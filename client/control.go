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
	utilnet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/wait"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type Control struct {
	// service context
	ctx context.Context
	xl  *xlog.Logger

	// The client configuration
	clientCfg *v1.ClientCommonConfig

	// sets authentication based on selected method
	authSetter auth.Setter

	// Unique ID obtained from frps.
	// It should be attached to the login message when reconnecting.
	runID string

	// manage all proxies
	pxyCfgs []v1.ProxyConfigurer
	pm      *proxy.Manager

	// manage all visitors
	vm *visitor.Manager

	// control connection. Once conn is closed, the msgDispatcher and the entire Control will exit.
	conn net.Conn

	// use cm to create new connections, which could be real TCP connections or virtual streams.
	cm *ConnectionManager

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

func NewControl(
	ctx context.Context, runID string, conn net.Conn, cm *ConnectionManager,
	clientCfg *v1.ClientCommonConfig,
	pxyCfgs []v1.ProxyConfigurer,
	visitorCfgs []v1.VisitorConfigurer,
	authSetter auth.Setter,
) (*Control, error) {
	// new xlog instance
	ctl := &Control{
		ctx:        ctx,
		xl:         xlog.FromContextSafe(ctx),
		clientCfg:  clientCfg,
		authSetter: authSetter,
		runID:      runID,
		pxyCfgs:    pxyCfgs,
		conn:       conn,
		cm:         cm,
		doneCh:     make(chan struct{}),
	}
	ctl.lastPong.Store(time.Now())

	cryptoRW, err := utilnet.NewCryptoReadWriter(conn, []byte(clientCfg.Auth.Token))
	if err != nil {
		return nil, err
	}

	ctl.msgDispatcher = msg.NewDispatcher(cryptoRW)
	ctl.registerMsgHandlers()

	ctl.msgTransporter = transport.NewMessageTransporter(ctl.msgDispatcher.SendChannel())

	ctl.pm = proxy.NewManager(ctl.ctx, clientCfg, ctl.msgTransporter)
	ctl.vm = visitor.NewManager(ctl.ctx, ctl.runID, ctl.clientCfg, ctl.connectServer, ctl.msgTransporter)
	ctl.vm.Reload(visitorCfgs)
	return ctl, nil
}

func (ctl *Control) Run() {
	go ctl.worker()

	// start all proxies
	ctl.pm.Reload(ctl.pxyCfgs)

	// start all visitors
	go ctl.vm.Run()
}

func (ctl *Control) handleReqWorkConn(_ msg.Message) {
	xl := ctl.xl
	workConn, err := ctl.connectServer()
	if err != nil {
		xl.Warn("start new connection to server error: %v", err)
		return
	}

	m := &msg.NewWorkConn{
		RunID: ctl.runID,
	}

	if err = ctl.authSetter.SetNewWorkConn(m); err != nil {
		xl.Warn("error during NewWorkConn authentication: %v", err)
		return
	}

	if err = msg.WriteMsg(workConn, m); err != nil {
		xl.Warn("work connection write to server error: %v", err)
		workConn.Close()
		return
	}

	var startMsg msg.StartWorkConn
	if err = msg.ReadMsgInto(workConn, &startMsg); err != nil {
		xl.Trace("work connection closed before response StartWorkConn message: %v", err)
		workConn.Close()
		return
	}

	if startMsg.Error != "" {
		xl.Error("StartWorkConn contains error: %s", startMsg.Error)
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
		xl.Warn("[%s] start error: %v", inMsg.ProxyName, err)
	} else {
		xl.Info("[%s] start proxy success", inMsg.ProxyName)
	}
}

func (ctl *Control) handleNatHoleResp(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.NatHoleResp)

	// Dispatch the NatHoleResp message to the related proxy.
	ok := ctl.msgTransporter.DispatchWithType(inMsg, msg.TypeNameNatHoleResp, inMsg.TransactionID)
	if !ok {
		xl.Trace("dispatch NatHoleResp message to related proxy error")
	}
}

func (ctl *Control) handlePong(m msg.Message) {
	xl := ctl.xl
	inMsg := m.(*msg.Pong)

	if inMsg.Error != "" {
		xl.Error("Pong message contains error: %s", inMsg.Error)
		ctl.conn.Close()
		return
	}
	ctl.lastPong.Store(time.Now())
	xl.Debug("receive heartbeat from server")
}

func (ctl *Control) Close() error {
	return ctl.GracefulClose(0)
}

func (ctl *Control) GracefulClose(d time.Duration) error {
	ctl.pm.Close()
	ctl.vm.Close()

	time.Sleep(d)

	ctl.conn.Close()
	ctl.cm.Close()
	return nil
}

// Done returns a channel that will be closed after all resources are released
func (ctl *Control) Done() <-chan struct{} {
	return ctl.doneCh
}

// connectServer return a new connection to frps
func (ctl *Control) connectServer() (conn net.Conn, err error) {
	return ctl.cm.Connect()
}

func (ctl *Control) registerMsgHandlers() {
	ctl.msgDispatcher.RegisterHandler(&msg.ReqWorkConn{}, msg.AsyncHandler(ctl.handleReqWorkConn))
	ctl.msgDispatcher.RegisterHandler(&msg.NewProxyResp{}, ctl.handleNewProxyResp)
	ctl.msgDispatcher.RegisterHandler(&msg.NatHoleResp{}, ctl.handleNatHoleResp)
	ctl.msgDispatcher.RegisterHandler(&msg.Pong{}, ctl.handlePong)
}

// headerWorker sends heartbeat to server and check heartbeat timeout.
func (ctl *Control) heartbeatWorker() {
	xl := ctl.xl

	// TODO(fatedier): Change default value of HeartbeatInterval to -1 if tcpmux is enabled.
	// Users can still enable heartbeat feature by setting HeartbeatInterval to a positive value.
	if ctl.clientCfg.Transport.HeartbeatInterval > 0 {
		// send heartbeat to server
		sendHeartBeat := func() error {
			xl.Debug("send heartbeat to server")
			pingMsg := &msg.Ping{}
			if err := ctl.authSetter.SetPing(pingMsg); err != nil {
				xl.Warn("error during ping authentication: %v, skip sending ping message", err)
				return err
			}
			_ = ctl.msgDispatcher.Send(pingMsg)
			return nil
		}

		go wait.BackoffUntil(sendHeartBeat,
			wait.NewFastBackoffManager(wait.FastBackoffOptions{
				Duration:           time.Duration(ctl.clientCfg.Transport.HeartbeatInterval) * time.Second,
				InitDurationIfFail: time.Second,
				Factor:             2.0,
				Jitter:             0.1,
				MaxDuration:        time.Duration(ctl.clientCfg.Transport.HeartbeatInterval) * time.Second,
			}),
			true, ctl.doneCh,
		)
	}

	// Check heartbeat timeout only if TCPMux is not enabled and users don't disable heartbeat feature.
	if ctl.clientCfg.Transport.HeartbeatInterval > 0 && ctl.clientCfg.Transport.HeartbeatTimeout > 0 &&
		!lo.FromPtr(ctl.clientCfg.Transport.TCPMux) {

		go wait.Until(func() {
			if time.Since(ctl.lastPong.Load().(time.Time)) > time.Duration(ctl.clientCfg.Transport.HeartbeatTimeout)*time.Second {
				xl.Warn("heartbeat timeout")
				ctl.conn.Close()
				return
			}
		}, time.Second, ctl.doneCh)
	}
}

func (ctl *Control) worker() {
	go ctl.heartbeatWorker()
	go ctl.msgDispatcher.Run()

	<-ctl.msgDispatcher.Done()
	ctl.conn.Close()

	ctl.pm.Close()
	ctl.vm.Close()
	ctl.cm.Close()

	close(ctl.doneCh)
}

func (ctl *Control) ReloadConf(pxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error {
	ctl.vm.Reload(visitorCfgs)
	ctl.pm.Reload(pxyCfgs)
	return nil
}

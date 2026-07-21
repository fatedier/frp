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
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/wait"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/server/controller"
	"github.com/fatedier/frp/server/metrics"
	"github.com/fatedier/frp/server/proxy"
	"github.com/fatedier/frp/server/registry"
)

type ControlID uint64

var nextControlID atomic.Uint64

type controlEntry struct {
	ctl *Control
	id  ControlID
	// runMu serializes lifecycle and routing decisions for one run ID.
	// Replacements inherit it; removing the entry releases the manager's reference.
	runMu *sync.Mutex

	registryOnline    bool
	registryControlID ControlID
}

type ControlManager struct {
	// controls indexed by run id
	ctlsByRunID map[string]*controlEntry
	registry    *registry.ClientRegistry
	closed      bool

	mu sync.RWMutex
}

func NewControlManager(clientRegistry *registry.ClientRegistry) *ControlManager {
	return &ControlManager{
		ctlsByRunID: make(map[string]*controlEntry),
		registry:    clientRegistry,
	}
}

// lockCurrentRun returns the current entry with its run gate held. It never
// waits for the gate while holding cm.mu and revalidates the gate after waiting.
// The global order is runMu, cm.mu, ctl.lifecycleMu, then registry locks.
func (cm *ControlManager) lockCurrentRun(runID string, allowClosed bool) (*controlEntry, bool) {
	cm.mu.RLock()
	entry, ok := cm.ctlsByRunID[runID]
	if cm.closed && !allowClosed {
		ok = false
	}
	cm.mu.RUnlock()
	if !ok {
		return nil, false
	}

	runMu := entry.runMu
	runMu.Lock()
	cm.mu.RLock()
	entry, ok = cm.ctlsByRunID[runID]
	if (cm.closed && !allowClosed) || !ok || entry.runMu != runMu {
		ok = false
	}
	cm.mu.RUnlock()
	if !ok {
		runMu.Unlock()
		return nil, false
	}
	return entry, true
}

// Add makes ctl the pending current generation and records the predecessor
// finalization barrier it must wait for before activation.
func (cm *ControlManager) Add(ctl *Control) error {
	for {
		// Never wait for a run gate while holding cm.mu.
		cm.mu.RLock()
		old := cm.ctlsByRunID[ctl.runID]
		cm.mu.RUnlock()
		if old != nil {
			old.runMu.Lock()
		}

		cm.mu.Lock()
		if cm.closed {
			cm.mu.Unlock()
			if old != nil {
				old.runMu.Unlock()
			}
			return fmt.Errorf("control manager is closed")
		}
		if cm.ctlsByRunID[ctl.runID] != old {
			cm.mu.Unlock()
			if old != nil {
				old.runMu.Unlock()
			}
			continue
		}

		id := ControlID(nextControlID.Add(1))
		if err := ctl.admit(cm, id); err != nil {
			cm.mu.Unlock()
			if old != nil {
				old.runMu.Unlock()
			}
			return err
		}

		runMu := &sync.Mutex{}
		if old != nil {
			runMu = old.runMu
		}
		entry := &controlEntry{ctl: ctl, id: id, runMu: runMu}
		var (
			oldCtl  *Control
			barrier <-chan struct{}
		)
		if old != nil {
			oldCtl = old.ctl
			barrier = oldCtl.markReplaced()
			ctl.setHandoffBarrier(barrier)
			entry.registryOnline = old.registryOnline
			entry.registryControlID = old.registryControlID
		}
		cm.ctlsByRunID[ctl.runID] = entry
		cm.mu.Unlock()
		if old != nil {
			old.runMu.Unlock()
		}

		if oldCtl != nil {
			oldCtl.Replaced(ctl)
		}
		return nil
	}
}

// Activate registers ctl as online only if it is still the pending current
// generation.
func (cm *ControlManager) Activate(ctl *Control) (bool, error) {
	entry, ok := cm.lockCurrentRun(ctl.runID, false)
	if !ok {
		return false, nil
	}
	defer entry.runMu.Unlock()
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.closed || cm.ctlsByRunID[ctl.runID] != entry || entry.ctl != ctl || entry.id != ctl.controlID {
		return false, nil
	}

	ctl.lifecycleMu.Lock()
	defer ctl.lifecycleMu.Unlock()
	if ctl.state != controlStatePending {
		return false, nil
	}
	if ctl.activated {
		return true, nil
	}

	loginMsg := ctl.sessionCtx.LoginMsg
	remoteAddr := ctl.sessionCtx.Conn.RemoteAddr().String()
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		remoteAddr = host
	}
	_, conflict := cm.registry.RegisterWithControlID(
		loginMsg.User,
		loginMsg.ClientID,
		ctl.runID,
		loginMsg.Hostname,
		loginMsg.Version,
		remoteAddr,
		ctl.sessionCtx.WireProtocol,
		uint64(entry.id),
	)
	if conflict {
		return true, fmt.Errorf("client_id [%s] for user [%s] is already online", loginMsg.ClientID, loginMsg.User)
	}

	entry.registryOnline = true
	entry.registryControlID = entry.id
	ctl.activated = true
	return true, nil
}

// completeLogin reserves ctl's current ownership with its run gate while the
// bounded successful LoginResp write runs, then transitions it to running.
// The callback must only perform that bounded write; it must not call back into
// the control manager or the same control lifecycle.
func (cm *ControlManager) completeLogin(ctl *Control, writeSuccess func() error) (bool, error) {
	entry, ok := cm.lockCurrentRun(ctl.runID, false)
	if !ok {
		return false, nil
	}
	defer entry.runMu.Unlock()
	if entry.ctl != ctl || entry.id != ctl.controlID {
		return false, nil
	}

	ctl.lifecycleMu.Lock()
	defer ctl.lifecycleMu.Unlock()
	if ctl.state != controlStatePending || !ctl.activated {
		return false, nil
	}
	if err := writeSuccess(); err != nil {
		return false, err
	}
	if !ctl.startLocked() {
		return false, nil
	}
	return true, nil
}

// Remove deletes and offlines ctl only if it is still the current generation.
func (cm *ControlManager) Remove(ctl *Control) bool {
	entry, ok := cm.lockCurrentRun(ctl.runID, true)
	if !ok {
		return false
	}
	defer entry.runMu.Unlock()
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.ctlsByRunID[ctl.runID] != entry || entry.ctl != ctl || entry.id != ctl.controlID {
		return false
	}
	delete(cm.ctlsByRunID, ctl.runID)
	if entry.registryOnline {
		cm.registry.MarkOfflineByRunIDAndControlID(ctl.runID, uint64(entry.registryControlID))
	}
	return true
}

func (cm *ControlManager) GetByID(runID string) (ctl *Control, ok bool) {
	entry, ok := cm.lockCurrentRun(runID, false)
	if !ok {
		return nil, false
	}
	defer entry.runMu.Unlock()
	ctl = entry.ctl

	ctl.lifecycleMu.Lock()
	defer ctl.lifecycleMu.Unlock()
	if ctl.state != controlStateRunning {
		return nil, false
	}
	return ctl, true
}

// admitVisitorByRunID commits a visitor admission against the current running
// control while its run and lifecycle ownership are held. The callback must
// only perform the in-memory, buffered visitor admission.
func (cm *ControlManager) admitVisitorByRunID(runID string, admit func(user string) error) (bool, error) {
	entry, ok := cm.lockCurrentRun(runID, false)
	if !ok {
		return false, nil
	}
	defer entry.runMu.Unlock()
	ctl := entry.ctl

	ctl.lifecycleMu.Lock()
	defer ctl.lifecycleMu.Unlock()
	if ctl.state != controlStateRunning {
		return false, nil
	}
	return true, admit(ctl.sessionCtx.LoginMsg.User)
}

// RegisterWorkConn transfers conn to ctl only if ctl is still the current
// running generation. On error, ownership remains with the caller.
func (cm *ControlManager) RegisterWorkConn(ctl *Control, conn *proxy.WorkConn) error {
	entry, ok := cm.lockCurrentRun(ctl.runID, false)
	if !ok {
		cm.mu.RLock()
		closed := cm.closed
		cm.mu.RUnlock()
		if closed {
			return fmt.Errorf("control manager is closed")
		}
		return fmt.Errorf("client control for run id [%s] is no longer current", ctl.runID)
	}
	defer entry.runMu.Unlock()
	if entry.ctl != ctl || entry.id != ctl.controlID {
		return fmt.Errorf("client control for run id [%s] is no longer current", ctl.runID)
	}

	ctl.lifecycleMu.Lock()
	defer ctl.lifecycleMu.Unlock()
	if ctl.state != controlStateRunning {
		return fmt.Errorf("client control for run id [%s] is not running", ctl.runID)
	}

	select {
	case ctl.workConnCh <- conn:
		ctl.xl.Debugf("new work connection registered")
		return nil
	default:
		ctl.xl.Debugf("work connection pool is full, discarding")
		return fmt.Errorf("work connection pool is full, discarding")
	}
}

func (cm *ControlManager) Close() error {
	cm.mu.Lock()
	cm.closed = true
	ctls := make([]*Control, 0, len(cm.ctlsByRunID))
	for _, entry := range cm.ctlsByRunID {
		ctls = append(ctls, entry.ctl)
	}
	cm.mu.Unlock()

	for _, ctl := range ctls {
		cm.Remove(ctl)
		_ = ctl.Close()
	}
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
	Conn *msg.Conn
	// login message
	LoginMsg *msg.Login
	// server configuration
	ServerCfg *v1.ServerConfig
	// negotiated wire protocol for this client session
	WireProtocol string
}

type controlState uint8

const (
	controlStateCreated controlState = iota
	controlStatePending
	controlStateRunning
	controlStateClosing
	controlStateClosed
)

type Control struct {
	// session context
	sessionCtx *SessionContext

	// other components can use this to communicate with client
	msgTransporter transport.MessageTransporter

	// msgDispatcher is a wrapper for control connection.
	// It provides a channel for sending messages, and you can register handlers to process messages based on their respective types.
	msgDispatcher *msg.Dispatcher

	// work connections
	workConnCh chan *proxy.WorkConn

	// proxies in one client
	proxies map[string]proxy.Proxy

	// pool count
	poolCount int

	// ports used, for limitations
	portsUsedNum int

	// last time got the Ping message
	lastPing atomic.Value

	// runID never changes during the lifetime of a control. controlID is assigned
	// once by ControlManager and distinguishes same-runID generations.
	runID     string
	controlID ControlID
	manager   *ControlManager

	lifecycleMu    sync.Mutex
	state          controlState
	activated      bool
	handoffBarrier <-chan struct{}

	interruptOnce sync.Once
	interruptErr  error

	mu sync.RWMutex

	xl            *xlog.Logger
	ctx           context.Context
	doneCh        chan struct{}
	serverMetrics metrics.ServerMetrics
}

func NewControl(ctx context.Context, sessionCtx *SessionContext) (*Control, error) {
	poolCount := min(sessionCtx.LoginMsg.PoolCount, int(sessionCtx.ServerCfg.Transport.MaxPoolCount))
	ctl := &Control{
		sessionCtx:    sessionCtx,
		workConnCh:    make(chan *proxy.WorkConn, poolCount+10),
		proxies:       make(map[string]proxy.Proxy),
		poolCount:     poolCount,
		portsUsedNum:  0,
		runID:         sessionCtx.LoginMsg.RunID,
		state:         controlStateCreated,
		xl:            xlog.FromContextSafe(ctx),
		ctx:           ctx,
		doneCh:        make(chan struct{}),
		serverMetrics: metrics.Server,
	}
	ctl.lastPing.Store(time.Now())

	ctl.msgDispatcher = msg.NewDispatcher(sessionCtx.Conn)
	ctl.registerMsgHandlers()
	ctl.msgTransporter = transport.NewMessageTransporter(ctl.msgDispatcher)
	return ctl, nil
}

func (ctl *Control) RunID() string {
	return ctl.runID
}

func (ctl *Control) ID() ControlID {
	ctl.lifecycleMu.Lock()
	defer ctl.lifecycleMu.Unlock()
	return ctl.controlID
}

func (ctl *Control) admit(manager *ControlManager, id ControlID) error {
	ctl.lifecycleMu.Lock()
	defer ctl.lifecycleMu.Unlock()
	if ctl.state != controlStateCreated {
		return fmt.Errorf("control [%s] is not in created state", ctl.runID)
	}
	ctl.manager = manager
	ctl.controlID = id
	ctl.state = controlStatePending
	return nil
}

func (ctl *Control) setHandoffBarrier(barrier <-chan struct{}) {
	ctl.lifecycleMu.Lock()
	ctl.handoffBarrier = barrier
	ctl.lifecycleMu.Unlock()
}

func (ctl *Control) WaitForHandoff() {
	ctl.lifecycleMu.Lock()
	barrier := ctl.handoffBarrier
	ctl.lifecycleMu.Unlock()
	if barrier != nil {
		<-barrier
	}
}

// Start starts the control session workers after login succeeds.
func (ctl *Control) Start() bool {
	ctl.lifecycleMu.Lock()
	defer ctl.lifecycleMu.Unlock()
	return ctl.startLocked()
}

func (ctl *Control) startLocked() bool {
	if ctl.state != controlStatePending || !ctl.activated {
		return false
	}
	ctl.state = controlStateRunning
	go ctl.worker()
	return true
}

func (ctl *Control) Close() error {
	ctl.lifecycleMu.Lock()
	switch ctl.state {
	case controlStateCreated, controlStatePending:
		ctl.state = controlStateClosing
		ctl.finishLocked()
	case controlStateRunning:
		ctl.state = controlStateClosing
	}
	ctl.lifecycleMu.Unlock()
	return ctl.interruptReadAndClose()
}

func (ctl *Control) Replaced(newCtl *Control) {
	ctl.markReplaced()
	ctl.xl.Infof("replaced by client [%s] (control ID %d)", newCtl.runID, newCtl.ID())
	_ = ctl.interruptReadAndClose()
}

// markReplaced returns the transitive predecessor barrier. A pending control
// has no worker, so it finishes immediately and passes its inherited barrier
// to the replacement. A running control is finished only by its worker.
func (ctl *Control) markReplaced() <-chan struct{} {
	ctl.lifecycleMu.Lock()
	defer ctl.lifecycleMu.Unlock()

	switch ctl.state {
	case controlStateCreated:
		ctl.state = controlStateClosing
		ctl.finishLocked()
		return nil
	case controlStatePending:
		barrier := ctl.handoffBarrier
		ctl.state = controlStateClosing
		ctl.finishLocked()
		return barrier
	case controlStateRunning:
		ctl.state = controlStateClosing
		return ctl.doneCh
	case controlStateClosing, controlStateClosed:
		return ctl.doneCh
	default:
		return ctl.doneCh
	}
}

func (ctl *Control) interruptReadAndClose() error {
	ctl.interruptOnce.Do(func() {
		_ = ctl.sessionCtx.Conn.SetReadDeadline(time.Now())
		ctl.interruptErr = ctl.sessionCtx.Conn.Close()
	})
	return ctl.interruptErr
}

func (ctl *Control) finishLocked() {
	if ctl.state == controlStateClosed {
		return
	}
	ctl.state = controlStateClosed
	close(ctl.doneCh)
}

// When frps get one user connection, we get one work connection from the pool and return it.
// If no workConn available in the pool, send message to frpc to get one or more
// and wait until it is available.
// return an error if wait timeout
func (ctl *Control) GetWorkConn() (workConn *proxy.WorkConn, err error) {
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
	wait.Until(func() {
		if time.Since(ctl.lastPing.Load().(time.Time)) > time.Duration(ctl.sessionCtx.ServerCfg.Transport.HeartbeatTimeout)*time.Second {
			xl.Warnf("heartbeat timeout")
			_ = ctl.Close()
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
		RunID: ctl.runID,
	}
}

func (ctl *Control) closeProxy(pxy proxy.Proxy) {
	pxy.Close()
	ctl.sessionCtx.PxyManager.Del(pxy.GetName())
	ctl.serverMetrics.CloseProxy(pxy.GetName(), pxy.GetConfigurer().GetBaseConfig().Type)

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
	ctl.serverMetrics.NewClient()

	go ctl.heartbeatWorker()
	go ctl.msgDispatcher.Run()
	go func() {
		for i := 0; i < ctl.poolCount; i++ {
			// Ignore the error: it means this control is already closing.
			_ = ctl.msgDispatcher.Send(&msg.ReqWorkConn{})
		}
	}()

	<-ctl.msgDispatcher.Done()
	ctl.lifecycleMu.Lock()
	if ctl.state == controlStateRunning {
		ctl.state = controlStateClosing
	}
	ctl.lifecycleMu.Unlock()
	_ = ctl.interruptReadAndClose()

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

	ctl.serverMetrics.CloseClient()
	if ctl.manager != nil {
		ctl.manager.Remove(ctl)
	}
	xl.Infof("client exit success")
	ctl.lifecycleMu.Lock()
	ctl.finishLocked()
	ctl.lifecycleMu.Unlock()
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
			clientID = ctl.runID
		}
		ctl.serverMetrics.NewProxy(inMsg.ProxyName, inMsg.ProxyType, ctl.sessionCtx.LoginMsg.User, clientID)
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
		WireProtocol:       ctl.sessionCtx.WireProtocol,
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

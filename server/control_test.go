// Copyright 2026 The frp Authors
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
	"errors"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/server/controller"
	"github.com/fatedier/frp/server/proxy"
	"github.com/fatedier/frp/server/registry"
)

func TestControlPendingReplacementFinishesWithoutStarting(t *testing.T) {
	clientRegistry := registry.NewClientRegistry()
	manager := NewControlManager(clientRegistry)
	metrics := newCountingServerMetrics()
	oldCtl, oldConn := newLifecycleTestControl(t, "same-run", "client", metrics)
	newCtl, _ := newLifecycleTestControl(t, "same-run", "client", metrics)

	mustAddAndActivate(t, manager, oldCtl)

	err := manager.Add(newCtl)
	require.NoError(t, err)
	waitForControlDone(t, oldCtl)
	require.False(t, oldCtl.Start())
	require.Equal(t, []string{"deadline", "close"}, oldConn.eventsSnapshot())
	require.Equal(t, int64(0), metrics.newClients())
	require.Equal(t, int64(0), metrics.closedClients())
}

func TestControlRunningReplacementFinishesInWorker(t *testing.T) {
	clientRegistry := registry.NewClientRegistry()
	manager := NewControlManager(clientRegistry)
	metrics := newCountingServerMetrics()
	oldCtl, oldConn := newLifecycleTestControl(t, "same-run", "client", metrics)
	newCtl, _ := newLifecycleTestControl(t, "same-run", "client", metrics)

	mustAddAndActivate(t, manager, oldCtl)
	require.True(t, oldCtl.Start())
	waitForSignal(t, oldConn.readStarted, "control reader to start")

	err := manager.Add(newCtl)
	require.NoError(t, err)
	waitForControlDone(t, oldCtl)
	require.Equal(t, []string{"deadline", "close"}, oldConn.eventsSnapshot())
	require.Equal(t, int64(1), metrics.newClients())
	require.Equal(t, int64(1), metrics.closedClients())

	_, ok := manager.GetByID("same-run")
	require.False(t, ok)
	require.Same(t, newCtl, currentControlForTest(manager, "same-run"))
	info, ok := clientRegistry.GetByKey("client")
	require.True(t, ok)
	require.True(t, info.Online)
	require.Equal(t, uint64(oldCtl.ID()), info.ControlID)

	active, err := manager.Activate(newCtl)
	require.NoError(t, err)
	require.True(t, active)
	_, ok = manager.GetByID("same-run")
	require.False(t, ok)
	info, ok = clientRegistry.GetByKey("client")
	require.True(t, ok)
	require.Equal(t, uint64(newCtl.ID()), info.ControlID)
}

func TestControlClosePendingAndRunning(t *testing.T) {
	t.Run("pending", func(t *testing.T) {
		manager := NewControlManager(registry.NewClientRegistry())
		metrics := newCountingServerMetrics()
		ctl, conn := newLifecycleTestControl(t, "pending", "pending", metrics)
		err := manager.Add(ctl)
		require.NoError(t, err)

		require.NoError(t, ctl.Close())
		waitForControlDone(t, ctl)
		require.Equal(t, []string{"deadline", "close"}, conn.eventsSnapshot())
		require.Equal(t, int64(0), metrics.newClients())
		require.Equal(t, int64(0), metrics.closedClients())
	})

	t.Run("running", func(t *testing.T) {
		manager := NewControlManager(registry.NewClientRegistry())
		metrics := newCountingServerMetrics()
		ctl, conn := newLifecycleTestControl(t, "running", "running", metrics)
		mustAddAndActivate(t, manager, ctl)
		require.True(t, ctl.Start())
		waitForSignal(t, conn.readStarted, "control reader to start")

		require.NoError(t, ctl.Close())
		waitForControlDone(t, ctl)
		require.Equal(t, []string{"deadline", "close"}, conn.eventsSnapshot())
		require.Equal(t, int64(1), metrics.newClients())
		require.Equal(t, int64(1), metrics.closedClients())
	})
}

func TestControlCloseAndReplacedAreIdempotent(t *testing.T) {
	manager := NewControlManager(registry.NewClientRegistry())
	metrics := newCountingServerMetrics()
	ctl, conn := newLifecycleTestControl(t, "same-run", "client", metrics)
	replacement, _ := newLifecycleTestControl(t, "same-run", "client", metrics)

	err := manager.Add(ctl)
	require.NoError(t, err)
	err = manager.Add(replacement)
	require.NoError(t, err)
	require.NoError(t, ctl.Close())
	ctl.Replaced(replacement)
	require.NoError(t, ctl.Close())
	waitForControlDone(t, ctl)

	require.Equal(t, []string{"deadline", "close"}, conn.eventsSnapshot())
	require.Equal(t, int64(0), metrics.newClients())
	require.Equal(t, int64(0), metrics.closedClients())
}

func TestControlHeartbeatTimeoutInterruptsRead(t *testing.T) {
	manager := NewControlManager(registry.NewClientRegistry())
	metrics := newCountingServerMetrics()
	ctl, conn := newLifecycleTestControl(t, "heartbeat", "heartbeat", metrics)
	ctl.sessionCtx.ServerCfg.Transport.HeartbeatTimeout = 1
	ctl.lastPing.Store(time.Now().Add(-2 * time.Second))

	mustAddAndActivate(t, manager, ctl)
	require.True(t, ctl.Start())
	waitForSignal(t, conn.readStarted, "control reader to start")
	waitForControlDone(t, ctl)

	require.Equal(t, []string{"deadline", "close"}, conn.eventsSnapshot())
	require.Equal(t, int64(1), metrics.newClients())
	require.Equal(t, int64(1), metrics.closedClients())
}

func TestControlStartReplacementRacePairsMetrics(t *testing.T) {
	for range 100 {
		clientRegistry := registry.NewClientRegistry()
		manager := NewControlManager(clientRegistry)
		metrics := newCountingServerMetrics()
		ctl, _ := newLifecycleTestControl(t, "same-run", "client", metrics)
		replacement, _ := newLifecycleTestControl(t, "same-run", "client", metrics)

		mustAddAndActivate(t, manager, ctl)

		startGate := make(chan struct{})
		startedCh := make(chan bool, 1)
		addErrCh := make(chan error, 1)
		go func() {
			<-startGate
			startedCh <- ctl.Start()
		}()
		go func() {
			<-startGate
			addErr := manager.Add(replacement)
			addErrCh <- addErr
		}()
		close(startGate)

		started := <-startedCh
		require.NoError(t, <-addErrCh)
		waitForControlDone(t, ctl)
		if started {
			require.Equal(t, int64(1), metrics.newClients())
			require.Equal(t, int64(1), metrics.closedClients())
		} else {
			require.Equal(t, int64(0), metrics.newClients())
			require.Equal(t, int64(0), metrics.closedClients())
		}
	}
}

func TestControlManagerRejectsStaleActivateAndRemove(t *testing.T) {
	clientRegistry := registry.NewClientRegistry()
	manager := NewControlManager(clientRegistry)
	metrics := newCountingServerMetrics()
	oldCtl, _ := newLifecycleTestControl(t, "same-run", "client", metrics)
	newCtl, _ := newLifecycleTestControl(t, "same-run", "client", metrics)

	mustAddAndActivate(t, manager, oldCtl)
	err := manager.Add(newCtl)
	require.NoError(t, err)
	require.Greater(t, uint64(newCtl.ID()), uint64(oldCtl.ID()))

	active, err := manager.Activate(oldCtl)
	require.NoError(t, err)
	require.False(t, active)
	require.False(t, manager.Remove(oldCtl))

	_, ok := manager.GetByID("same-run")
	require.False(t, ok)
	require.Same(t, newCtl, currentControlForTest(manager, "same-run"))
	info, ok := clientRegistry.GetByKey("client")
	require.True(t, ok)
	require.True(t, info.Online)
	require.Equal(t, uint64(oldCtl.ID()), info.ControlID)

	active, err = manager.Activate(newCtl)
	require.NoError(t, err)
	require.True(t, active)
	info, ok = clientRegistry.GetByKey("client")
	require.True(t, ok)
	require.True(t, info.Online)
	require.Equal(t, uint64(newCtl.ID()), info.ControlID)
}

func TestControlManagerPreservesClientIDConflict(t *testing.T) {
	clientRegistry := registry.NewClientRegistry()
	manager := NewControlManager(clientRegistry)
	metrics := newCountingServerMetrics()
	first, _ := newLifecycleTestControl(t, "run-one", "shared-client", metrics)
	conflicting, _ := newLifecycleTestControl(t, "run-two", "shared-client", metrics)

	mustAddAndActivate(t, manager, first)
	err := manager.Add(conflicting)
	require.NoError(t, err)
	active, err := manager.Activate(conflicting)
	require.True(t, active)
	require.ErrorContains(t, err, "already online")

	require.True(t, manager.Remove(conflicting))
	info, ok := clientRegistry.GetByKey("shared-client")
	require.True(t, ok)
	require.True(t, info.Online)
	require.Equal(t, "run-one", info.RunID)
}

func TestControlManagerFailedLoginWriteReleasesRunWithoutStarting(t *testing.T) {
	clientRegistry := registry.NewClientRegistry()
	manager := NewControlManager(clientRegistry)
	metrics := newCountingServerMetrics()
	ctl, _ := newLifecycleTestControl(t, "same-run", "client", metrics)
	replacement, _ := newLifecycleTestControl(t, "same-run", "client", metrics)

	mustAddAndActivate(t, manager, ctl)

	writeErr := errors.New("write failed")
	committed, err := manager.completeLogin(ctl, func() error { return writeErr })
	require.ErrorIs(t, err, writeErr)
	require.False(t, committed)

	err = manager.Add(replacement)
	require.NoError(t, err)
	waitForControlDone(t, ctl)
	require.Same(t, replacement, currentControlForTest(manager, "same-run"))
	require.Equal(t, int64(0), metrics.newClients())
	require.Equal(t, int64(0), metrics.closedClients())
	require.True(t, manager.Remove(replacement))
	info, ok := clientRegistry.GetByKey("client")
	require.True(t, ok)
	require.False(t, info.Online)
	require.Empty(t, info.RunID)
	require.Zero(t, info.ControlID)
	require.False(t, info.DisconnectedAt.IsZero())
	require.NoError(t, replacement.Close())
}

func TestControlManagerCloseWaitsForInFlightLoginRun(t *testing.T) {
	clientRegistry := registry.NewClientRegistry()
	manager := NewControlManager(clientRegistry)
	metrics := newCountingServerMetrics()
	ctl, _ := newLifecycleTestControl(t, "same-run", "client", metrics)

	mustAddAndActivate(t, manager, ctl)

	writeEntered := make(chan struct{})
	resumeWrite := make(chan struct{})
	loginDone := make(chan struct {
		committed bool
		err       error
	}, 1)
	go func() {
		committed, loginErr := manager.completeLogin(ctl, func() error {
			close(writeEntered)
			<-resumeWrite
			return nil
		})
		loginDone <- struct {
			committed bool
			err       error
		}{committed: committed, err: loginErr}
	}()
	waitForSignal(t, writeEntered, "LoginResp write")

	closeDone := make(chan error, 1)
	go func() { closeDone <- manager.Close() }()
	waitForManagerClosed(t, manager)
	select {
	case err := <-closeDone:
		t.Fatalf("manager close completed during LoginResp write: %v", err)
	default:
	}

	close(resumeWrite)
	result := <-loginDone
	require.NoError(t, result.err)
	require.True(t, result.committed)
	require.NoError(t, <-closeDone)
	waitForControlDone(t, ctl)
	require.Nil(t, currentControlForTest(manager, "same-run"))
	require.Equal(t, int64(1), metrics.newClients())
	require.Equal(t, int64(1), metrics.closedClients())
	info, ok := clientRegistry.GetByKey("client")
	require.True(t, ok)
	require.False(t, info.Online)
}

func newLifecycleTestControl(
	t *testing.T,
	runID string,
	clientID string,
	serverMetrics *countingServerMetrics,
) (*Control, *deadlineReadConn) {
	t.Helper()
	conn := newDeadlineReadConn()
	msgConn := msg.NewConn(conn, msg.NewV1ReadWriter(conn))
	ctl, err := NewControl(context.Background(), &SessionContext{
		RC:            &controller.ResourceController{},
		PxyManager:    proxy.NewManager(),
		PluginManager: plugin.NewManager(),
		AuthVerifier:  auth.AlwaysPassVerifier,
		Conn:          msgConn,
		LoginMsg: &msg.Login{
			RunID:    runID,
			ClientID: clientID,
		},
		ServerCfg: &v1.ServerConfig{},
	})
	require.NoError(t, err)
	ctl.serverMetrics = serverMetrics
	t.Cleanup(func() { _ = ctl.Close() })
	return ctl, conn
}

func mustAddAndActivate(t *testing.T, manager *ControlManager, ctl *Control) {
	t.Helper()
	require.NoError(t, manager.Add(ctl))
	active, err := manager.Activate(ctl)
	require.NoError(t, err)
	require.True(t, active)
}

func waitForControlDone(t *testing.T, ctl *Control) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		ctl.WaitClosed()
		close(done)
	}()
	waitForSignal(t, done, "control to finish")
}

func currentControlForTest(manager *ControlManager, runID string) *Control {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	entry := manager.ctlsByRunID[runID]
	if entry == nil {
		return nil
	}
	return entry.ctl
}

func currentRunGateForTest(manager *ControlManager, runID string) *sync.Mutex {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	entry := manager.ctlsByRunID[runID]
	if entry == nil {
		return nil
	}
	return entry.runMu
}

func waitForManagerClosed(t *testing.T, manager *ControlManager) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		manager.mu.RLock()
		closed := manager.closed
		manager.mu.RUnlock()
		if closed {
			return
		}
	}
	t.Fatal("timed out waiting for control manager to close")
}

func waitForSignal(t *testing.T, ch <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

type deadlineReadConn struct {
	readStarted chan struct{}
	unblockRead chan struct{}

	readOnce     sync.Once
	unblockOnce  sync.Once
	deadlineOnce sync.Once
	closeOnce    sync.Once

	eventsMu sync.Mutex
	events   []string
}

func newDeadlineReadConn() *deadlineReadConn {
	return &deadlineReadConn{
		readStarted: make(chan struct{}),
		unblockRead: make(chan struct{}),
	}
}

func (c *deadlineReadConn) Read([]byte) (int, error) {
	c.readOnce.Do(func() { close(c.readStarted) })
	<-c.unblockRead
	return 0, os.ErrDeadlineExceeded
}

func (*deadlineReadConn) Write(p []byte) (int, error) { return len(p), nil }

func (c *deadlineReadConn) Close() error {
	c.closeOnce.Do(func() {
		c.recordEvent("close")
		c.unblockOnce.Do(func() { close(c.unblockRead) })
	})
	return nil
}

func (*deadlineReadConn) LocalAddr() net.Addr  { return lifecycleTestAddr("local") }
func (*deadlineReadConn) RemoteAddr() net.Addr { return lifecycleTestAddr("remote") }

func (c *deadlineReadConn) SetDeadline(deadline time.Time) error {
	if err := c.SetReadDeadline(deadline); err != nil {
		return err
	}
	return c.SetWriteDeadline(deadline)
}

func (c *deadlineReadConn) SetReadDeadline(deadline time.Time) error {
	if deadline.IsZero() {
		return nil
	}
	c.deadlineOnce.Do(func() {
		c.recordEvent("deadline")
		c.unblockOnce.Do(func() { close(c.unblockRead) })
	})
	return nil
}

func (*deadlineReadConn) SetWriteDeadline(time.Time) error { return nil }

func (c *deadlineReadConn) recordEvent(event string) {
	c.eventsMu.Lock()
	c.events = append(c.events, event)
	c.eventsMu.Unlock()
}

func (c *deadlineReadConn) eventsSnapshot() []string {
	c.eventsMu.Lock()
	defer c.eventsMu.Unlock()
	return append([]string(nil), c.events...)
}

type lifecycleTestAddr string

func (a lifecycleTestAddr) Network() string { return string(a) }
func (a lifecycleTestAddr) String() string  { return string(a) }

type countingServerMetrics struct {
	mu          sync.Mutex
	newCount    int64
	closeCount  int64
	closeEnter  chan struct{}
	closeResume chan struct{}
	closeOnce   sync.Once
}

func newCountingServerMetrics() *countingServerMetrics {
	return &countingServerMetrics{}
}

func (m *countingServerMetrics) NewClient() {
	m.mu.Lock()
	m.newCount++
	m.mu.Unlock()
}

func (m *countingServerMetrics) CloseClient() {
	m.mu.Lock()
	m.closeCount++
	closeEnter := m.closeEnter
	closeResume := m.closeResume
	m.mu.Unlock()
	if closeEnter != nil {
		m.closeOnce.Do(func() { close(closeEnter) })
		<-closeResume
	}
}

func (*countingServerMetrics) NewProxy(string, string, string, string) {}
func (*countingServerMetrics) CloseProxy(string, string)               {}
func (*countingServerMetrics) OpenConnection(string, string)           {}
func (*countingServerMetrics) CloseConnection(string, string)          {}
func (*countingServerMetrics) AddTrafficIn(string, string, int64)      {}
func (*countingServerMetrics) AddTrafficOut(string, string, int64)     {}

func (m *countingServerMetrics) newClients() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.newCount
}

func (m *countingServerMetrics) closedClients() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCount
}

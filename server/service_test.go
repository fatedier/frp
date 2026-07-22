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
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fatedier/golib/net/mux"
	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/pkg/proto/wire"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/server/controller"
	"github.com/fatedier/frp/server/proxy"
	"github.com/fatedier/frp/server/registry"
	"github.com/fatedier/frp/server/visitor"
)

func TestWriteWithDeadlineTimesOutAndClearsDeadline(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	err := writeWithDeadline(serverConn, 50*time.Millisecond, func() error {
		_, writeErr := serverConn.Write([]byte("x"))
		return writeErr
	})
	require.Error(t, err)

	var netErr net.Error
	require.True(t, errors.As(err, &netErr))
	require.True(t, netErr.Timeout())

	readCh := make(chan byte, 1)
	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 1)
		if _, readErr := clientConn.Read(buf); readErr != nil {
			errCh <- readErr
			return
		}
		readCh <- buf[0]
	}()

	_, err = serverConn.Write([]byte("y"))
	require.NoError(t, err)

	select {
	case b := <-readCh:
		require.Equal(t, byte('y'), b)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for write after deadline reset")
	}
}

func TestSharedPortHTTPListenerProtocols(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	sharedMux := mux.NewMux(listener)
	httpListener := sharedMux.ListenHTTP(1)
	muxServeErr := make(chan error, 1)
	go func() {
		muxServeErr <- sharedMux.Serve()
	}()

	newProtocols := func(http1, unencryptedHTTP2 bool) *http.Protocols {
		protocols := new(http.Protocols)
		protocols.SetHTTP1(http1)
		protocols.SetUnencryptedHTTP2(unencryptedHTTP2)
		return protocols
	}

	const handlerProtocolHeader = "X-Test-Handler-Protocol"
	httpServer := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(handlerProtocolHeader, r.Proto)
			w.WriteHeader(http.StatusNoContent)
		}),
		ReadHeaderTimeout: time.Second,
		Protocols:         newProtocols(true, true),
	}
	httpServeErr := make(chan error, 1)
	go func() {
		httpServeErr <- httpServer.Serve(httpListener)
	}()
	t.Cleanup(func() {
		require.NoError(t, httpServer.Close())
		require.ErrorIs(t, waitForResult(t, httpServeErr, "shared HTTP server to stop"), http.ErrServerClosed)
		require.NoError(t, sharedMux.Close())
		require.ErrorIs(t, waitForResult(t, muxServeErr, "shared mux to stop"), net.ErrClosed)
	})

	for _, tc := range []struct {
		name             string
		http1            bool
		unencryptedHTTP2 bool
		expectedProtocol string
	}{
		{name: "HTTP/1.1", http1: true, expectedProtocol: "HTTP/1.1"},
		{name: "HTTP/2 prior knowledge", unencryptedHTTP2: true, expectedProtocol: "HTTP/2.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := &http.Transport{
				Protocols: newProtocols(tc.http1, tc.unencryptedHTTP2),
			}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+listener.Addr().String()+"/", nil)
			require.NoError(t, err)
			response, err := client.Do(request)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, response.StatusCode)
			require.Equal(t, tc.expectedProtocol, response.Proto)
			require.Equal(t, tc.expectedProtocol, response.Header.Get(handlerProtocolHeader))
			require.NoError(t, response.Body.Close())
		})
	}
}

func TestServiceControlHandoffSkipsStalePendingGeneration(t *testing.T) {
	svr := newControlTestService(t)
	metrics := newCountingServerMetrics()
	metrics.closeEnter = make(chan struct{})
	metrics.closeResume = make(chan struct{})

	ctlA, connA, err := registerLifecycleTestControl(svr)
	require.NoError(t, err)
	ctlA.serverMetrics = metrics
	require.NoError(t, svr.completeControlLogin(ctlA, func() error { return nil }))
	waitForSignal(t, connA.readStarted, "A reader to start")

	require.NoError(t, ctlA.Close())
	waitForSignal(t, metrics.closeEnter, "A finalization barrier")

	type registerResult struct {
		ctl  *Control
		conn *deadlineReadConn
		err  error
	}
	resultB := make(chan registerResult, 1)
	go func() {
		ctl, conn, registerErr := registerLifecycleTestControl(svr)
		resultB <- registerResult{ctl: ctl, conn: conn, err: registerErr}
	}()
	ctlB := waitForDifferentCurrentControl(t, svr.ctlManager, "shared-run", ctlA)
	ctlB.serverMetrics = metrics

	resultC := make(chan registerResult, 1)
	go func() {
		ctl, conn, registerErr := registerLifecycleTestControl(svr)
		resultC <- registerResult{ctl: ctl, conn: conn, err: registerErr}
	}()
	ctlC := waitForDifferentCurrentControl(t, svr.ctlManager, "shared-run", ctlB)
	ctlC.serverMetrics = metrics
	waitForControlDone(t, ctlB)

	select {
	case result := <-resultB:
		t.Fatalf("B returned before A finalized: %v", result.err)
	default:
	}
	select {
	case result := <-resultC:
		t.Fatalf("C returned before A finalized: %v", result.err)
	default:
	}

	close(metrics.closeResume)
	waitForControlDone(t, ctlA)

	b := <-resultB
	require.Same(t, ctlB, b.ctl)
	require.ErrorIs(t, b.err, errControlReplaced)
	require.False(t, svr.ctlManager.Remove(ctlB))
	require.NoError(t, ctlB.Close())

	c := <-resultC
	require.NoError(t, c.err)
	require.Same(t, ctlC, c.ctl)
	_, ok := svr.ctlManager.GetByID("shared-run")
	require.False(t, ok)
	require.Same(t, ctlC, currentControlForTest(svr.ctlManager, "shared-run"))

	info, ok := svr.clientRegistry.GetByKey("client")
	require.True(t, ok)
	require.True(t, info.Online)
	require.Equal(t, uint64(ctlC.ID()), info.ControlID)

	var staleWrites atomic.Int64
	err = svr.completeControlLogin(ctlB, func() error {
		staleWrites.Add(1)
		return nil
	})
	require.ErrorIs(t, err, errControlReplaced)
	require.Equal(t, int64(0), staleWrites.Load())

	require.NoError(t, svr.completeControlLogin(ctlC, func() error { return nil }))
	waitForSignal(t, c.conn.readStarted, "C reader to start")
	current, ok := svr.ctlManager.GetByID("shared-run")
	require.True(t, ok)
	require.Same(t, ctlC, current)
	require.Equal(t, int64(2), metrics.newClients())
	require.Equal(t, int64(1), metrics.closedClients())

	require.NoError(t, ctlC.Close())
	waitForControlDone(t, ctlC)
	require.Equal(t, int64(2), metrics.newClients())
	require.Equal(t, int64(2), metrics.closedClients())
	_, ok = svr.ctlManager.GetByID("shared-run")
	require.False(t, ok)
}

func TestServiceLoginResponseSynchronizationIsScopedToRun(t *testing.T) {
	svr := newControlTestService(t)
	metrics := newCountingServerMetrics()
	ctlA, connA, err := registerLifecycleTestControl(svr)
	require.NoError(t, err)
	ctlA.serverMetrics = metrics

	writeEntered := make(chan struct{})
	resumeWrite := make(chan struct{})
	var resumeWriteOnce sync.Once
	resume := func() {
		resumeWriteOnce.Do(func() { close(resumeWrite) })
	}
	t.Cleanup(resume)
	writeCount := atomic.Int64{}
	loginDone := make(chan error, 1)
	go func() {
		loginDone <- svr.completeControlLogin(ctlA, func() error {
			close(writeEntered)
			<-resumeWrite
			writeCount.Add(1)
			return nil
		})
	}()
	waitForSignal(t, writeEntered, "A LoginResp write")

	runMu := currentRunGateForTest(svr.ctlManager, "shared-run")
	require.NotNil(t, runMu)
	if !svr.ctlManager.mu.TryLock() {
		t.Fatal("ControlManager mutex was held while LoginResp write was in progress")
	}
	svr.ctlManager.mu.Unlock()

	ctlB, connB := newLifecycleTestControl(t, "shared-run", "client", metrics)
	gateAvailable := make(chan bool)
	addDone := make(chan error, 1)
	go func() {
		if runMu.TryLock() {
			runMu.Unlock()
			gateAvailable <- true
		} else {
			gateAvailable <- false
		}
		addErr := svr.ctlManager.Add(ctlB)
		addDone <- addErr
	}()
	available := waitForResult(t, gateAvailable, "same-run replacement gate probe")
	require.False(t, available, "same-run gate was available to replacement during LoginResp write")
	select {
	case addErr := <-addDone:
		t.Fatalf("same-run replacement completed during LoginResp write: %v", addErr)
	case <-time.After(20 * time.Millisecond):
	}
	require.Same(t, ctlA, currentControlForTest(svr.ctlManager, "shared-run"))

	otherMetrics := newCountingServerMetrics()
	otherCtl, otherConn := newLifecycleTestControl(t, "other-run", "other-client", otherMetrics)
	type unrelatedResult struct {
		addErr      error
		active      bool
		activateErr error
		loginErr    error
		current     *Control
		found       bool
	}
	unrelatedDone := make(chan unrelatedResult, 1)
	go func() {
		result := unrelatedResult{}
		result.addErr = svr.ctlManager.Add(otherCtl)
		if result.addErr == nil {
			result.active, result.activateErr = svr.ctlManager.Activate(otherCtl)
		}
		if result.activateErr == nil && result.active {
			result.loginErr = svr.completeControlLogin(otherCtl, func() error { return nil })
		}
		result.current, result.found = svr.ctlManager.GetByID("other-run")
		unrelatedDone <- result
	}()
	result := waitForResult(t, unrelatedDone, "unrelated run lifecycle")
	require.NoError(t, result.addErr)
	require.NoError(t, result.activateErr)
	require.True(t, result.active)
	require.NoError(t, result.loginErr)
	require.True(t, result.found)
	require.Same(t, otherCtl, result.current)
	waitForSignal(t, otherConn.readStarted, "unrelated control reader to start")
	require.Equal(t, int64(1), otherMetrics.newClients())

	resume()
	require.NoError(t, waitForResult(t, loginDone, "LoginResp completion"))
	require.NoError(t, waitForResult(t, addDone, "replacement"))
	waitForControlDone(t, ctlA)
	require.Same(t, ctlB, currentControlForTest(svr.ctlManager, "shared-run"))
	require.Equal(t, int64(1), writeCount.Load())
	require.Equal(t, int64(1), metrics.newClients())
	require.Equal(t, int64(1), metrics.closedClients())
	require.Equal(t, []string{"deadline", "close"}, connA.eventsSnapshot())

	require.False(t, svr.ctlManager.Remove(ctlA))
	require.NoError(t, ctlA.Close())
	require.True(t, svr.ctlManager.Remove(ctlB))
	require.NoError(t, ctlB.Close())
	require.Equal(t, []string{"deadline", "close"}, connB.eventsSnapshot())

	require.NoError(t, otherCtl.Close())
	waitForControlDone(t, otherCtl)
	require.Equal(t, int64(1), otherMetrics.newClients())
	require.Equal(t, int64(1), otherMetrics.closedClients())
}

func TestServiceVisitorAdmissionSerializesReplacement(t *testing.T) {
	svr := newControlTestService(t)
	ctlA, controlConn, err := registerLifecycleTestControl(svr)
	require.NoError(t, err)
	ctlA.sessionCtx.LoginMsg.User = "old-user"
	require.NoError(t, svr.completeControlLogin(ctlA, func() error { return nil }))
	waitForSignal(t, controlConn.readStarted, "A reader to start")

	admissionEntered := make(chan struct{})
	resumeAdmission := make(chan struct{})
	var resumeOnce sync.Once
	resume := func() {
		resumeOnce.Do(func() { close(resumeAdmission) })
	}
	t.Cleanup(resume)
	type admissionResult struct {
		admitted bool
		user     string
		err      error
	}
	admissionDone := make(chan admissionResult, 1)
	go func() {
		var admittedUser string
		admitted, admitErr := svr.ctlManager.admitVisitorByRunID("shared-run", func(user string) error {
			admittedUser = user
			close(admissionEntered)
			<-resumeAdmission
			return nil
		})
		admissionDone <- admissionResult{admitted: admitted, user: admittedUser, err: admitErr}
	}()
	waitForSignal(t, admissionEntered, "visitor admission callback")
	runMu := currentRunGateForTest(svr.ctlManager, "shared-run")
	require.NotNil(t, runMu)

	type registerResult struct {
		ctl *Control
		err error
	}
	gateAvailable := make(chan bool)
	replacementDone := make(chan registerResult, 1)
	go func() {
		if runMu.TryLock() {
			runMu.Unlock()
			gateAvailable <- true
		} else {
			gateAvailable <- false
		}
		ctl, _, registerErr := registerLifecycleTestControl(svr)
		replacementDone <- registerResult{ctl: ctl, err: registerErr}
	}()
	available := waitForResult(t, gateAvailable, "visitor replacement gate probe")
	require.False(t, available, "same-run gate was available during visitor admission")
	select {
	case result := <-replacementDone:
		t.Fatalf("replacement completed during visitor admission: %v", result.err)
	case <-time.After(20 * time.Millisecond):
	}
	require.Same(t, ctlA, currentControlForTest(svr.ctlManager, "shared-run"))

	resume()
	admission := waitForResult(t, admissionDone, "visitor admission")
	require.NoError(t, admission.err)
	require.True(t, admission.admitted)
	require.Equal(t, "old-user", admission.user)
	replacement := waitForResult(t, replacementDone, "replacement")
	require.NoError(t, replacement.err)
	ctlB := replacement.ctl
	require.Same(t, ctlB, currentControlForTest(svr.ctlManager, "shared-run"))
	waitForControlDone(t, ctlA)
	require.True(t, svr.ctlManager.Remove(ctlB))
	require.NoError(t, ctlB.Close())
}

func TestServiceWorkConnRoutingRequiresCurrentRunningControl(t *testing.T) {
	svr := newControlTestService(t)
	ctl, controlConn, err := registerLifecycleTestControl(svr)
	require.NoError(t, err)

	pendingConn := newCountingCloseConn()
	pendingMsgConn := msg.NewConn(pendingConn, msg.NewV1ReadWriter(pendingConn))
	err = registerWorkConnAsCaller(svr, pendingMsgConn, &msg.NewWorkConn{RunID: "shared-run"})
	require.Error(t, err)
	require.Equal(t, int64(1), pendingConn.closeCount.Load())
	require.Len(t, ctl.workConnCh, 0)

	require.NoError(t, svr.completeControlLogin(ctl, func() error { return nil }))
	waitForSignal(t, controlConn.readStarted, "control reader to start")
	current, ok := svr.ctlManager.GetByID("shared-run")
	require.True(t, ok)
	require.Same(t, ctl, current)
	require.Len(t, ctl.workConnCh, 0)

	runningConn := newCountingCloseConn()
	runningMsgConn := msg.NewConn(runningConn, msg.NewV1ReadWriter(runningConn))
	require.NoError(t, svr.RegisterWorkConn(runningMsgConn, &msg.NewWorkConn{RunID: "shared-run"}))
	require.Len(t, ctl.workConnCh, 1)

	require.NoError(t, ctl.Close())
	waitForControlDone(t, ctl)
	require.Equal(t, int64(1), runningConn.closeCount.Load())
}

func TestServiceWorkConnRoutingRejectsLostGeneration(t *testing.T) {
	for _, action := range []string{"replace", "close"} {
		t.Run(action, func(t *testing.T) {
			svr := newControlTestService(t)
			ctl, controlConn, err := registerLifecycleTestControl(svr)
			require.NoError(t, err)
			require.NoError(t, svr.completeControlLogin(ctl, func() error { return nil }))
			waitForSignal(t, controlConn.readStarted, "control reader to start")

			barrier := newWorkConnBarrierPlugin()
			svr.pluginManager.Register(barrier)
			workConn := newCountingCloseConn()
			workMsgConn := msg.NewConn(workConn, msg.NewV1ReadWriter(workConn))
			routeDone := make(chan error, 1)
			go func() {
				routeDone <- registerWorkConnAsCaller(svr, workMsgConn, &msg.NewWorkConn{RunID: "shared-run"})
			}()
			waitForSignal(t, barrier.entered, "work connection plugin barrier")

			var replacement *Control
			switch action {
			case "replace":
				replacement, _, err = registerLifecycleTestControl(svr)
				require.NoError(t, err)
			case "close":
				require.NoError(t, ctl.Close())
				waitForControlDone(t, ctl)
			}

			close(barrier.resume)
			require.Error(t, waitForResult(t, routeDone, "work connection route to finish"))
			require.Equal(t, int64(1), workConn.closeCount.Load())
			require.Len(t, ctl.workConnCh, 0)

			if replacement != nil {
				require.Len(t, replacement.workConnCh, 0)
				require.True(t, svr.ctlManager.Remove(replacement))
				require.NoError(t, replacement.Close())
				waitForControlDone(t, replacement)
			}
		})
	}
}

func TestServiceVisitorRoutingExcludesPendingUser(t *testing.T) {
	svr := newControlTestService(t)
	listener, err := svr.rc.VisitorManager.Listen("visitor", "secret", []string{"pending-user"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	controlConn := newDeadlineReadConn()
	controlMsgConn := msg.NewConn(controlConn, msg.NewV1ReadWriter(controlConn))
	ctl, err := svr.RegisterControl(controlMsgConn, &msg.Login{
		RunID:    "visitor-run",
		User:     "pending-user",
		ClientID: "visitor-client",
		ClientSpec: msg.ClientSpec{
			AlwaysAuthPass: true,
		},
	}, true, wire.ProtocolV1)
	require.NoError(t, err)

	timestamp := time.Now().Unix()
	visitorMsg := &msg.NewVisitorConn{
		RunID:     "visitor-run",
		ProxyName: "visitor",
		Timestamp: timestamp,
		SignKey:   util.GetAuthKey("secret", timestamp),
	}
	pendingConn := newCountingCloseConn()
	err = svr.RegisterVisitorConn(pendingConn, visitorMsg, wire.ProtocolV1)
	require.ErrorContains(t, err, "no client control found")
	require.NoError(t, pendingConn.Close())
	require.Equal(t, int64(1), pendingConn.closeCount.Load())

	require.NoError(t, svr.completeControlLogin(ctl, func() error { return nil }))
	waitForSignal(t, controlConn.readStarted, "control reader to start")
	runningConn := newCountingCloseConn()
	require.NoError(t, svr.RegisterVisitorConn(runningConn, visitorMsg, wire.ProtocolV1))
	accepted, err := listener.Accept()
	require.NoError(t, err)
	require.NoError(t, accepted.Close())
	require.Equal(t, int64(1), runningConn.closeCount.Load())

	require.NoError(t, ctl.Close())
	waitForControlDone(t, ctl)
}

func newControlTestService(t *testing.T) *Service {
	t.Helper()
	cfg := &v1.ServerConfig{}
	cfg.Auth.Method = v1.AuthMethodToken
	authRuntime, err := auth.BuildServerAuth(&cfg.Auth)
	require.NoError(t, err)
	clientRegistry := registry.NewClientRegistry()
	return &Service{
		ctlManager:     NewControlManager(clientRegistry),
		clientRegistry: clientRegistry,
		pxyManager:     proxy.NewManager(),
		pluginManager:  plugin.NewManager(),
		rc: &controller.ResourceController{
			VisitorManager: visitor.NewManager(),
		},
		auth: authRuntime,
		cfg:  cfg,
	}
}

func registerLifecycleTestControl(svr *Service) (*Control, *deadlineReadConn, error) {
	conn := newDeadlineReadConn()
	msgConn := msg.NewConn(conn, msg.NewReadWriter(conn, wire.ProtocolV1))
	ctl, err := svr.RegisterControl(msgConn, &msg.Login{
		RunID:    "shared-run",
		ClientID: "client",
		ClientSpec: msg.ClientSpec{
			AlwaysAuthPass: true,
		},
	}, true, wire.ProtocolV1)
	return ctl, conn, err
}

func waitForDifferentCurrentControl(t *testing.T, manager *ControlManager, runID string, old *Control) *Control {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ctl := currentControlForTest(manager, runID); ctl != nil && ctl != old {
			return ctl
		}
		runtime.Gosched()
	}
	t.Fatalf("timed out waiting for a new current control after ID %d", old.ID())
	return nil
}

func registerWorkConnAsCaller(svr *Service, workConn *msg.Conn, newMsg *msg.NewWorkConn) error {
	err := svr.RegisterWorkConn(workConn, newMsg)
	if err != nil {
		_ = workConn.Close()
	}
	return err
}

func waitForResult[T any](t *testing.T, ch <-chan T, description string) T {
	t.Helper()
	select {
	case result := <-ch:
		return result
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
		var zero T
		return zero
	}
}

type workConnBarrierPlugin struct {
	entered chan struct{}
	resume  chan struct{}
}

func newWorkConnBarrierPlugin() *workConnBarrierPlugin {
	return &workConnBarrierPlugin{
		entered: make(chan struct{}),
		resume:  make(chan struct{}),
	}
}

func (*workConnBarrierPlugin) Name() string { return "work-conn-barrier" }

func (*workConnBarrierPlugin) IsSupport(op string) bool { return op == plugin.OpNewWorkConn }

func (p *workConnBarrierPlugin) Handle(
	context.Context,
	string,
	any,
) (*plugin.Response, any, error) {
	close(p.entered)
	<-p.resume
	return &plugin.Response{Unchange: true}, nil, nil
}

type countingCloseConn struct {
	closeCount atomic.Int64
}

func newCountingCloseConn() *countingCloseConn { return &countingCloseConn{} }

func (*countingCloseConn) Read([]byte) (int, error)         { return 0, net.ErrClosed }
func (*countingCloseConn) Write(p []byte) (int, error)      { return len(p), nil }
func (c *countingCloseConn) Close() error                   { c.closeCount.Add(1); return nil }
func (*countingCloseConn) LocalAddr() net.Addr              { return lifecycleTestAddr("local") }
func (*countingCloseConn) RemoteAddr() net.Addr             { return lifecycleTestAddr("remote") }
func (*countingCloseConn) SetDeadline(time.Time) error      { return nil }
func (*countingCloseConn) SetReadDeadline(time.Time) error  { return nil }
func (*countingCloseConn) SetWriteDeadline(time.Time) error { return nil }

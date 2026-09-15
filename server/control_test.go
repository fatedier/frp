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
	"fmt"
	"math"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	pkgerr "github.com/fatedier/frp/pkg/errors"
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

func TestNewControlPoolCountBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name          string
		poolCount     int
		maxPoolCount  int64
		wantErr       string
		wantPoolCount int
		wantCapacity  int
	}{
		{name: "negative pool count below offset", poolCount: -11, maxPoolCount: 5, wantErr: "invalid pool count"},
		{name: "negative pool count at offset", poolCount: -10, maxPoolCount: 5, wantErr: "invalid pool count"},
		{name: "negative pool count", poolCount: -1, maxPoolCount: 5, wantErr: "invalid pool count"},
		{name: "zero pool count", poolCount: 0, maxPoolCount: 5, wantPoolCount: 0, wantCapacity: 10},
		{name: "pool count capped", poolCount: 10, maxPoolCount: 5, wantPoolCount: 5, wantCapacity: 15},
		{name: "maximum int pool count capped", poolCount: math.MaxInt, maxPoolCount: 5, wantPoolCount: 5, wantCapacity: 15},
		{name: "negative maximum", poolCount: 1, maxPoolCount: -1, wantErr: "invalid max pool count"},
		{name: "maximum int64 with small client pool", poolCount: 1, maxPoolCount: math.MaxInt64, wantPoolCount: 1, wantCapacity: 11},
		{name: "maximum int client and server overflow", poolCount: math.MaxInt, maxPoolCount: math.MaxInt64, wantErr: "cannot safely add"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := newDeadlineReadConn()
			msgConn := msg.NewConn(conn, msg.NewV1ReadWriter(conn))
			cfg := &v1.ServerConfig{}
			cfg.Transport.MaxPoolCount = tc.maxPoolCount

			ctl, err := NewControl(context.Background(), &SessionContext{
				RC:            &controller.ResourceController{},
				PxyManager:    proxy.NewManager(),
				PluginManager: plugin.NewManager(),
				AuthVerifier:  auth.AlwaysPassVerifier,
				Conn:          msgConn,
				LoginMsg: &msg.Login{
					RunID:     "pool-count-run",
					PoolCount: tc.poolCount,
				},
				ServerCfg: cfg,
			})
			if tc.wantErr != "" {
				require.Nil(t, ctl)
				require.ErrorContains(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantPoolCount, ctl.poolCount)
			require.Equal(t, tc.wantCapacity, cap(ctl.workConnCh))
			require.NoError(t, ctl.Close())
		})
	}
}

func TestControlReserveRefillUsesSlidingWindow(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 20)
	ctl.workConnLeaseTimeout = 100 * time.Millisecond
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	ctl.reconcileWorkConnRequests()
	for range workConnReserveWindowMax {
		req := readWorkConnRequest(t, clientMsgConn)
		require.Equal(t, msg.WorkConnTypeReserve, req.WorkConnType)
	}

	for i := range 20 {
		conn := newCountingCloseConn()
		workConn := proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn)))
		accepted, err := ctl.registerWorkConn(workConn, workConnRequestReserve)
		require.NoError(t, err)
		require.True(t, accepted)
		if i < 12 {
			req := readWorkConnRequest(t, clientMsgConn)
			require.Equal(t, msg.WorkConnTypeReserve, req.WorkConnType)
		}
	}
	require.Len(t, ctl.workConnCh, 20)
}

func TestControlDemandRefillPrecedesReserve(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 2)
	ctl.controlID = 7
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	resultCh := make(chan workConnResult, 1)
	go func() {
		conn, err := ctl.GetWorkConn()
		resultCh <- workConnResult{conn: conn, err: err}
	}()

	demandReq := readWorkConnRequest(t, clientMsgConn)
	require.Equal(t, msg.WorkConnTypeDemand, demandReq.WorkConnType)
	require.Equal(t, uint64(7), demandReq.ControlID)
	require.Equal(t, msg.WorkConnTypeReserve, readWorkConnRequest(t, clientMsgConn).WorkConnType)
	require.Equal(t, msg.WorkConnTypeReserve, readWorkConnRequest(t, clientMsgConn).WorkConnType)

	conn := newCountingCloseConn()
	accepted, err := ctl.registerWorkConn(
		proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
		workConnRequestDemand,
	)
	require.NoError(t, err)
	require.True(t, accepted)
	result := <-resultCh
	require.NoError(t, result.err)
	require.NotNil(t, result.conn)
}

func TestControlLegacySchedulingCapsMixedDemandAndReserveWindow(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 20)
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	// Startup reserve requests consume part of the legacy aggregate window.
	ctl.reconcileWorkConnRequests()
	for range workConnReserveWindowMax {
		require.Equal(t, msg.WorkConnTypeReserve, readWorkConnRequest(t, clientMsgConn).WorkConnType)
	}

	var firstWaiter *workConnWaiter
	ctl.workConnMu.Lock()
	for range 128 {
		waiter := &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
		ctl.appendWorkConnWaiterLocked(waiter)
		if firstWaiter == nil {
			firstWaiter = waiter
		}
	}
	ctl.workConnMu.Unlock()

	ctl.reconcileWorkConnRequests()
	ctl.workConnMu.Lock()
	legacyMode := ctl.workConnLegacyMode
	demandPending := len(ctl.workConnPending[workConnRequestDemand])
	reservePending := len(ctl.workConnPending[workConnRequestReserve])
	ctl.workConnMu.Unlock()
	require.True(t, legacyMode)
	require.Equal(t, workConnLegacyWindow-workConnReserveWindowMax, demandPending)
	require.Equal(t, workConnReserveWindowMax, reservePending)

	// A legacy response keeps the aggregate in-flight count bounded while
	// refilling demand for the remaining waiters.
	conn := newCountingCloseConn()
	accepted, err := ctl.registerWorkConn(
		proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
		workConnRequestLegacy,
	)
	require.NoError(t, err)
	require.True(t, accepted)
	result := <-firstWaiter.resultCh
	require.NotNil(t, result.conn)
	_ = result.conn.Close()
	ctl.workConnMu.Lock()
	pending := len(ctl.workConnPending[workConnRequestDemand]) + len(ctl.workConnPending[workConnRequestReserve])
	ctl.workConnMu.Unlock()
	require.LessOrEqual(t, pending, workConnLegacyWindow)
}

func TestControlLegacyResponsesDeliverConcurrentWaiters(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 20)
	ctl.workConnLeaseTimeout = time.Second
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	// The net.Pipe peer must keep draining ReqWorkConn messages so the real
	// dispatcher send loop can exercise refill pacing without blocking.
	go func() {
		for {
			if _, err := clientMsgConn.ReadMsg(); err != nil {
				return
			}
		}
	}()

	results := make(chan workConnResult, 128)
	for i := 0; i < cap(results); i++ {
		go func() {
			conn, err := ctl.GetWorkConn()
			results <- workConnResult{conn: conn, err: err}
		}()
	}

	deadline := time.Now().Add(time.Second)
	for {
		ctl.workConnMu.Lock()
		waiters := ctl.activeWaiterCountLocked()
		demandPending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestDemand)
		reservePending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestReserve)
		legacyMode := ctl.workConnLegacyMode
		ctl.workConnMu.Unlock()
		if waiters == cap(results) {
			require.True(t, legacyMode)
			require.LessOrEqual(t, demandPending+reservePending, workConnLegacyWindow)
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d work connection waiters, got %d", cap(results), waiters)
		}
		time.Sleep(time.Millisecond)
	}

	for i := 0; i < cap(results); i++ {
		conn := newCountingCloseConn()
		accepted, err := ctl.registerWorkConn(
			proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
			workConnRequestLegacy,
		)
		require.NoError(t, err)
		require.True(t, accepted)
	}

	for i := 0; i < cap(results); i++ {
		result := <-results
		require.NoError(t, result.err)
		require.NotNil(t, result.conn)
		require.NoError(t, result.conn.Close())
	}
}

func TestControlTypedReserveResponsesDoNotOvershootPool(t *testing.T) {
	const (
		poolCount   = 10
		waiterCount = 100
	)
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, poolCount)
	ctl.workConnLeaseTimeout = time.Minute
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	go func() {
		for {
			if _, err := clientMsgConn.ReadMsg(); err != nil {
				return
			}
		}
	}()

	waiters := make([]*workConnWaiter, 0, waiterCount)
	ctl.workConnMu.Lock()
	ctl.workConnLegacyMode = false
	for range waiterCount {
		waiter := &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
		waiters = append(waiters, waiter)
		ctl.appendWorkConnWaiterLocked(waiter)
	}
	ctl.workConnMu.Unlock()
	ctl.reconcileWorkConnRequests()

	ctl.workConnMu.Lock()
	demandPending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestDemand)
	reservePending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestReserve)
	lateKinds := len(ctl.workConnLate)
	ctl.workConnMu.Unlock()
	require.Equal(t, workConnDemandWindow, demandPending)
	require.Equal(t, workConnReserveWindowMax, reservePending)
	require.Zero(t, lateKinds)

	registerResponse := func(kind workConnRequestKind) bool {
		conn := newCountingCloseConn()
		accepted, err := ctl.registerWorkConn(
			proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
			kind,
		)
		if accepted {
			require.NoError(t, err)
			return true
		}
		require.ErrorIs(t, err, errWorkConnPoolFull)
		require.NoError(t, conn.Close())
		return false
	}

	reserveResponses := 0
	for {
		ctl.workConnMu.Lock()
		activeWaiters := ctl.activeWaiterCountLocked()
		reservePending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestReserve)
		ctl.workConnMu.Unlock()
		if activeWaiters == 0 || reservePending == 0 {
			break
		}
		require.True(t, registerResponse(workConnRequestReserve))
		reserveResponses++
		require.LessOrEqual(t, reserveResponses, waiterCount)
	}

	discarded := 0
	totalResponses := reserveResponses
	for totalResponses <= 3*(waiterCount+poolCount) {
		ctl.workConnMu.Lock()
		demandSupply := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestDemand) +
			ctl.leaseCountLocked(ctl.workConnLate, workConnRequestDemand)
		reserveSupply := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestReserve) +
			ctl.leaseCountLocked(ctl.workConnLate, workConnRequestReserve)
		ctl.workConnMu.Unlock()
		if demandSupply+reserveSupply == 0 {
			break
		}
		kind := workConnRequestReserve
		if demandSupply > 0 {
			kind = workConnRequestDemand
		}
		if !registerResponse(kind) {
			discarded++
		}
		totalResponses++
	}

	require.Equal(t, waiterCount+poolCount, totalResponses, "each requested connection should contribute exactly once to waiter demand or idle reserve")
	require.Equal(t, waiterCount+poolCount-workConnDemandWindow, reserveResponses)
	require.Zero(t, discarded)
	require.Len(t, ctl.workConnCh, poolCount)
	ctl.workConnMu.Lock()
	activeWaiters := ctl.activeWaiterCountLocked()
	pendingKinds := len(ctl.workConnPending)
	lateKinds = len(ctl.workConnLate)
	ctl.workConnMu.Unlock()
	require.Zero(t, activeWaiters)
	require.Zero(t, pendingKinds)
	require.Zero(t, lateKinds)

	for _, waiter := range waiters {
		result := <-waiter.resultCh
		require.NoError(t, result.err)
		require.NotNil(t, result.conn)
		require.NoError(t, result.conn.Close())
	}
}

func TestControlLegacyResponsesKeepExactSupply(t *testing.T) {
	const (
		poolCount   = 10
		waiterCount = 100
	)
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, poolCount)
	ctl.workConnLeaseTimeout = time.Minute
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	go func() {
		for {
			if _, err := clientMsgConn.ReadMsg(); err != nil {
				return
			}
		}
	}()

	waiters := make([]*workConnWaiter, 0, waiterCount)
	ctl.workConnMu.Lock()
	for range waiterCount {
		waiter := &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
		waiters = append(waiters, waiter)
		ctl.appendWorkConnWaiterLocked(waiter)
	}
	ctl.workConnMu.Unlock()
	ctl.reconcileWorkConnRequests()

	ctl.workConnMu.Lock()
	legacyMode := ctl.workConnLegacyMode
	pending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestDemand) +
		ctl.leaseCountLocked(ctl.workConnPending, workConnRequestReserve)
	ctl.workConnMu.Unlock()
	require.True(t, legacyMode)
	require.Equal(t, workConnLegacyWindow, pending)

	for range waiterCount + poolCount {
		conn := newCountingCloseConn()
		accepted, err := ctl.registerWorkConn(
			proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
			workConnRequestLegacy,
		)
		require.NoError(t, err)
		require.True(t, accepted)
	}

	require.Len(t, ctl.workConnCh, poolCount)
	ctl.workConnMu.Lock()
	activeWaiters := ctl.activeWaiterCountLocked()
	pendingKinds := len(ctl.workConnPending)
	lateKinds := len(ctl.workConnLate)
	ctl.workConnMu.Unlock()
	require.Zero(t, activeWaiters)
	require.Zero(t, pendingKinds)
	require.Zero(t, lateKinds)

	for _, waiter := range waiters {
		result := <-waiter.resultCh
		require.NoError(t, result.err)
		require.NotNil(t, result.conn)
		require.NoError(t, result.conn.Close())
	}
}

func TestControlReserveTimeoutReleasesWindowBudget(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 4)
	ctl.workConnLeaseTimeout = time.Minute
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	ctl.reconcileWorkConnRequests()
	for range 4 {
		require.Equal(t, msg.WorkConnTypeReserve, readWorkConnRequest(t, clientMsgConn).WorkConnType)
	}

	// Move the four dispatched leases through the first timeout transition
	// deterministically, then let reconcile create their replacement budget.
	for range 4 {
		moveReserveWorkConnLeaseToLateForTest(t, ctl)
	}
	ctl.reconcileWorkConnRequests()
	for range 4 {
		require.Equal(t, msg.WorkConnTypeReserve, readWorkConnRequest(t, clientMsgConn).WorkConnType)
	}

	conn := newCountingCloseConn()
	accepted, err := ctl.registerWorkConn(
		proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
		workConnRequestReserve,
	)
	require.True(t, accepted)
	require.NoError(t, err)
	require.Len(t, ctl.workConnCh, 1)
	ctl.workConnMu.Lock()
	pending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestReserve)
	ctl.workConnMu.Unlock()
	require.Equal(t, 3, pending)
}

func TestControlLateReserveResponseRejectedWhenReplacementCoversPool(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 1)
	ctl.workConnLeaseTimeout = time.Minute
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	ctl.reconcileWorkConnRequests()
	require.Equal(t, msg.WorkConnTypeReserve, readWorkConnRequest(t, clientMsgConn).WorkConnType)
	moveReserveWorkConnLeaseToLateForTest(t, ctl)
	ctl.reconcileWorkConnRequests()
	require.Equal(t, msg.WorkConnTypeReserve, readWorkConnRequest(t, clientMsgConn).WorkConnType)

	// The replacement fills the pool even when the original never responded.
	replacementConn := newCountingCloseConn()
	accepted, err := ctl.registerWorkConn(
		proxy.NewWorkConn(msg.NewConn(replacementConn, msg.NewV1ReadWriter(replacementConn))),
		workConnRequestReserve,
	)
	require.True(t, accepted)
	require.NoError(t, err)
	require.Len(t, ctl.workConnCh, 1)

	// If the original eventually arrives, it cannot grow an already full pool.
	conn := newCountingCloseConn()
	accepted, err = ctl.registerWorkConn(
		proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
		workConnRequestReserve,
	)
	require.False(t, accepted)
	require.ErrorIs(t, err, errWorkConnPoolFull)
	require.NoError(t, conn.Close())
	ctl.workConnMu.Lock()
	late := len(ctl.workConnLate[workConnRequestReserve])
	pending := len(ctl.workConnPending[workConnRequestReserve])
	ctl.workConnMu.Unlock()
	require.Zero(t, late)
	require.Zero(t, pending)
}

func TestControlLegacyResponseKeepsDemandPriority(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 0)
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	resultCh := make(chan workConnResult, 1)
	go func() {
		conn, err := ctl.GetWorkConn()
		resultCh <- workConnResult{conn: conn, err: err}
	}()
	require.Equal(t, msg.WorkConnTypeDemand, readWorkConnRequest(t, clientMsgConn).WorkConnType)

	conn := newCountingCloseConn()
	accepted, err := ctl.registerWorkConn(
		proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
		workConnRequestLegacy,
	)
	require.NoError(t, err)
	require.True(t, accepted)
	result := <-resultCh
	require.NoError(t, result.err)
	require.NotNil(t, result.conn)
}

func TestControlReserveResponseKeepsOutstandingDemandSupply(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 2)
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	resultCh := make(chan workConnResult, 1)
	go func() {
		conn, err := ctl.GetWorkConn()
		resultCh <- workConnResult{conn: conn, err: err}
	}()
	require.Equal(t, msg.WorkConnTypeDemand, readWorkConnRequest(t, clientMsgConn).WorkConnType)
	require.Equal(t, msg.WorkConnTypeReserve, readWorkConnRequest(t, clientMsgConn).WorkConnType)
	require.Equal(t, msg.WorkConnTypeReserve, readWorkConnRequest(t, clientMsgConn).WorkConnType)

	conn := newCountingCloseConn()
	accepted, err := ctl.registerWorkConn(
		proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
		workConnRequestReserve,
	)
	require.NoError(t, err)
	require.True(t, accepted)
	result := <-resultCh
	require.NoError(t, result.err)
	require.NotNil(t, result.conn)
	_ = result.conn.Close()

	ctl.workConnMu.Lock()
	demandPending := len(ctl.workConnPending[workConnRequestDemand])
	reservePending := len(ctl.workConnPending[workConnRequestReserve])
	lateKinds := len(ctl.workConnLate)
	ctl.workConnMu.Unlock()
	require.Equal(t, 1, demandPending)
	require.Equal(t, 1, reservePending)
	require.Zero(t, lateKinds)

	conn = newCountingCloseConn()
	accepted, err = ctl.registerWorkConn(
		proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
		workConnRequestDemand,
	)
	require.NoError(t, err)
	require.True(t, accepted)

	conn = newCountingCloseConn()
	accepted, err = ctl.registerWorkConn(
		proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
		workConnRequestReserve,
	)
	require.NoError(t, err)
	require.True(t, accepted)
	require.Len(t, ctl.workConnCh, 2)
}

func TestControlCloseWakesWorkConnWaiter(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 0)
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	resultCh := make(chan error, 1)
	go func() {
		_, err := ctl.GetWorkConn()
		resultCh <- err
	}()
	require.Equal(t, msg.WorkConnTypeDemand, readWorkConnRequest(t, clientMsgConn).WorkConnType)
	ctl.closeWorkConnPool()
	require.ErrorIs(t, <-resultCh, pkgerr.ErrCtlClosed)
}

func TestControlManagerRegisterWorkConnDoesNotHoldLifecycleDuringRefill(t *testing.T) {
	for _, action := range []string{"close", "replace"} {
		t.Run(action, func(t *testing.T) {
			manager := NewControlManager(registry.NewClientRegistry())
			conn := newBlockingWriteConn()
			ctl, err := NewControl(context.Background(), &SessionContext{
				RC:            &controller.ResourceController{},
				PxyManager:    proxy.NewManager(),
				PluginManager: plugin.NewManager(),
				AuthVerifier:  auth.AlwaysPassVerifier,
				Conn:          msg.NewConn(conn, msg.NewV1ReadWriter(conn)),
				LoginMsg: &msg.Login{
					RunID:     "stalled-refill",
					ClientID:  "stalled-refill-client",
					PoolCount: 10,
				},
				ServerCfg: &v1.ServerConfig{},
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = ctl.Close() })
			require.NoError(t, manager.Add(ctl))
			active, err := manager.Activate(ctl)
			require.NoError(t, err)
			require.True(t, active)
			require.True(t, ctl.Start())

			// Block the dispatcher writer, then leave enough queued messages for
			// the refill batch to block in Dispatcher.Send.
			require.NoError(t, ctl.msgDispatcher.Send(&msg.Ping{}))
			waitForSignal(t, conn.writeStarted, "dispatcher write to stall")
			for range 40 {
				require.NoError(t, ctl.msgDispatcher.Send(&msg.Ping{}))
			}

			const waiterCount = 100
			waiters := make([]*workConnWaiter, 0, waiterCount)
			ctl.workConnMu.Lock()
			ctl.workConnLegacyMode = false
			for range waiterCount {
				waiter := &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
				waiters = append(waiters, waiter)
				ctl.appendWorkConnWaiterLocked(waiter)
			}
			ctl.workConnMu.Unlock()

			registerDone := make(chan error, 1)
			workConn := newCountingCloseConn()
			go func() {
				registerDone <- manager.RegisterWorkConnWithType(
					ctl,
					proxy.NewWorkConn(msg.NewConn(workConn, msg.NewV1ReadWriter(workConn))),
					workConnRequestDemand,
				)
			}()
			result := waitForResult(t, waiters[0].resultCh, "registered work connection waiter")
			require.NoError(t, result.err)
			require.NotNil(t, result.conn)
			deadline := time.Now().Add(time.Second)
			for {
				ctl.workConnMu.Lock()
				pending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestDemand) +
					ctl.leaseCountLocked(ctl.workConnPending, workConnRequestReserve)
				ctl.workConnMu.Unlock()
				if pending > 0 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("refill reconciler did not create a lease")
				}
				time.Sleep(time.Millisecond)
			}

			if action == "close" {
				closeDone := make(chan error, 1)
				go func() { closeDone <- ctl.Close() }()
				select {
				case err := <-closeDone:
					require.NoError(t, err)
				case <-time.After(time.Second):
					_ = ctl.interruptReadAndClose()
					t.Errorf("Close blocked while refill Send was stalled")
				}
			} else {
				replacement, _ := newLifecycleTestControl(t, "stalled-refill", "replacement", newCountingServerMetrics())
				addDone := make(chan error, 1)
				go func() { addDone <- manager.Add(replacement) }()
				select {
				case err := <-addDone:
					require.NoError(t, err)
				case <-time.After(time.Second):
					_ = ctl.interruptReadAndClose()
					t.Errorf("replacement blocked while refill Send was stalled")
				}
			}

			select {
			case err := <-registerDone:
				require.NoError(t, err)
			case <-time.After(time.Second):
				_ = ctl.interruptReadAndClose()
				t.Fatal("RegisterWorkConn did not finish after control unblocked")
			}
			_ = result.conn.Close()
			waitForControlDone(t, ctl)
			select {
			case <-ctl.workConnReconcileDone:
			case <-time.After(time.Second):
				t.Fatal("work connection reconciler leaked after control close/replacement")
			}
		})
	}
}

func TestControlCloseAndReplacementDoNotBlockOnPooledWorkConnClose(t *testing.T) {
	for _, action := range []string{"close", "replace"} {
		t.Run(action, func(t *testing.T) {
			manager := NewControlManager(registry.NewClientRegistry())
			metrics := newCountingServerMetrics()
			ctl, controlConn := newLifecycleTestControl(t, "blocked-cleanup", "blocked-client", metrics)
			mustAddAndActivate(t, manager, ctl)
			require.True(t, ctl.Start())
			waitForSignal(t, controlConn.readStarted, "control reader to start")

			pooledConn := newBlockingCloseConn()
			pooledWorkConn := proxy.NewWorkConn(msg.NewConn(pooledConn, msg.NewV1ReadWriter(pooledConn)))
			ctl.workConnMu.Lock()
			ctl.workConnCh <- pooledWorkConn
			ctl.workConnMu.Unlock()

			if action == "close" {
				closeDone := make(chan error, 1)
				go func() { closeDone <- ctl.Close() }()
				select {
				case err := <-closeDone:
					require.NoError(t, err)
				case <-time.After(time.Second):
					t.Fatal("Control.Close blocked on pooled work connection Close")
				}
			} else {
				replacement, _ := newLifecycleTestControl(t, "blocked-cleanup", "replacement", newCountingServerMetrics())
				addDone := make(chan error, 1)
				go func() { addDone <- manager.Add(replacement) }()
				select {
				case err := <-addDone:
					require.NoError(t, err)
				case <-time.After(time.Second):
					t.Fatal("same-run replacement blocked on pooled work connection Close")
				}
			}

			waitForSignal(t, pooledConn.closeStarted, "pooled work connection cleanup to start")
			close(pooledConn.allowClose)
			waitForSignal(t, ctl.workConnCleanupDone, "pooled work connection cleanup to finish")
			require.Equal(t, int64(1), pooledConn.closeCount.Load())
			waitForControlDone(t, ctl)
		})
	}
}

func TestControlUndeliveredWaiterHandoffIsClosedOnShutdown(t *testing.T) {
	for _, action := range []string{"close", "replace"} {
		t.Run(action, func(t *testing.T) {
			metrics := newCountingServerMetrics()
			ctl, controlConn := newLifecycleTestControl(t, "undelivered-handoff", "old-client", metrics)
			manager := NewControlManager(registry.NewClientRegistry())
			mustAddAndActivate(t, manager, ctl)
			require.True(t, ctl.Start())
			waitForSignal(t, controlConn.readStarted, "control reader to start")

			waiter := &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
			ctl.workConnMu.Lock()
			ctl.appendWorkConnWaiterLocked(waiter)
			ctl.workConnMu.Unlock()

			pooledConn := newBlockingCloseConn()
			workConn := proxy.NewWorkConn(msg.NewConn(pooledConn, msg.NewV1ReadWriter(pooledConn)))
			accepted, err := ctl.registerWorkConnState(workConn, workConnRequestDemand)
			require.NoError(t, err)
			require.True(t, accepted)
			// Leave the buffered result unread to model a consumer preempted
			// between the channel wakeup and its ownership claim.
			require.Len(t, waiter.resultCh, 1)

			if action == "close" {
				closeDone := make(chan error, 1)
				go func() { closeDone <- ctl.Close() }()
				select {
				case err := <-closeDone:
					require.NoError(t, err)
				case <-time.After(time.Second):
					t.Fatal("Control.Close blocked on an undelivered waiter handoff")
				}
			} else {
				replacement, _ := newLifecycleTestControl(t, "undelivered-handoff", "new-client", newCountingServerMetrics())
				addDone := make(chan error, 1)
				go func() { addDone <- manager.Add(replacement) }()
				select {
				case err := <-addDone:
					require.NoError(t, err)
				case <-time.After(time.Second):
					t.Fatal("same-run replacement blocked on an undelivered waiter handoff")
				}
			}

			waitForSignal(t, pooledConn.closeStarted, "undelivered work connection cleanup to start")
			result := <-waiter.resultCh
			conn, err := ctl.consumeWorkConnWaiterResult(waiter, result)
			require.Nil(t, conn)
			require.ErrorIs(t, err, pkgerr.ErrCtlClosed)
			close(pooledConn.allowClose)
			waitForSignal(t, ctl.workConnCleanupDone, "undelivered work connection cleanup to finish")
			require.Equal(t, int64(1), pooledConn.closeCount.Load())
			waitForControlDone(t, ctl)
		})
	}
}

func TestControlGetWorkConnTimeoutStartsBeforeBackpressuredRefill(t *testing.T) {
	conn := newBlockingWriteConn()
	cfg := &v1.ServerConfig{UserConnTimeout: 1}
	cfg.Transport.MaxPoolCount = 0
	ctl, err := NewControl(context.Background(), &SessionContext{
		RC:            &controller.ResourceController{},
		PxyManager:    proxy.NewManager(),
		PluginManager: plugin.NewManager(),
		AuthVerifier:  auth.AlwaysPassVerifier,
		Conn:          msg.NewConn(conn, msg.NewV1ReadWriter(conn)),
		LoginMsg: &msg.Login{
			RunID:     "backpressured-timeout",
			PoolCount: 0,
		},
		ServerCfg: cfg,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctl.Close() })
	ctl.state = controlStateRunning
	ctl.msgDispatcher.Run()
	ctl.workConnLeaseTimeout = 30 * time.Millisecond

	require.NoError(t, ctl.msgDispatcher.Send(&msg.Ping{}))
	waitForSignal(t, conn.writeStarted, "dispatcher write to stall")
	for range 100 {
		require.NoError(t, ctl.msgDispatcher.Send(&msg.Ping{}))
	}

	resultCh := make(chan error, 1)
	started := time.Now()
	go func() {
		_, err := ctl.GetWorkConn()
		resultCh <- err
	}()
	select {
	case err := <-resultCh:
		require.ErrorContains(t, err, "timeout trying to get work connection")
		require.GreaterOrEqual(t, time.Since(started), 20*time.Millisecond)
		require.Less(t, time.Since(started), 300*time.Millisecond)
	case <-time.After(time.Second):
		t.Fatal("GetWorkConn ignored timeout while refill was backpressured")
	}

	_ = ctl.Close()
	select {
	case <-ctl.workConnReconcileDone:
	case <-time.After(time.Second):
		t.Fatal("work connection reconciler did not stop after close")
	}
}

func TestControlUnsentWorkConnLeaseStartsTimeoutAfterDispatch(t *testing.T) {
	conn := newBlockingWriteConn()
	cfg := &v1.ServerConfig{UserConnTimeout: 1}
	cfg.Transport.MaxPoolCount = 0
	ctl, err := NewControl(context.Background(), &SessionContext{
		RC:            &controller.ResourceController{},
		PxyManager:    proxy.NewManager(),
		PluginManager: plugin.NewManager(),
		AuthVerifier:  auth.AlwaysPassVerifier,
		Conn:          msg.NewConn(conn, msg.NewV1ReadWriter(conn)),
		LoginMsg: &msg.Login{
			RunID:     "queued-lease-timeout",
			PoolCount: 0,
		},
		ServerCfg: cfg,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctl.Close() })
	ctl.state = controlStateRunning
	ctl.msgDispatcher.Run()
	ctl.workConnLeaseTimeout = 25 * time.Millisecond

	require.NoError(t, ctl.msgDispatcher.Send(&msg.Ping{}))
	waitForSignal(t, conn.writeStarted, "dispatcher write to stall")
	for range 100 {
		require.NoError(t, ctl.msgDispatcher.Send(&msg.Ping{}))
	}

	resultCh := make(chan error, 1)
	go func() {
		_, err := ctl.GetWorkConn()
		resultCh <- err
	}()
	deadline := time.Now().Add(time.Second)
	for {
		ctl.workConnMu.Lock()
		pending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestDemand)
		late := ctl.leaseCountLocked(ctl.workConnLate, workConnRequestDemand)
		var timer *time.Timer
		dispatched := false
		if leases := ctl.workConnPending[workConnRequestDemand]; len(leases) > 0 {
			timer = leases[0].timer
			dispatched = leases[0].dispatched
		}
		ctl.workConnMu.Unlock()
		if pending == 1 {
			require.Zero(t, late)
			require.Nil(t, timer)
			require.False(t, dispatched)
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("demand lease was not queued")
		}
		time.Sleep(time.Millisecond)
	}
	time.Sleep(80 * time.Millisecond)
	ctl.workConnMu.Lock()
	pending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestDemand)
	late := ctl.leaseCountLocked(ctl.workConnLate, workConnRequestDemand)
	var leaseTimer *time.Timer
	if leases := ctl.workConnPending[workConnRequestDemand]; len(leases) > 0 {
		leaseTimer = leases[0].timer
	}
	ctl.workConnMu.Unlock()
	require.Equal(t, 1, pending)
	require.Zero(t, late)
	require.Nil(t, leaseTimer)

	conn.releaseWrite()
	deadline = time.Now().Add(time.Second)
	for {
		ctl.workConnMu.Lock()
		var timer *time.Timer
		if leases := ctl.workConnPending[workConnRequestDemand]; len(leases) > 0 {
			timer = leases[0].timer
		}
		ctl.workConnMu.Unlock()
		if timer != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("queued request did not start its response timeout after dispatch")
		}
		time.Sleep(time.Millisecond)
	}
	_ = ctl.Close()
	select {
	case <-ctl.workConnReconcileDone:
	case <-time.After(time.Second):
		t.Fatal("work connection reconciler did not stop after close")
	}
	select {
	case err := <-resultCh:
		require.ErrorContains(t, err, "timeout trying to get work connection")
	case <-time.After(time.Second):
		t.Fatal("GetWorkConn did not finish")
	}
}

func TestControlLeaseTimerDoesNotStartAfterResponseWinsRace(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 1)
	ctl.workConnLeaseTimeout = time.Minute
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	ctl.workConnMu.Lock()
	lease := ctl.newWorkConnLeaseLocked(workConnRequestDemand)
	ctl.workConnMu.Unlock()

	conn := newCountingCloseConn()
	accepted, err := ctl.registerWorkConnState(
		proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
		workConnRequestDemand,
	)
	require.True(t, accepted)
	require.NoError(t, err)
	ctl.startWorkConnLeaseTimer(lease)

	ctl.workConnMu.Lock()
	dispatched := lease.dispatched
	leaseTimer := lease.timer
	pendingKinds := len(ctl.workConnPending)
	ctl.workConnMu.Unlock()
	require.False(t, dispatched)
	require.Nil(t, leaseTimer)
	require.Zero(t, pendingKinds)
}

func moveReserveWorkConnLeaseToLateForTest(t *testing.T, ctl *Control) {
	t.Helper()
	const kind = workConnRequestReserve
	ctl.workConnMu.Lock()
	leases := ctl.workConnPending[kind]
	if len(leases) == 0 {
		ctl.workConnMu.Unlock()
		t.Fatal("expected a pending work connection lease")
		return
	}
	lease := leases[0]
	ctl.removeLeaseLocked(ctl.workConnPending, lease)
	lease.late = true
	if lease.timer != nil {
		lease.timer.Stop()
		lease.timer = nil
	}
	ctl.workConnLate[kind] = append(ctl.workConnLate[kind], lease)
	ctl.workConnMu.Unlock()
}

func TestControlLateResponseAdmissionAvoidsReplacementOversupply(t *testing.T) {
	for _, responseKind := range []workConnRequestKind{workConnRequestReserve, workConnRequestLegacy} {
		t.Run(string(responseKind), func(t *testing.T) {
			ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 1)
			ctl.workConnLeaseTimeout = time.Minute
			t.Cleanup(func() {
				ctl.closeWorkConnPool()
				_ = serverMsgConn.Close()
				_ = clientMsgConn.Close()
			})

			ctl.reconcileWorkConnRequests()
			require.Equal(t, msg.WorkConnTypeReserve, readWorkConnRequest(t, clientMsgConn).WorkConnType)
			moveReserveWorkConnLeaseToLateForTest(t, ctl)
			ctl.reconcileWorkConnRequests()
			require.Equal(t, msg.WorkConnTypeReserve, readWorkConnRequest(t, clientMsgConn).WorkConnType)

			// Replies carry no request ID. Even if the original arrives first,
			// it retires pending supply and the subsequent duplicate is rejected.
			lateConn := newCountingCloseConn()
			accepted, err := ctl.registerWorkConnState(
				proxy.NewWorkConn(msg.NewConn(lateConn, msg.NewV1ReadWriter(lateConn))),
				responseKind,
			)
			require.True(t, accepted)
			require.NoError(t, err)
			require.Len(t, ctl.workConnCh, 1)

			replacementConn := newCountingCloseConn()
			accepted, err = ctl.registerWorkConnState(
				proxy.NewWorkConn(msg.NewConn(replacementConn, msg.NewV1ReadWriter(replacementConn))),
				responseKind,
			)
			require.False(t, accepted)
			require.ErrorIs(t, err, errWorkConnPoolFull)
			require.NoError(t, replacementConn.Close())
			require.Len(t, ctl.workConnCh, 1)
		})
	}
}

func TestControlUnmatchedResponseCannotCacheBeyondDemandOrPool(t *testing.T) {
	for _, responseKind := range []workConnRequestKind{
		workConnRequestDemand,
		workConnRequestReserve,
		workConnRequestLegacy,
	} {
		t.Run(string(responseKind), func(t *testing.T) {
			ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 0)
			t.Cleanup(func() {
				ctl.closeWorkConnPool()
				_ = serverMsgConn.Close()
				_ = clientMsgConn.Close()
			})

			conn := newCountingCloseConn()
			accepted, err := ctl.registerWorkConnState(
				proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
				responseKind,
			)
			require.False(t, accepted)
			require.ErrorIs(t, err, errWorkConnPoolFull)
			require.NoError(t, conn.Close())
			require.Len(t, ctl.workConnCh, 0)
		})
	}

	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 0)
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})
	waiter := &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
	ctl.workConnMu.Lock()
	ctl.appendWorkConnWaiterLocked(waiter)
	lease := ctl.newWorkConnLeaseLocked(workConnRequestDemand)
	ctl.workConnMu.Unlock()
	require.True(t, ctl.cancelWorkConnWaiter(waiter))

	conn := newCountingCloseConn()
	accepted, err := ctl.registerWorkConnState(
		proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
		workConnRequestDemand,
	)
	require.False(t, accepted)
	require.ErrorIs(t, err, errWorkConnPoolFull)
	require.NoError(t, conn.Close())
	ctl.workConnMu.Lock()
	pending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestDemand)
	leaseTimer := lease.timer
	ctl.workConnMu.Unlock()
	require.Zero(t, pending)
	require.Nil(t, leaseTimer)
}

func TestControlLateResponseCanSatisfyWaiterOrRealGap(t *testing.T) {
	for _, responseKind := range []workConnRequestKind{workConnRequestDemand, workConnRequestLegacy} {
		t.Run(string(responseKind), func(t *testing.T) {
			ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 0)
			t.Cleanup(func() {
				ctl.closeWorkConnPool()
				_ = serverMsgConn.Close()
				_ = clientMsgConn.Close()
			})

			waiter := &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
			ctl.workConnMu.Lock()
			ctl.appendWorkConnWaiterLocked(waiter)
			ctl.workConnLate[workConnRequestDemand] = append(ctl.workConnLate[workConnRequestDemand], &workConnLease{
				kind:   workConnRequestDemand,
				active: true,
				late:   true,
			})
			ctl.workConnMu.Unlock()

			conn := newCountingCloseConn()
			accepted, err := ctl.registerWorkConnState(
				proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
				responseKind,
			)
			require.True(t, accepted)
			require.NoError(t, err)
			result := <-waiter.resultCh
			handoff, err := ctl.consumeWorkConnWaiterResult(waiter, result)
			require.NoError(t, err)
			require.NotNil(t, handoff)
			require.NoError(t, handoff.Close())
		})
	}

	for _, responseKind := range []workConnRequestKind{workConnRequestReserve, workConnRequestLegacy} {
		t.Run("gap_"+string(responseKind), func(t *testing.T) {
			ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 2)
			t.Cleanup(func() {
				ctl.closeWorkConnPool()
				_ = serverMsgConn.Close()
				_ = clientMsgConn.Close()
			})
			ctl.workConnMu.Lock()
			ctl.workConnLate[workConnRequestReserve] = append(ctl.workConnLate[workConnRequestReserve], &workConnLease{
				kind:   workConnRequestReserve,
				active: true,
				late:   true,
			})
			ctl.workConnPending[workConnRequestReserve] = append(ctl.workConnPending[workConnRequestReserve], &workConnLease{
				kind:   workConnRequestReserve,
				active: true,
			})
			ctl.workConnMu.Unlock()

			lateConn := newCountingCloseConn()
			accepted, err := ctl.registerWorkConnState(
				proxy.NewWorkConn(msg.NewConn(lateConn, msg.NewV1ReadWriter(lateConn))),
				responseKind,
			)
			require.True(t, accepted)
			require.NoError(t, err)
			require.Len(t, ctl.workConnCh, 1)

			pendingConn := newCountingCloseConn()
			accepted, err = ctl.registerWorkConnState(
				proxy.NewWorkConn(msg.NewConn(pendingConn, msg.NewV1ReadWriter(pendingConn))),
				responseKind,
			)
			require.True(t, accepted)
			require.NoError(t, err)
			require.Len(t, ctl.workConnCh, 2)
		})
	}
}

func TestControlActiveWaiterCountTracksFIFOCancelHandoffAndClose(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 0)
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	waiters := []*workConnWaiter{
		{resultCh: make(chan workConnResult, 1), active: true},
		{resultCh: make(chan workConnResult, 1), active: true},
		{resultCh: make(chan workConnResult, 1), active: true},
	}
	ctl.workConnMu.Lock()
	for _, waiter := range waiters {
		ctl.appendWorkConnWaiterLocked(waiter)
	}
	activeWaiters := ctl.activeWaiterCountLocked()
	ctl.workConnMu.Unlock()
	require.Equal(t, 3, activeWaiters)

	require.True(t, ctl.cancelWorkConnWaiter(waiters[1]))
	ctl.workConnMu.Lock()
	activeWaiters = ctl.activeWaiterCountLocked()
	ctl.workConnMu.Unlock()
	require.Equal(t, 2, activeWaiters)

	conn := newCountingCloseConn()
	accepted, err := ctl.registerWorkConnState(
		proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn))),
		workConnRequestDemand,
	)
	require.True(t, accepted)
	require.NoError(t, err)
	ctl.workConnMu.Lock()
	activeWaiters = ctl.activeWaiterCountLocked()
	ctl.workConnMu.Unlock()
	require.Equal(t, 1, activeWaiters)

	result := <-waiters[0].resultCh
	handoff, err := ctl.consumeWorkConnWaiterResult(waiters[0], result)
	require.NoError(t, err)
	require.NotNil(t, handoff)
	require.NoError(t, handoff.Close())
	ctl.workConnMu.Lock()
	activeWaiters = ctl.activeWaiterCountLocked()
	handoffs := ctl.workConnHandoffs.Len()
	ctl.workConnMu.Unlock()
	require.Equal(t, 1, activeWaiters)
	require.Zero(t, handoffs)

	ctl.closeWorkConnPool()
	ctl.workConnMu.Lock()
	activeWaiters = ctl.activeWaiterCountLocked()
	ctl.workConnMu.Unlock()
	require.Zero(t, activeWaiters)
}

func TestControlActiveWaiterCountLargeConcurrentCancellation(t *testing.T) {
	const waiterCount = 1024

	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 0)
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	waiters := make([]*workConnWaiter, waiterCount)
	ctl.workConnMu.Lock()
	for i := range waiters {
		waiters[i] = &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
		ctl.appendWorkConnWaiterLocked(waiters[i])
	}
	initialCount := ctl.activeWaiterCountLocked()
	ctl.workConnMu.Unlock()
	require.Equal(t, waiterCount, initialCount)

	var (
		wg            sync.WaitGroup
		canceledCount atomic.Int64
	)
	for i := 0; i < waiterCount; i += 2 {
		wg.Add(1)
		go func(waiter *workConnWaiter) {
			defer wg.Done()
			if ctl.cancelWorkConnWaiter(waiter) {
				canceledCount.Add(1)
			}
		}(waiters[i])
	}
	wg.Wait()
	require.Equal(t, int64(waiterCount/2), canceledCount.Load())

	ctl.workConnMu.Lock()
	activeCount := ctl.activeWaiterCountLocked()
	queueLen := len(ctl.workConnWaiters)
	// Canceled waiters remain as lazy tombstones; reconcile reads the O(1)
	// active count and does not compact or rescan the full queue on every
	// cancellation. A threshold-triggered compact may already have removed
	// the tombstones.
	ctl.workConnMu.Unlock()
	require.Equal(t, waiterCount/2, activeCount)
	require.LessOrEqual(t, queueLen, waiterCount)

	ctl.workConnMu.Lock()
	popped := make([]*workConnWaiter, 0, waiterCount/2)
	for i := 1; i < waiterCount; i += 2 {
		popped = append(popped, ctl.popWorkConnWaiterLocked())
	}
	nilPop := ctl.popWorkConnWaiterLocked()
	remainingCount := ctl.activeWaiterCountLocked()
	remainingQueueLen := len(ctl.workConnWaiters)
	remainingTombstones := ctl.workConnWaiterTombstones
	ctl.workConnMu.Unlock()
	for i, waiter := range popped {
		require.Same(t, waiters[2*i+1], waiter)
	}
	require.Nil(t, nilPop)
	require.Zero(t, remainingCount)
	require.Zero(t, remainingQueueLen)
	require.Zero(t, remainingTombstones)
}

func TestControlCanceledWaitersAreCompactedAndBounded(t *testing.T) {
	const canceledCount = 10000

	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 0)
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	for range canceledCount {
		waiter := &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
		ctl.workConnMu.Lock()
		ctl.appendWorkConnWaiterLocked(waiter)
		ctl.workConnMu.Unlock()
		require.True(t, ctl.cancelWorkConnWaiter(waiter))
	}

	ctl.workConnMu.Lock()
	activeCount := ctl.activeWaiterCountLocked()
	tombstoneCount := ctl.workConnWaiterTombstones
	queueLen := len(ctl.workConnWaiters)
	ctl.workConnMu.Unlock()
	require.Zero(t, activeCount)
	require.Less(t, tombstoneCount, workConnWaiterCompactMinTombstones)
	require.Less(t, queueLen, workConnWaiterCompactMinTombstones)
	require.Equal(t, activeCount+tombstoneCount, queueLen)

	// Closing the control clears both active waiters and residual tombstones.
	ctl.closeWorkConnPool()
	ctl.workConnMu.Lock()
	activeCount = ctl.activeWaiterCountLocked()
	tombstoneCount = ctl.workConnWaiterTombstones
	queueLen = len(ctl.workConnWaiters)
	ctl.workConnMu.Unlock()
	require.Zero(t, activeCount)
	require.Zero(t, tombstoneCount)
	require.Zero(t, queueLen)
}

func TestControlCanceledWaiterCompactionPreservesFIFO(t *testing.T) {
	ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControl(t, 0)
	t.Cleanup(func() {
		ctl.closeWorkConnPool()
		_ = serverMsgConn.Close()
		_ = clientMsgConn.Close()
	})

	const waiterCount = 256
	waiters := make([]*workConnWaiter, waiterCount)
	ctl.workConnMu.Lock()
	for i := range waiters {
		waiters[i] = &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
		ctl.appendWorkConnWaiterLocked(waiters[i])
	}
	ctl.workConnMu.Unlock()

	for i := 1; i < waiterCount; i += 2 {
		require.True(t, ctl.cancelWorkConnWaiter(waiters[i]))
	}

	ctl.workConnMu.Lock()
	activeCount := ctl.activeWaiterCountLocked()
	ctl.workConnMu.Unlock()
	require.Equal(t, waiterCount/2, activeCount)

	ctl.workConnMu.Lock()
	popped := make([]*workConnWaiter, 0, waiterCount/2)
	for i := 0; i < waiterCount; i += 2 {
		popped = append(popped, ctl.popWorkConnWaiterLocked())
	}
	nilPop := ctl.popWorkConnWaiterLocked()
	remainingCount := ctl.activeWaiterCountLocked()
	remainingQueueLen := len(ctl.workConnWaiters)
	remainingTombstones := ctl.workConnWaiterTombstones
	ctl.workConnMu.Unlock()
	for i, waiter := range popped {
		require.Same(t, waiters[2*i], waiter)
	}
	require.Nil(t, nilPop)
	require.Zero(t, remainingCount)
	require.Zero(t, remainingQueueLen)
	require.Zero(t, remainingTombstones)
}

func TestControlNonPositiveWorkConnTimeoutIsSharedByWaiterAndLease(t *testing.T) {
	for _, configuredTimeout := range []int64{0, -1} {
		t.Run(fmt.Sprintf("timeout_%d", configuredTimeout), func(t *testing.T) {
			ctl, serverMsgConn, clientMsgConn := newWorkConnSchedulingControlWithTimeout(t, 0, configuredTimeout)
			require.Equal(t, 10*time.Second, ctl.workConnLeaseTimeoutValue())
			// Shorten the effective value for a deterministic test while retaining
			// the non-positive configuration that would otherwise fall back to 10s.
			ctl.workConnLeaseTimeout = 30 * time.Millisecond
			t.Cleanup(func() {
				ctl.closeWorkConnPool()
				_ = serverMsgConn.Close()
				_ = clientMsgConn.Close()
			})

			resultCh := make(chan error, 1)
			started := time.Now()
			go func() {
				_, err := ctl.GetWorkConn()
				resultCh <- err
			}()
			require.Equal(t, msg.WorkConnTypeDemand, readWorkConnRequest(t, clientMsgConn).WorkConnType)
			select {
			case err := <-resultCh:
				require.ErrorContains(t, err, "timeout trying to get work connection")
				require.GreaterOrEqual(t, time.Since(started), 20*time.Millisecond)
				require.Less(t, time.Since(started), 300*time.Millisecond)
			case <-time.After(time.Second):
				t.Fatal("waiter did not use the effective non-positive timeout")
			}

			deadline := time.Now().Add(time.Second)
			for {
				ctl.workConnMu.Lock()
				pending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestDemand)
				late := ctl.leaseCountLocked(ctl.workConnLate, workConnRequestDemand)
				ctl.workConnMu.Unlock()
				if pending == 0 && late == 1 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("lease timeout did not match waiter timeout: pending=%d late=%d", pending, late)
				}
				time.Sleep(time.Millisecond)
			}
			ctl.closeWorkConnPool()
			select {
			case <-ctl.workConnReconcileDone:
			case <-time.After(time.Second):
				t.Fatal("work connection reconciler did not stop after pool close")
			}
		})
	}
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
	ctl.waitWorkConnCleanup()
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

func newWorkConnSchedulingControl(t *testing.T, poolCount int) (*Control, *msg.Conn, *msg.Conn) {
	return newWorkConnSchedulingControlWithTimeout(t, poolCount, 1)
}

func newWorkConnSchedulingControlWithTimeout(t *testing.T, poolCount int, userConnTimeout int64) (*Control, *msg.Conn, *msg.Conn) {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	serverMsgConn := msg.NewConn(serverConn, msg.NewV1ReadWriter(serverConn))
	clientMsgConn := msg.NewConn(clientConn, msg.NewV1ReadWriter(clientConn))
	cfg := &v1.ServerConfig{}
	cfg.Transport.MaxPoolCount = int64(poolCount)
	cfg.UserConnTimeout = userConnTimeout
	ctl, err := NewControl(context.Background(), &SessionContext{
		RC:            &controller.ResourceController{},
		PxyManager:    proxy.NewManager(),
		PluginManager: plugin.NewManager(),
		AuthVerifier:  auth.AlwaysPassVerifier,
		Conn:          serverMsgConn,
		LoginMsg: &msg.Login{
			RunID:     "work-conn-scheduling",
			PoolCount: poolCount,
		},
		ServerCfg: cfg,
	})
	require.NoError(t, err)
	ctl.state = controlStateRunning
	ctl.msgDispatcher.Run()
	return ctl, serverMsgConn, clientMsgConn
}

func readWorkConnRequest(t *testing.T, conn *msg.Conn) *msg.ReqWorkConn {
	t.Helper()
	resultCh := make(chan struct {
		msg *msg.ReqWorkConn
		err error
	}, 1)
	go func() {
		m, err := conn.ReadMsg()
		req, ok := m.(*msg.ReqWorkConn)
		if !ok && err == nil {
			err = errors.New("unexpected work connection request message")
		}
		resultCh <- struct {
			msg *msg.ReqWorkConn
			err error
		}{msg: req, err: err}
	}()
	select {
	case result := <-resultCh:
		require.NoError(t, result.err)
		require.NotNil(t, result.msg)
		return result.msg
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ReqWorkConn")
		return nil
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

type blockingWriteConn struct {
	writeStarted chan struct{}
	closed       chan struct{}
	allowWrite   chan struct{}
	writeOnce    sync.Once
	closeOnce    sync.Once
	allowOnce    sync.Once
}

func newBlockingWriteConn() *blockingWriteConn {
	return &blockingWriteConn{
		writeStarted: make(chan struct{}),
		closed:       make(chan struct{}),
		allowWrite:   make(chan struct{}),
	}
}

type blockingCloseConn struct {
	closeStarted chan struct{}
	allowClose   chan struct{}
	closeCount   atomic.Int64
	closeOnce    sync.Once
}

func newBlockingCloseConn() *blockingCloseConn {
	return &blockingCloseConn{
		closeStarted: make(chan struct{}),
		allowClose:   make(chan struct{}),
	}
}

func (*blockingCloseConn) Read([]byte) (int, error)    { return 0, net.ErrClosed }
func (*blockingCloseConn) Write(p []byte) (int, error) { return len(p), nil }
func (c *blockingCloseConn) Close() error {
	c.closeOnce.Do(func() { close(c.closeStarted) })
	<-c.allowClose
	c.closeCount.Add(1)
	return nil
}
func (*blockingCloseConn) LocalAddr() net.Addr              { return lifecycleTestAddr("local") }
func (*blockingCloseConn) RemoteAddr() net.Addr             { return lifecycleTestAddr("remote") }
func (*blockingCloseConn) SetDeadline(time.Time) error      { return nil }
func (*blockingCloseConn) SetReadDeadline(time.Time) error  { return nil }
func (*blockingCloseConn) SetWriteDeadline(time.Time) error { return nil }

func (c *blockingWriteConn) Read([]byte) (int, error) {
	<-c.closed
	return 0, net.ErrClosed
}

func (c *blockingWriteConn) Write(p []byte) (int, error) {
	c.writeOnce.Do(func() { close(c.writeStarted) })
	select {
	case <-c.allowWrite:
		return len(p), nil
	case <-c.closed:
		return 0, net.ErrClosed
	}
}

func (c *blockingWriteConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
		c.allowOnce.Do(func() { close(c.allowWrite) })
	})
	return nil
}

func (c *blockingWriteConn) releaseWrite() {
	c.allowOnce.Do(func() { close(c.allowWrite) })
}

func (*blockingWriteConn) LocalAddr() net.Addr              { return lifecycleTestAddr("local") }
func (*blockingWriteConn) RemoteAddr() net.Addr             { return lifecycleTestAddr("remote") }
func (*blockingWriteConn) SetDeadline(time.Time) error      { return nil }
func (*blockingWriteConn) SetReadDeadline(time.Time) error  { return nil }
func (*blockingWriteConn) SetWriteDeadline(time.Time) error { return nil }

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

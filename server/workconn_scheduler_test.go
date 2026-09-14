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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	pkgerr "github.com/fatedier/frp/pkg/errors"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/server/proxy"
)

func TestWorkConnReplacementRestoresPoolWithoutOriginalResponse(t *testing.T) {
	for _, kind := range []workConnRequestKind{workConnRequestReserve, workConnRequestLegacy} {
		t.Run(string(kind), func(t *testing.T) {
			ctl, serverConn, clientConn := newWorkConnSchedulingControl(t, 1)
			ctl.workConnLeaseTimeout = time.Minute
			t.Cleanup(func() {
				ctl.closeWorkConnPool()
				_ = serverConn.Close()
				_ = clientConn.Close()
			})
			ctl.reconcileWorkConnRequests()
			readWorkConnRequest(t, clientConn)
			// The original request never returns. Trigger its timeout transition
			// without sleeping, then deliver only the replacement's response.
			moveReserveWorkConnLeaseToLateForTest(t, ctl)
			ctl.reconcileWorkConnRequests()
			readWorkConnRequest(t, clientConn)
			conn := newCountingCloseConn()
			workConn := proxy.NewWorkConn(msg.NewConn(conn, msg.NewV1ReadWriter(conn)))
			accepted, err := ctl.registerWorkConnState(workConn, kind)
			if !accepted {
				_ = workConn.Close()
			}
			require.NoError(t, err)
			require.True(t, accepted)
			ctl.workConnMu.Lock()
			pending := ctl.leaseCountLocked(ctl.workConnPending, workConnRequestReserve)
			late := ctl.leaseCountLocked(ctl.workConnLate, workConnRequestReserve)
			idle := len(ctl.workConnCh)
			ctl.workConnMu.Unlock()
			require.Zero(t, pending, "the successful retry must not time out and retry again")
			require.Equal(t, 1, late)
			require.Equal(t, 1, idle)
			got, err := ctl.GetWorkConn()
			if got != nil {
				t.Cleanup(func() { _ = got.Close() })
			}
			require.NoError(t, err)
			require.Same(t, workConn, got)
		})
	}
}

func TestWorkConnLeaseMatchingPrefersPendingSupply(t *testing.T) {
	for _, tc := range []struct {
		name         string
		responseKind workConnRequestKind
		pendingKind  workConnRequestKind
		lateKind     workConnRequestKind
	}{
		{"typed demand", workConnRequestDemand, workConnRequestDemand, workConnRequestDemand},
		{"typed reserve", workConnRequestReserve, workConnRequestReserve, workConnRequestReserve},
		{"legacy demand", workConnRequestLegacy, workConnRequestDemand, workConnRequestDemand},
		{"legacy reserve before late demand", workConnRequestLegacy, workConnRequestReserve, workConnRequestDemand},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newWorkConnScheduler(1, time.Minute, xlog.New(), nil)
			t.Cleanup(s.closeWorkConnPool)
			s.workConnMu.Lock()
			late := &workConnLease{kind: tc.lateKind, active: true, late: true}
			s.workConnLate[tc.lateKind] = []*workConnLease{late}
			pending := s.newWorkConnLeaseLocked(tc.pendingKind)
			first := s.consumeWorkConnLeaseLocked(tc.responseKind)
			lateStillActive := late.active
			second := s.consumeWorkConnLeaseLocked(tc.responseKind)
			s.workConnMu.Unlock()
			require.Same(t, pending, first)
			require.True(t, lateStillActive)
			require.Same(t, late, second)
		})
	}
}

func TestWorkConnQueuedLeaseTimerUsesAdmissionBoundary(t *testing.T) {
	conn := newBlockingWriteConn()
	dispatcher := msg.NewDispatcher(msg.NewConn(conn, msg.NewV1ReadWriter(conn)))
	s := newWorkConnScheduler(1, time.Minute, xlog.New(), nil)
	s.sendRequest = func(kind workConnRequestKind) error {
		return dispatcher.Send(&msg.ReqWorkConn{WorkConnType: string(kind)})
	}
	s.dispatcherDone = func() <-chan struct{} { return dispatcher.Done() }
	t.Cleanup(func() {
		s.closeWorkConnPool()
		_ = conn.Close()
	})
	dispatcher.Run()
	require.NoError(t, dispatcher.Send(&msg.Ping{}))
	waitForSignal(t, conn.writeStarted, "initial dispatcher write to block")

	// Unlike the full-queue test, Send can enqueue here. This records the
	// remaining API boundary: a timer starts while the writer is still blocked
	// on Ping, because Dispatcher offers no per-message write acknowledgement.
	s.reconcileWorkConnRequests()
	s.workConnMu.Lock()
	leases := s.workConnPending[workConnRequestReserve]
	pending := len(leases)
	timerInstalled := pending == 1 && leases[0].timer != nil && leases[0].dispatched
	s.workConnMu.Unlock()
	require.Equal(t, 1, pending)
	require.True(t, timerInstalled)
	select {
	case <-conn.allowWrite:
		t.Fatal("writer was unexpectedly released")
	default:
	}
	_ = conn.Close()
	waitForSignal(t, dispatcher.Done(), "dispatcher shutdown")
}

type orderedWorkConnClose struct {
	*countingCloseConn
	id     int
	closed chan<- int
}

func (c *orderedWorkConnClose) Close() error {
	c.closed <- c.id
	return c.countingCloseConn.Close()
}

func TestWorkConnHandoffsPreserveFIFOAndCleanupOrder(t *testing.T) {
	s := newWorkConnScheduler(0, time.Minute, xlog.New(), nil)
	t.Cleanup(s.closeWorkConnPool)
	const count = 6
	closed := make(chan int, count)
	waiters := make([]*workConnWaiter, count)
	conns := make([]*orderedWorkConnClose, count)
	results := make([]workConnResult, count)
	for i := range waiters {
		waiters[i] = &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
		s.workConnMu.Lock()
		s.appendWorkConnWaiterLocked(waiters[i])
		s.workConnMu.Unlock()
	}
	for i := range waiters {
		conns[i] = &orderedWorkConnClose{countingCloseConn: newCountingCloseConn(), id: i, closed: closed}
		workConn := proxy.NewWorkConn(msg.NewConn(conns[i], msg.NewV1ReadWriter(conns[i])))
		accepted, err := s.registerWorkConnState(workConn, workConnRequestDemand)
		require.NoError(t, err)
		require.True(t, accepted)
		results[i] = waitForResult(t, waiters[i].resultCh, "FIFO waiter delivery")
		require.Same(t, workConn, results[i].conn)
	}
	// Consumers may run out of order. Removing middle/head/tail nodes must
	// preserve the remaining registration order used by shutdown cleanup.
	for _, i := range []int{3, 0, 5} {
		conn, err := s.consumeWorkConnWaiterResult(waiters[i], results[i])
		require.NoError(t, err)
		require.Same(t, results[i].conn, conn)
		require.NoError(t, conn.Close())
		require.Equal(t, i, waitForResult(t, closed, "consumer close"))
	}
	s.closeWorkConnPool()
	waitForSignal(t, s.workConnCleanupDone, "ordered handoff cleanup")
	for _, i := range []int{1, 2, 4} {
		require.Equal(t, i, waitForResult(t, closed, "cleanup close"))
		conn, err := s.consumeWorkConnWaiterResult(waiters[i], results[i])
		require.Nil(t, conn)
		require.ErrorIs(t, err, pkgerr.ErrCtlClosed)
	}
	for _, conn := range conns {
		require.Equal(t, int64(1), conn.closeCount.Load())
	}
}

func TestWorkConnConcurrentHandoffConsumptionAndClose(t *testing.T) {
	s := newWorkConnScheduler(0, time.Minute, xlog.New(), nil)
	t.Cleanup(s.closeWorkConnPool)
	const count = 512
	waiters := make([]*workConnWaiter, count)
	conns := make([]*countingCloseConn, count)
	results := make([]workConnResult, count)
	for i := range waiters {
		waiters[i] = &workConnWaiter{resultCh: make(chan workConnResult, 1), active: true}
		s.workConnMu.Lock()
		s.appendWorkConnWaiterLocked(waiters[i])
		s.workConnMu.Unlock()
	}
	for i := range waiters {
		conns[i] = newCountingCloseConn()
		accepted, err := s.registerWorkConnState(
			proxy.NewWorkConn(msg.NewConn(conns[i], msg.NewV1ReadWriter(conns[i]))), workConnRequestDemand,
		)
		require.NoError(t, err)
		require.True(t, accepted)
		results[i] = waitForResult(t, waiters[i].resultCh, "burst waiter delivery")
	}
	// Guarantee coverage of both ownership outcomes in addition to the race:
	// one quarter claims before close, one quarter only attempts after close.
	for i := range count / 4 {
		conn, err := s.consumeWorkConnWaiterResult(waiters[i], results[i])
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	}
	start := make(chan struct{})
	closed := make(chan struct{})
	outcomes := make(chan workConnResult, count)
	var ready sync.WaitGroup
	ready.Add(count - count/4)
	for i := count / 4; i < count; i++ {
		go func() {
			ready.Done()
			if i >= 3*count/4 {
				<-closed
			} else {
				<-start
			}
			conn, err := s.consumeWorkConnWaiterResult(waiters[i], results[i])
			if conn != nil {
				_ = conn.Close()
			}
			outcomes <- workConnResult{conn: conn, err: err}
		}()
	}
	ready.Wait()
	close(start)
	s.closeWorkConnPool()
	close(closed)
	for i := count / 4; i < count; i++ {
		outcome := waitForResult(t, outcomes, "handoff consumer to finish")
		if outcome.err != nil {
			require.ErrorIs(t, outcome.err, pkgerr.ErrCtlClosed)
			require.Nil(t, outcome.conn)
		} else {
			require.NotNil(t, outcome.conn)
		}
	}
	waitForSignal(t, s.workConnCleanupDone, "concurrent handoff cleanup")
	s.workConnMu.Lock()
	remaining := s.workConnHandoffs.Len()
	retained := false
	for _, waiter := range waiters {
		retained = retained || waiter.handoffConn != nil || waiter.handoffElement != nil
	}
	s.workConnMu.Unlock()
	require.Zero(t, remaining)
	require.False(t, retained)
	for _, conn := range conns {
		require.Equal(t, int64(1), conn.closeCount.Load(), "consumer and cleanup must never both close the same connection")
	}
}

func BenchmarkWorkConnPoolHit(b *testing.B) {
	s := newWorkConnScheduler(1, time.Minute, xlog.New(), nil)
	// Isolate acquisition from background refill work. The coalesced trigger
	// remains buffered; every iteration restores the same idle connection.
	s.workConnReconcileOnce.Do(func() {})
	conn := &proxy.WorkConn{}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.workConnCh <- conn
		got, err := s.GetWorkConn()
		if err != nil || got != conn {
			b.Fatalf("unexpected pool result: conn=%p err=%v", got, err)
		}
	}
}

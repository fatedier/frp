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
	"container/list"
	"errors"
	"fmt"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	pkgerr "github.com/fatedier/frp/pkg/errors"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/server/proxy"
)

const workConnPoolCapacityOffset = 10

const (
	workConnReserveWindowMax = 8
	workConnDemandWindow     = 64
	workConnLegacyWindow     = 64

	// Canceled waiters remain as lazy tombstones so cancellation stays O(1).
	// Compact only after enough tombstones have accumulated and comprise at
	// least half the queue, making the occasional O(n) copy O(1) amortized.
	workConnWaiterCompactMinTombstones = 64
)

var (
	errWorkConnPoolFull           = errors.New("work connection pool is full, discarding")
	errWorkConnControlUnavailable = errors.New("work connection control unavailable")
)

type workConnRequestKind string

const (
	workConnRequestDemand  workConnRequestKind = msg.WorkConnTypeDemand
	workConnRequestReserve workConnRequestKind = msg.WorkConnTypeReserve
	workConnRequestLegacy  workConnRequestKind = "legacy"
)

type workConnWaiter struct {
	resultCh       chan workConnResult
	active         bool
	handoffConn    *proxy.WorkConn
	handoffClosed  bool
	handoffElement *list.Element
}

type workConnResult struct {
	conn *proxy.WorkConn
	err  error
}

type workConnLease struct {
	kind  workConnRequestKind
	timer *time.Timer

	active     bool
	dispatched bool
	late       bool
}

// workConnScheduler owns all work-connection supply state. It deliberately
// communicates with Control through callbacks for request dispatch and
// dispatcher shutdown; it never acquires Control lifecycle or manager locks.
type workConnScheduler struct {
	workConnCh chan *proxy.WorkConn

	workConnMu               sync.Mutex
	workConnWaiters          []*workConnWaiter
	workConnActiveWaiters    int
	workConnWaiterTombstones int
	// Elements retain registration order for cleanup; each waiter owns its
	// element so consumption removes it in O(1), even during large bursts.
	workConnHandoffs   list.List
	workConnPending    map[workConnRequestKind][]*workConnLease
	workConnLate       map[workConnRequestKind][]*workConnLease
	workConnPoolClosed bool

	workConnReconcileCh       chan struct{}
	workConnReconcileStop     chan struct{}
	workConnReconcileDone     chan struct{}
	workConnReconcileOnce     sync.Once
	workConnReconcileStopOnce sync.Once

	workConnCleanupMu      sync.Mutex
	workConnCleanupPending []*proxy.WorkConn
	workConnCleanupReady   bool
	workConnCleanupStarted bool
	workConnCleanupDone    chan struct{}

	workConnLegacyMode    bool
	workConnReserveWindow int
	workConnLeaseTimeout  time.Duration
	poolCount             int

	// sendRequest is called only after workConnMu has been released. The
	// callback captures the Control's immutable generation metadata and sends
	// the wire message without exposing lifecycle state to the scheduler.
	sendRequest    func(workConnRequestKind) error
	dispatcherDone func() <-chan struct{}
	logger         *xlog.Logger
	doneCh         <-chan struct{}
}

func newWorkConnScheduler(poolCount int, leaseTimeout time.Duration, logger *xlog.Logger, doneCh <-chan struct{}) workConnScheduler {
	return workConnScheduler{
		workConnCh:            make(chan *proxy.WorkConn, poolCount+workConnPoolCapacityOffset),
		poolCount:             poolCount,
		workConnPending:       make(map[workConnRequestKind][]*workConnLease),
		workConnLate:          make(map[workConnRequestKind][]*workConnLease),
		workConnReconcileCh:   make(chan struct{}, 1),
		workConnReconcileStop: make(chan struct{}),
		workConnReconcileDone: make(chan struct{}),
		workConnCleanupDone:   make(chan struct{}),
		workConnLegacyMode:    true,
		workConnReserveWindow: min(poolCount, workConnReserveWindowMax),
		workConnLeaseTimeout:  leaseTimeout,
		logger:                logger,
		doneCh:                doneCh,
	}
}

func normalizeWorkConnRequestKind(kind workConnRequestKind) workConnRequestKind {
	if kind == workConnRequestDemand || kind == workConnRequestReserve {
		return kind
	}
	return workConnRequestLegacy
}

func (s *workConnScheduler) workConnLeaseTimeoutValue() time.Duration {
	if s.workConnLeaseTimeout > 0 {
		return s.workConnLeaseTimeout
	}
	return 10 * time.Second
}

func (s *workConnScheduler) startWorkConnReconciler() {
	s.workConnReconcileOnce.Do(func() {
		go func() {
			defer close(s.workConnReconcileDone)
			for {
				select {
				case <-s.workConnReconcileCh:
					s.reconcileWorkConnRequests()
				case <-s.workConnReconcileStop:
					return
				}
			}
		}()
	})
}

// requestWorkConnReconcile coalesces refill requests into one bounded trigger.
// The reconciler may block on dispatcher backpressure, but it owns no control
// lifecycle or generation locks and is stopped with the control pool.
func (s *workConnScheduler) requestWorkConnReconcile() {
	s.workConnMu.Lock()
	closed := s.workConnPoolClosed
	s.workConnMu.Unlock()
	if closed {
		return
	}

	s.startWorkConnReconciler()
	select {
	case <-s.workConnReconcileStop:
		return
	case s.workConnReconcileCh <- struct{}{}:
	default:
	}
}

func (s *workConnScheduler) activeWaiterCountLocked() int {
	return s.workConnActiveWaiters
}

func (s *workConnScheduler) appendWorkConnWaiterLocked(waiter *workConnWaiter) {
	s.workConnWaiters = append(s.workConnWaiters, waiter)
	if waiter.active {
		s.workConnActiveWaiters++
	}
}

func (s *workConnScheduler) popWorkConnWaiterLocked() *workConnWaiter {
	for len(s.workConnWaiters) > 0 {
		waiter := s.workConnWaiters[0]
		s.workConnWaiters[0] = nil
		s.workConnWaiters = s.workConnWaiters[1:]
		if waiter.active {
			waiter.active = false
			s.workConnActiveWaiters--
			if len(s.workConnWaiters) == 0 {
				s.workConnWaiters = nil
			}
			return waiter
		}
		if s.workConnWaiterTombstones > 0 {
			s.workConnWaiterTombstones--
		}
	}
	s.workConnWaiters = nil
	s.workConnWaiterTombstones = 0
	return nil
}

func (s *workConnScheduler) compactWorkConnWaitersLocked() {
	if s.workConnWaiterTombstones == 0 {
		return
	}
	activeWaiters := make([]*workConnWaiter, 0, s.workConnActiveWaiters)
	for _, waiter := range s.workConnWaiters {
		if waiter.active {
			activeWaiters = append(activeWaiters, waiter)
		}
	}
	s.workConnWaiters = activeWaiters
	s.workConnWaiterTombstones = 0
}

func (s *workConnScheduler) shouldCompactWorkConnWaitersLocked() bool {
	tombstones := s.workConnWaiterTombstones
	if tombstones < workConnWaiterCompactMinTombstones {
		return false
	}
	return tombstones*2 >= len(s.workConnWaiters)
}

func (s *workConnScheduler) removeWorkConnHandoffLocked(target *workConnWaiter) {
	if target.handoffElement != nil {
		s.workConnHandoffs.Remove(target.handoffElement)
		target.handoffElement = nil
	}
}

func (s *workConnScheduler) removeLeaseLocked(leases map[workConnRequestKind][]*workConnLease, target *workConnLease) {
	items := leases[target.kind]
	for i, lease := range items {
		if lease == target {
			items = append(items[:i], items[i+1:]...)
			if len(items) == 0 {
				delete(leases, target.kind)
			} else {
				leases[target.kind] = items
			}
			return
		}
	}
}

func (s *workConnScheduler) hasLeaseLocked(leases map[workConnRequestKind][]*workConnLease, target *workConnLease) bool {
	return slices.Contains(leases[target.kind], target)
}

func (s *workConnScheduler) popLeaseLocked(leases map[workConnRequestKind][]*workConnLease, kind workConnRequestKind) *workConnLease {
	items := leases[kind]
	for len(items) > 0 {
		lease := items[0]
		items = items[1:]
		if len(items) == 0 {
			delete(leases, kind)
		} else {
			leases[kind] = items
		}
		if lease.active {
			return lease
		}
	}
	return nil
}

func (s *workConnScheduler) leaseCountLocked(leases map[workConnRequestKind][]*workConnLease, kind workConnRequestKind) int {
	count := 0
	for _, lease := range leases[kind] {
		if lease.active {
			count++
		}
	}
	return count
}

func (s *workConnScheduler) workConnSupplyGapLocked() bool {
	target := s.activeWaiterCountLocked() + s.poolCount
	pending := s.leaseCountLocked(s.workConnPending, workConnRequestDemand) +
		s.leaseCountLocked(s.workConnPending, workConnRequestReserve)
	return len(s.workConnCh)+pending < target
}

func (s *workConnScheduler) consumeWorkConnLeaseLocked(kind workConnRequestKind) *workConnLease {
	kind = normalizeWorkConnRequestKind(kind)
	s.workConnLegacyMode = kind == workConnRequestLegacy
	kinds := []workConnRequestKind{kind}
	if s.workConnLegacyMode {
		// Untyped responses satisfy pending demand before pending reserve.
		kinds = []workConnRequestKind{workConnRequestDemand, workConnRequestReserve}
	}
	// There is no request ID: same-kind replies are interchangeable. A timely
	// retry must retire pending supply before late leases, otherwise its reply
	// is discarded while the retry remains pending and the pool never recovers.
	// For legacy clients, all pending kinds take precedence over late leases.
	for _, leases := range []map[workConnRequestKind][]*workConnLease{s.workConnPending, s.workConnLate} {
		for _, candidate := range kinds {
			if lease := s.popLeaseLocked(leases, candidate); lease != nil {
				lease.active = false
				if lease.timer != nil {
					lease.timer.Stop()
				}
				return lease
			}
		}
	}
	return nil
}

func (s *workConnScheduler) expireLateWorkConnLease(lease *workConnLease) {
	s.workConnMu.Lock()
	if !lease.active || !lease.late {
		s.workConnMu.Unlock()
		return
	}
	lease.active = false
	s.removeLeaseLocked(s.workConnLate, lease)
	s.workConnMu.Unlock()
	s.requestWorkConnReconcile()
}

func (s *workConnScheduler) expireWorkConnLease(lease *workConnLease) {
	s.workConnMu.Lock()
	if !lease.active || lease.late {
		s.workConnMu.Unlock()
		return
	}
	s.removeLeaseLocked(s.workConnPending, lease)
	lease.late = true
	s.workConnLate[lease.kind] = append(s.workConnLate[lease.kind], lease)
	lease.timer = time.AfterFunc(s.workConnLeaseTimeoutValue(), func() {
		s.expireLateWorkConnLease(lease)
	})
	s.workConnMu.Unlock()
	s.requestWorkConnReconcile()
}

func (s *workConnScheduler) newWorkConnLeaseLocked(kind workConnRequestKind) *workConnLease {
	lease := &workConnLease{
		kind:   kind,
		active: true,
	}
	s.workConnPending[kind] = append(s.workConnPending[kind], lease)
	return lease
}

// startWorkConnLeaseTimer starts the response timeout only after the request
// has been accepted by Dispatcher.Send. A response may race this call; the
// active/pending check under workConnMu prevents installing a timer on a lease
// that was already consumed, failed, or closed.
// Send acknowledges queue admission, not WriteMsg completion. The dispatcher
// has no per-message write acknowledgement, so this timeout includes queue time.
func (s *workConnScheduler) startWorkConnLeaseTimer(lease *workConnLease) {
	s.workConnMu.Lock()
	if lease.active && !lease.dispatched && !lease.late && lease.timer == nil && s.hasLeaseLocked(s.workConnPending, lease) {
		lease.dispatched = true
		lease.timer = time.AfterFunc(s.workConnLeaseTimeoutValue(), func() {
			s.expireWorkConnLease(lease)
		})
	}
	s.workConnMu.Unlock()
}

func (s *workConnScheduler) registerWorkConn(conn *proxy.WorkConn, kind workConnRequestKind) (bool, error) {
	accepted, err := s.registerWorkConnState(conn, kind)
	s.reconcileWorkConnRequests()
	return accepted, err
}

// registerWorkConnState commits work connection ownership and lease accounting
// without sending refill requests. Callers that hold lifecycle or generation
// locks must release them before invoking reconcileWorkConnRequests.
func (s *workConnScheduler) registerWorkConnState(conn *proxy.WorkConn, kind workConnRequestKind) (bool, error) {
	s.workConnMu.Lock()
	if s.workConnPoolClosed {
		s.workConnMu.Unlock()
		return false, pkgerr.ErrCtlClosed
	}
	s.consumeWorkConnLeaseLocked(kind)
	if waiter := s.popWorkConnWaiterLocked(); waiter != nil {
		// A reserve response may satisfy a demand waiter, but any other
		// pending lease is still an in-flight source of supply until it really
		// times out. Keep it accounted for instead of creating a replacement.
		waiter.handoffConn = conn
		waiter.handoffElement = s.workConnHandoffs.PushBack(waiter)
		waiter.resultCh <- workConnResult{conn: conn}
		s.workConnMu.Unlock()
		return true, nil
	}
	// Any response that did not satisfy a waiter must still prove a real supply
	// gap before entering idle. This covers unmatched responses, demand leases
	// whose waiter timed out, and late leases whose replacement is already
	// pending. A matching lease has been consumed above, so pending includes
	// only the remaining supply credits; replies cannot identify their own lease.
	if !s.workConnSupplyGapLocked() {
		s.workConnMu.Unlock()
		return false, errWorkConnPoolFull
	}

	select {
	case s.workConnCh <- conn:
		s.workConnMu.Unlock()
		return true, nil
	default:
		s.workConnMu.Unlock()
		return false, errWorkConnPoolFull
	}
}

// consumeWorkConnWaiterResult transfers a direct waiter handoff to its
// consumer only while the control pool is still open. A close/replacement may
// have already detached the connection for cleanup even though the buffered
// result notification remains readable; in that case the caller must never
// receive the stale connection.
func (s *workConnScheduler) consumeWorkConnWaiterResult(waiter *workConnWaiter, result workConnResult) (*proxy.WorkConn, error) {
	s.workConnMu.Lock()
	if s.workConnPoolClosed || waiter.handoffClosed {
		s.workConnMu.Unlock()
		return nil, pkgerr.ErrCtlClosed
	}
	conn := waiter.handoffConn
	if conn != nil {
		waiter.handoffConn = nil
		s.removeWorkConnHandoffLocked(waiter)
	}
	s.workConnMu.Unlock()
	if result.err != nil {
		return nil, result.err
	}
	if conn == nil {
		conn = result.conn
	}
	return conn, nil
}

func (s *workConnScheduler) cancelWorkConnWaiter(waiter *workConnWaiter) bool {
	canceled := false
	s.workConnMu.Lock()
	if waiter.active {
		waiter.active = false
		s.workConnActiveWaiters--
		s.workConnWaiterTombstones++
		if s.shouldCompactWorkConnWaitersLocked() {
			s.compactWorkConnWaitersLocked()
		}
		canceled = true
	}
	s.workConnMu.Unlock()
	if canceled {
		s.requestWorkConnReconcile()
	}
	return canceled
}

func (s *workConnScheduler) startWorkConnCleanup() {
	s.workConnCleanupMu.Lock()
	if s.workConnCleanupStarted || !s.workConnCleanupReady {
		s.workConnCleanupMu.Unlock()
		return
	}
	conns := s.workConnCleanupPending
	s.workConnCleanupPending = nil
	s.workConnCleanupStarted = true
	s.workConnCleanupMu.Unlock()

	if len(conns) == 0 {
		close(s.workConnCleanupDone)
		return
	}
	go func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
		close(s.workConnCleanupDone)
	}()
}

func (s *workConnScheduler) waitWorkConnCleanup() {
	s.workConnCleanupMu.Lock()
	started := s.workConnCleanupStarted
	s.workConnCleanupMu.Unlock()
	if started {
		<-s.workConnCleanupDone
	}
}

func (s *workConnScheduler) takeWorkConnPool() {
	var conns []*proxy.WorkConn
	// Serialize pool extraction with cleanup startup. A concurrent worker may
	// observe the pool as closed before this caller has queued its streams;
	// keeping the cleanup gate held makes that ordering explicit and prevents
	// a premature workConnCleanupDone close from losing the extracted conns.
	s.workConnCleanupMu.Lock()
	s.workConnMu.Lock()
	if s.workConnPoolClosed {
		s.workConnMu.Unlock()
		s.workConnCleanupMu.Unlock()
		return
	}
	s.workConnPoolClosed = true
	s.workConnReconcileStopOnce.Do(func() { close(s.workConnReconcileStop) })
	for _, waiter := range s.workConnWaiters {
		if waiter.active {
			waiter.active = false
			s.workConnActiveWaiters--
			waiter.resultCh <- workConnResult{err: pkgerr.ErrCtlClosed}
		}
	}
	s.workConnWaiters = nil
	s.workConnActiveWaiters = 0
	s.workConnWaiterTombstones = 0
	for _, leases := range s.workConnPending {
		for _, lease := range leases {
			lease.active = false
			if lease.timer != nil {
				lease.timer.Stop()
			}
		}
	}
	for _, leases := range s.workConnLate {
		for _, lease := range leases {
			lease.active = false
			if lease.timer != nil {
				lease.timer.Stop()
			}
		}
	}
	for element := s.workConnHandoffs.Front(); element != nil; element = element.Next() {
		waiter := element.Value.(*workConnWaiter)
		if waiter.handoffConn != nil {
			conns = append(conns, waiter.handoffConn)
			waiter.handoffConn = nil
		}
		waiter.handoffClosed = true
		waiter.handoffElement = nil
	}
	s.workConnHandoffs.Init()
	for {
		select {
		case conn := <-s.workConnCh:
			conns = append(conns, conn)
		default:
			close(s.workConnCh)
			s.workConnMu.Unlock()
			s.workConnCleanupPending = append(s.workConnCleanupPending, conns...)
			s.workConnCleanupReady = true
			s.workConnCleanupMu.Unlock()
			return
		}
	}
}

func (s *workConnScheduler) closeWorkConnPool() {
	s.takeWorkConnPool()
	s.startWorkConnCleanup()
}

func (s *workConnScheduler) reconcileWorkConnRequests() {
	type request struct {
		lease *workConnLease
		kind  workConnRequestKind
	}

	s.workConnMu.Lock()
	if s.workConnPoolClosed {
		s.workConnMu.Unlock()
		return
	}
	var requests []request
	demandPending := s.leaseCountLocked(s.workConnPending, workConnRequestDemand)
	reservePending := s.leaseCountLocked(s.workConnPending, workConnRequestReserve)
	activeWaiters := s.activeWaiterCountLocked()
	demandNeed := activeWaiters - demandPending
	if demandNeed > 0 {
		budget := workConnDemandWindow - demandPending
		if s.workConnLegacyMode {
			legacyBudget := workConnLegacyWindow - demandPending - reservePending
			if budget > legacyBudget {
				budget = legacyBudget
			}
		}
		if budget < 0 {
			budget = 0
		}
		if demandNeed > budget {
			demandNeed = budget
		}
		for i := 0; i < demandNeed; i++ {
			requests = append(requests, request{lease: s.newWorkConnLeaseLocked(workConnRequestDemand), kind: workConnRequestDemand})
		}
		demandPending += demandNeed
	}

	// Keep the invariant available + pending >= waiters + poolCount, while
	// limiting only reserve requests to the sliding window. Late leases do not
	// consume retry budget; their responses are admitted separately above only
	// when this projected supply still has a real gap.
	reserveNeed := max(activeWaiters+s.poolCount-len(s.workConnCh)-demandPending-reservePending, 0)
	reserveBudget := s.workConnReserveWindow - reservePending
	if s.workConnLegacyMode {
		legacyBudget := workConnLegacyWindow - demandPending - reservePending
		if reserveBudget > legacyBudget {
			reserveBudget = legacyBudget
		}
	}
	if reserveBudget < 0 {
		reserveBudget = 0
	}
	if reserveNeed > reserveBudget {
		reserveNeed = reserveBudget
	}
	for i := 0; i < reserveNeed; i++ {
		requests = append(requests, request{lease: s.newWorkConnLeaseLocked(workConnRequestReserve), kind: workConnRequestReserve})
	}
	s.workConnMu.Unlock()

	for _, req := range requests {
		if err := s.sendRequest(req.kind); err != nil {
			s.failWorkConnLease(req.lease)
			if s.dispatcherDone != nil {
				select {
				case <-s.dispatcherDone():
					// Wake waiters immediately when the control transport is already
					// closed; otherwise they would wait until UserConnTimeout.
					s.closeWorkConnPool()
				default:
				}
			}
		} else {
			s.startWorkConnLeaseTimer(req.lease)
		}
	}
}

func (s *workConnScheduler) failWorkConnLease(lease *workConnLease) {
	s.workConnMu.Lock()
	if lease.active && !lease.late {
		lease.active = false
		s.removeLeaseLocked(s.workConnPending, lease)
		if lease.timer != nil {
			lease.timer.Stop()
		}
	}
	s.workConnMu.Unlock()
}

// When frps get one user connection, we get one work connection from the pool and return it.
// If no workConn available in the pool, send message to frpc to get one or more
// and wait until it is available.
// return an error if wait timeout
func (s *workConnScheduler) GetWorkConn() (workConn *proxy.WorkConn, err error) {
	xl := s.logger
	started := time.Now()
	defer func() {
		if recovered := recover(); recovered != nil {
			xl.Errorf("panic error: %v", recovered)
			xl.Errorf(string(debug.Stack()))
			workConn = nil
			err = pkgerr.ErrCtlClosed
		}
	}()

	s.workConnMu.Lock()
	if s.workConnPoolClosed {
		s.workConnMu.Unlock()
		return nil, pkgerr.ErrCtlClosed
	}
	// Get an idle connection first.
	select {
	case workConn = <-s.workConnCh:
		s.workConnMu.Unlock()
		xl.Debugf("get work connection from pool")
		s.requestWorkConnReconcile()
		return workConn, nil
	default:
	}

	waiter := &workConnWaiter{
		resultCh: make(chan workConnResult, 1),
		active:   true,
	}
	s.appendWorkConnWaiterLocked(waiter)
	s.workConnMu.Unlock()

	// Pool hits need no timer. On a miss, include time already spent acquiring
	// the pool lock and install the timer before requesting a refill.
	timer := time.NewTimer(s.workConnLeaseTimeoutValue() - time.Since(started))
	defer timer.Stop()
	s.requestWorkConnReconcile()
	select {
	case result := <-waiter.resultCh:
		return s.consumeWorkConnWaiterResult(waiter, result)
	case <-timer.C:
		if !s.cancelWorkConnWaiter(waiter) {
			select {
			case result := <-waiter.resultCh:
				return s.consumeWorkConnWaiterResult(waiter, result)
			default:
			}
		}
		s.workConnMu.Lock()
		closed := s.workConnPoolClosed
		s.workConnMu.Unlock()
		if closed {
			return nil, pkgerr.ErrCtlClosed
		}
		err = fmt.Errorf("timeout trying to get work connection")
		xl.Warnf("%v", err)
		return nil, err
	case <-s.doneCh:
		// doneCh is terminal for this control generation. Do not consume a
		// buffered successful handoff here: takeWorkConnPool detaches that
		// connection for cleanup, and returning it would leak it past shutdown.
		s.cancelWorkConnWaiter(waiter)
		return nil, pkgerr.ErrCtlClosed
	}
}

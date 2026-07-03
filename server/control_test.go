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
	"net"
	"testing"
	"time"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/xlog"
)

// newTestControl builds a minimal Control sufficient for lifecycle tests.
func newTestControl(t *testing.T) *Control {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})
	return &Control{
		sessionCtx: &SessionContext{Conn: msg.NewConn(serverConn, nil)},
		runID:      "runid",
		xl:         xlog.New(),
		doneCh:     make(chan struct{}),
	}
}

// A control that is replaced before it is ever started must still have its
// doneCh closed, so that WaitClosed - called from the re-login path in
// RegisterControl - returns instead of blocking forever. Without this, a
// control whose connection died silently (tcpMux) wedges a goroutine plus the
// whole Control graph permanently on every re-login. Regression test for #5391.
func TestControlReplacedBeforeStartUnblocksWaitClosed(t *testing.T) {
	ctl := newTestControl(t)
	newCtl := &Control{runID: "newrunid", xl: xlog.New()}

	ctl.Replaced(newCtl)

	done := make(chan struct{})
	go func() {
		ctl.WaitClosed()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("WaitClosed blocked after Replaced on a never-started control")
	}
}

// Replaced must not close doneCh for a control that is already running: its
// worker owns that lifecycle and closing it here would double-close / race.
func TestControlReplacedAfterStartLeavesDoneToWorker(t *testing.T) {
	ctl := newTestControl(t)
	ctl.started = true // simulate a started control without launching worker
	newCtl := &Control{runID: "newrunid", xl: xlog.New()}

	ctl.Replaced(newCtl)

	select {
	case <-ctl.doneCh:
		t.Fatal("Replaced closed doneCh for a started control; worker should own it")
	case <-time.After(100 * time.Millisecond):
		// expected: doneCh remains open, worker will close it
	}
}

// Del must only remove (and report removal of) the control still registered for a
// run id. A superseded control's late-running login handler relies on this to avoid
// offlining the run id now owned by its replacement.
func TestControlManagerDelOnlyRemovesActiveControl(t *testing.T) {
	cm := NewControlManager()
	ctlOld := newTestControl(t)
	ctlNew := newTestControl(t)

	cm.Add("R", ctlOld)
	cm.Add("R", ctlNew) // replaces ctlOld; ctlNew is now the registered control

	if cm.Del("R", ctlOld) {
		t.Fatal("Del reported removal for a superseded control")
	}
	if got, ok := cm.GetByID("R"); !ok || got != ctlNew {
		t.Fatal("stale Del affected the active control")
	}
	if !cm.Del("R", ctlNew) {
		t.Fatal("Del did not remove the active control")
	}
}

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
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/wire"
)

func TestWorkConnBoundToControlSession(t *testing.T) {
	t.Parallel()

	const (
		ownSession = iota
		foreignSession
		noSession
	)
	for _, tc := range []struct {
		name             string
		controlOnSession bool
		workConnSession  int
		wantErr          bool
	}{
		{name: "work connection on the control's session is accepted", controlOnSession: true, workConnSession: ownSession},
		{name: "work connection on another session is rejected", controlOnSession: true, workConnSession: foreignSession, wantErr: true},
		{name: "work connection without a session is rejected", controlOnSession: true, workConnSession: noSession, wantErr: true},
		{name: "session work connection for a control without one is rejected", workConnSession: foreignSession, wantErr: true},
		{name: "mux disabled on both sides is accepted", workConnSession: noSession},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svr := newControlTestService(t)

			var ctl *Control
			var session io.Closer
			if tc.controlOnSession {
				var own io.Closer
				ctl, _, own, _ = newYamuxTestControl(t, svr)
				if tc.workConnSession == ownSession {
					session = own
				}
			} else {
				var err error
				ctl, _, err = registerLifecycleTestControl(svr)
				require.NoError(t, err)
				require.NoError(t, svr.completeControlLogin(ctl, func() error { return nil }))
				t.Cleanup(func() {
					_ = ctl.Close()
					waitForControlDone(t, ctl)
				})
			}
			if tc.workConnSession == foreignSession {
				_, session, _, _ = newYamuxTestSession(t)
			}

			conn := newCountingCloseConn()
			workConn := msg.NewConn(conn, msg.NewV1ReadWriter(conn))
			err := svr.RegisterWorkConn(workConn, &msg.NewWorkConn{RunID: "shared-run"}, wire.ProtocolV1, false, session)

			if tc.wantErr {
				require.ErrorContains(t, err, "no client control found")
				require.Empty(t, ctl.workConnCh)
				return
			}
			require.NoError(t, err)
			require.Len(t, ctl.workConnCh, 1)
		})
	}
}

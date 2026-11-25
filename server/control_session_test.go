// Copyright 2026 fatedier, fatedier@gmail.com
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
	"net"
	"testing"
	"time"

	fmux "github.com/fatedier/yamux"
	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/wire"
	"github.com/fatedier/frp/pkg/util/util"
)

func TestControlReplacementClosesYamuxSession(t *testing.T) {
	svr := newControlTestService(t)
	_, clientA, serverA, _ := newYamuxTestControl(t, svr)
	_, _, _, connB := newYamuxTestControl(t, svr)

	waitForSignal(t, serverA.CloseChan(), "replaced control's server yamux session to close")
	waitForSignal(t, clientA.CloseChan(), "replaced control's client yamux session to close")
	_, err := clientA.OpenStream()
	require.ErrorIs(t, err, fmux.ErrSessionShutdown)

	require.NoError(t, connB.SetDeadline(time.Now().Add(3*time.Second)))
	require.NoError(t, connB.WriteMsg(&msg.Ping{}))
	response, err := connB.ReadMsg()
	require.NoError(t, err, "replacement control must still receive a Pong")
	require.Equal(t, &msg.Pong{}, response)
}

func TestControlCloseClosesYamuxSession(t *testing.T) {
	ctl, client, server, _ := newYamuxTestControl(t, newControlTestService(t))

	require.NoError(t, ctl.Close())
	waitForSignal(t, server.CloseChan(), "closed control's server yamux session to close")
	waitForSignal(t, client.CloseChan(), "closed control's client yamux session to close")
	_, err := client.OpenStream()
	require.ErrorIs(t, err, fmux.ErrSessionShutdown)
}

func TestControlConcurrentReplacementClosesLosingSessions(t *testing.T) {
	svr := newControlTestService(t)
	ctlA, clientA, serverA, connA := newYamuxTestControl(t, svr)
	type result struct {
		ctl            *Control
		client, server *fmux.Session
		conn           *msg.Conn
		err            error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for range 2 {
		client, server, clientConn, serverConn := newYamuxTestSession(t)
		go func() {
			r := result{client: client, server: server, conn: clientConn}
			select {
			case <-start:
				r.ctl, r.err = svr.RegisterControl(serverConn, &msg.Login{
					RunID:        "shared-run",
					ClientID:     "client",
					Timestamp:    1,
					PrivilegeKey: util.GetAuthKey("", 1),
				}, false, wire.ProtocolV1, "", server)
				if r.err == nil {
					r.err = svr.completeControlLogin(r.ctl, func() error { return nil })
				}
				if r.err != nil && r.ctl != nil {
					svr.ctlManager.Remove(r.ctl)
					_ = r.ctl.Close()
				}
			case <-ctx.Done():
				r.err = ctx.Err()
			}
			results <- r
		}()
	}
	close(start)
	controls := [3]result{{ctl: ctlA, client: clientA, server: serverA, conn: connA}}
	for i := range 2 {
		r := waitForResult(t, results, "concurrent control login")
		if r.ctl != nil {
			t.Cleanup(func() {
				_ = r.ctl.Close()
				waitForControlDone(t, r.ctl)
			})
		}
		if r.err != nil {
			require.ErrorIs(t, r.err, errControlReplaced)
		}
		controls[i+1] = r
	}

	current, ok := svr.ctlManager.GetByID("shared-run")
	require.True(t, ok, "a running current control must remain")
	require.NotSame(t, ctlA, current)
	svr.ctlManager.mu.RLock()
	currentCount := len(svr.ctlManager.ctlsByRunID)
	svr.ctlManager.mu.RUnlock()
	require.Equal(t, 1, currentCount)
	winners := 0
	for _, r := range controls {
		if r.ctl == current {
			winners++
			require.NoError(t, r.err)
			require.False(t, r.server.IsClosed(), "winner's server session must stay open")
			require.False(t, r.client.IsClosed(), "winner's client session must stay open")
			require.NoError(t, r.conn.SetDeadline(time.Now().Add(3*time.Second)))
			require.NoError(t, r.conn.WriteMsg(&msg.Ping{}))
			response, err := r.conn.ReadMsg()
			require.NoError(t, err, "winner must still receive a Pong")
			require.Equal(t, &msg.Pong{}, response)
			continue
		}
		waitForSignal(t, r.server.CloseChan(), "losing control's server yamux session to close")
		waitForSignal(t, r.client.CloseChan(), "losing control's client yamux session to close")
		_, err := r.client.OpenStream()
		require.ErrorIs(t, err, fmux.ErrSessionShutdown)
	}
	require.Equal(t, 1, winners)
}

func newYamuxTestSession(t *testing.T) (*fmux.Session, *fmux.Session, *msg.Conn, *msg.Conn) {
	t.Helper()
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	require.NoError(t, listener.SetDeadline(time.Now().Add(3*time.Second)))
	clientTCP, err := net.DialTimeout("tcp", listener.Addr().String(), 3*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = clientTCP.Close() })
	serverTCP, err := listener.Accept()
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverTCP.Close() })

	client, err := fmux.Client(clientTCP, fmux.DefaultConfig())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	server, err := fmux.Server(serverTCP, fmux.DefaultConfig())
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Close() })
	clientStream, err := client.OpenStream()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	serverStream, err := server.AcceptStreamWithContext(ctx)
	require.NoError(t, err)
	return client, server, msg.NewConn(clientStream, msg.NewV1ReadWriter(clientStream)),
		msg.NewConn(serverStream, msg.NewV1ReadWriter(serverStream))
}

func newYamuxTestControl(t *testing.T, svr *Service) (*Control, *fmux.Session, *fmux.Session, *msg.Conn) {
	t.Helper()
	client, server, clientConn, serverConn := newYamuxTestSession(t)
	ctl, err := svr.RegisterControl(serverConn, &msg.Login{
		RunID:        "shared-run",
		ClientID:     "client",
		Timestamp:    1,
		PrivilegeKey: util.GetAuthKey("", 1),
	}, false, wire.ProtocolV1, "", server)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = ctl.Close()
		waitForControlDone(t, ctl)
	})
	require.NoError(t, svr.completeControlLogin(ctl, func() error { return nil }))
	return ctl, client, server, clientConn
}

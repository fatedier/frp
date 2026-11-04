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

package net

import (
	"bytes"
	"errors"
	"net"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	kcp "github.com/xtaci/kcp-go/v5"
)

func kcpGoroutineDump(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, pprof.Lookup("goroutine").WriteTo(&buf, 2))
	return buf.String()
}

func TestKCPListenerCloseCancelsPendingHandoff(t *testing.T) {
	l, err := ListenKcp("127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = l.Close()
		// Also release a stuck handoff when this test runs against the old code.
		select {
		case conn := <-l.acceptCh:
			if conn != nil {
				_ = conn.Close()
			}
		default:
		}
	})

	sessionsBefore := atomic.LoadUint64(&kcp.DefaultSnmp.CurrEstab)
	peer, err := kcp.DialWithOptions(l.Addr().String(), nil, 10, 3)
	require.NoError(t, err)
	t.Cleanup(func() { _ = peer.Close() })
	peer.SetNoDelay(1, 10, 2, 1)
	require.NoError(t, peer.SetWriteDeadline(time.Now().Add(5*time.Second)))
	_, err = peer.Write([]byte("pending handoff"))
	require.NoError(t, err)

	var loopID string
	require.Eventually(t, func() bool {
		for stack := range strings.SplitSeq(kcpGoroutineDump(t), "\n\n") {
			if !strings.Contains(stack, "github.com/fatedier/frp/pkg/util/net.ListenKcp.func1()") ||
				strings.Contains(stack, "github.com/xtaci/kcp-go/v5.(*Listener).AcceptKCP(") {
				continue
			}
			header, _, _ := strings.Cut(stack, "\n")
			if strings.Contains(header, "[select]") || strings.Contains(header, "[chan send]") {
				loopID = strings.Fields(header)[1]
				return true
			}
		}
		return false
	}, 5*time.Second, time.Millisecond, "accept loop did not reach the unconsumed handoff")
	require.Equal(t, sessionsBefore+2, atomic.LoadUint64(&kcp.DefaultSnmp.CurrEstab))

	require.NoError(t, l.Close())
	require.Eventually(t, func() bool {
		return !strings.Contains(kcpGoroutineDump(t), "goroutine "+loopID+" [")
	}, 5*time.Second, time.Millisecond, "accept loop remained blocked after Close")

	// kcp-go has no close frame. Its session counter decreases only when the
	// session itself is closed; closing the listener socket is not sufficient.
	// Keep the client open so this decrement must belong to the pending server session.
	require.Equal(t, sessionsBefore+1, atomic.LoadUint64(&kcp.DefaultSnmp.CurrEstab))
	require.NoError(t, peer.Close())
	require.Equal(t, sessionsBefore, atomic.LoadUint64(&kcp.DefaultSnmp.CurrEstab))
}

func TestKCPListenerAcceptUnblocksOnClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := &KCPListener{
			listener: &closeStubListener{closed: make(chan struct{}), err: errors.New("listener close failed")},
			acceptCh: make(chan net.Conn),
			done:     make(chan struct{}),
		}
		t.Cleanup(func() { _ = l.Close() })
		acceptErr := make(chan error, 1)
		go func() {
			conn, err := l.Accept()
			if conn != nil {
				t.Error("Accept returned a connection without a sender")
			}
			acceptErr <- err
		}()
		synctest.Wait() // Accept is blocked before Close.
		require.Empty(t, acceptErr)

		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() {
				if err := l.Close(); err != nil {
					t.Errorf("Close: %v", err)
				}
			})
		}
		wg.Wait()
		synctest.Wait()
		require.Len(t, acceptErr, 1)
		require.ErrorIs(t, <-acceptErr, ErrKCPListenerClosed)

		require.NoError(t, l.Close())
		conn, err := l.Accept()
		require.Nil(t, conn)
		require.ErrorIs(t, err, ErrKCPListenerClosed)
	})
}

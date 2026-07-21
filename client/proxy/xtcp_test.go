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

//go:build !frps

package proxy

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/wire"
)

func TestReadNatHoleSidUsesSelectedWireProtocol(t *testing.T) {
	for _, tc := range []struct {
		name         string
		wireProtocol string
	}{
		{name: "v2", wireProtocol: wire.ProtocolV2},
		{name: "v1", wireProtocol: wire.ProtocolV1},
		{name: "default", wireProtocol: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, server := net.Pipe()
			defer client.Close()
			defer server.Close()
			setPipeDeadline(t, client, server)

			errCh := make(chan error, 1)
			go func() {
				writer := msg.NewConn(server, msg.NewReadWriter(server, tc.wireProtocol))
				errCh <- writer.WriteMsg(&msg.NatHoleSid{Sid: "sid"})
			}()

			out, err := readNatHoleSid(client, tc.wireProtocol)
			require.NoError(t, err)
			require.Equal(t, "sid", out.Sid)
			require.NoError(t, <-errCh)
		})
	}
}

func TestRunBoundedWorkConnAppliesBackpressure(t *testing.T) {
	slots := make(chan struct{}, 2)
	started := make(chan struct{}, 3)
	dispatched := make(chan bool, 3)
	release := make(chan struct{})

	go func() {
		for range 3 {
			dispatched <- runBoundedWorkConn(context.Background(), slots, func() {
				started <- struct{}{}
				<-release
			})
		}
	}()

	// The first two handlers start immediately.
	for range 2 {
		requireRecv(t, started, "handler did not start")
		require.True(t, <-dispatched)
	}
	// The third dispatch waits until a running handler finishes.
	select {
	case <-started:
		t.Fatal("third handler started beyond the limit")
	case <-time.After(50 * time.Millisecond):
	}
	release <- struct{}{}
	requireRecv(t, started, "handler did not start after a slot was freed")
	require.True(t, <-dispatched)
	close(release)
}

func TestRunBoundedWorkConnUnlimitedByDefault(t *testing.T) {
	const total = 5
	var wg sync.WaitGroup
	wg.Add(total)
	block := make(chan struct{})
	for range total {
		require.True(t, runBoundedWorkConn(context.Background(), nil, func() {
			wg.Done()
			<-block
		}))
	}
	// All handlers run concurrently without any dispatch blocking.
	wg.Wait()
	close(block)
}

func TestRunBoundedWorkConnReturnsFalseWhenContextDone(t *testing.T) {
	slots := make(chan struct{}, 1)
	release := make(chan struct{})
	require.True(t, runBoundedWorkConn(context.Background(), slots, func() {
		<-release
	}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.False(t, runBoundedWorkConn(ctx, slots, func() {
		<-release
	}))
	close(release)
}

func requireRecv(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal(msg)
	}
}

func setPipeDeadline(t *testing.T, conns ...net.Conn) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for _, conn := range conns {
		require.NoError(t, conn.SetDeadline(deadline))
	}
}

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
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type closeStubListener struct {
	closed chan struct{}
}

func (l *closeStubListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}

func (l *closeStubListener) Close() error {
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}
	return nil
}

func (l *closeStubListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}
}

func TestWebsocketListenerAcceptUnblocksOnClose(t *testing.T) {
	wl := NewWebsocketListener(&closeStubListener{closed: make(chan struct{})})

	errCh := make(chan error, 1)
	go func() {
		_, err := wl.Accept()
		errCh <- err
	}()

	select {
	case err := <-errCh:
		t.Fatalf("Accept returned before the listener was closed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, wl.Close())
	// Closing twice must stay safe and keep reporting the closed error.
	require.NoError(t, wl.Close())

	select {
	case err := <-errCh:
		require.True(t, errors.Is(err, ErrWebsocketListenerClosed), "got %v", err)
	case <-time.After(time.Second):
		t.Fatal("Accept did not return after the listener was closed")
	}
}

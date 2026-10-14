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
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
)

type closeStubListener struct {
	closed chan struct{}
	once   sync.Once
	err    error
}

func (l *closeStubListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}

func (l *closeStubListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return l.err
}

func (l *closeStubListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}
}

func TestWebsocketListenerAcceptUnblocksOnClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wl := NewWebsocketListener(&closeStubListener{closed: make(chan struct{})})
		t.Cleanup(func() { wl.Close() })
		errCh := make(chan error, 3)
		for range 3 {
			go func() {
				c, err := wl.Accept()
				if c != nil {
					t.Error("Accept returned a connection without a sender")
				}
				errCh <- err
			}()
		}
		synctest.Wait() // All Accept calls are blocked before Close.
		require.Empty(t, errCh)

		var wg sync.WaitGroup
		for range 3 {
			wg.Go(func() {
				if err := wl.Close(); err != nil {
					t.Errorf("Close: %v", err)
				}
			})
		}
		wg.Wait()
		synctest.Wait()
		require.Len(t, errCh, 3)
		for range 3 {
			require.ErrorIs(t, <-errCh, ErrWebsocketListenerClosed)
		}
		c, err := wl.Accept()
		require.Nil(t, c)
		require.ErrorIs(t, err, ErrWebsocketListenerClosed)
	})
}

func TestWebsocketListenerCloseError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		closeErr := errors.New("listener close failed")
		wl := NewWebsocketListener(&closeStubListener{closed: make(chan struct{}), err: closeErr})
		synctest.Wait() // Let Serve register its listener before Close.
		require.ErrorIs(t, wl.Close(), closeErr)
		require.NoError(t, wl.Close())
	})
}

type websocketPipeHijacker struct {
	*httptest.ResponseRecorder
	conn net.Conn
	rw   *bufio.ReadWriter
}

func (h *websocketPipeHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return h.conn, h.rw, nil
}

func newWebsocketPipe(t *testing.T) (*WebsocketListener, *websocket.Conn, net.Conn, <-chan struct{}) {
	t.Helper()
	wl := NewWebsocketListener(&closeStubListener{closed: make(chan struct{})})
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		wl.Close()
		serverConn.Close()
		clientConn.Close()
	})
	require.NoError(t, serverConn.SetDeadline(time.Now().Add(time.Second)))
	require.NoError(t, clientConn.SetDeadline(time.Now().Add(time.Second)))
	handlerDone := make(chan struct{})
	go func() {
		defer close(handlerDone)
		rw := bufio.NewReadWriter(bufio.NewReader(serverConn), bufio.NewWriter(serverConn))
		req, err := http.ReadRequest(rw.Reader)
		if err != nil {
			t.Errorf("ReadRequest: %v", err)
			return
		}
		wl.server.Handler.ServeHTTP(&websocketPipeHijacker{httptest.NewRecorder(), serverConn, rw}, req)
	}()
	cfg, err := websocket.NewConfig("ws://localhost"+FrpWebsocketPath, "http://localhost")
	require.NoError(t, err)
	peer, err := websocket.NewClient(cfg, clientConn)
	require.NoError(t, err)
	return wl, peer, serverConn, handlerDone
}

func TestWebsocketListenerUnacceptedConnectionClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wl, peer, raw, handlerDone := newWebsocketPipe(t)
		peerErr := make(chan error, 1)
		go func() {
			_, err := peer.Read(make([]byte, 1))
			peerErr <- err
		}()
		// No Accept consumer exists. Do not Wait here: it would synchronize
		// the handler's send with Close and hide the baseline send/close race.
		require.NoError(t, wl.Close())
		require.ErrorIs(t, <-peerErr, io.EOF)
		<-handlerDone
		_, err := raw.Read(make([]byte, 1))
		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

func TestWebsocketListenerAcceptedConnectionOwnership(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wl, peer, raw, handlerDone := newWebsocketPipe(t)
		owner, err := wl.Accept()
		require.NoError(t, err)
		t.Cleanup(func() {
			raw.Close()
			owner.Close()
		})
		require.NoError(t, wl.Close())

		payload := []byte{0, 255, 128, 42}
		reply := []byte("reply")
		gotReply := make([]byte, len(reply))
		peerErr := make(chan error, 2)
		go func() {
			if _, err := peer.Write(payload); err != nil {
				peerErr <- err
				return
			}
			_, err := io.ReadFull(peer, gotReply)
			peerErr <- err
			_, err = peer.Read(make([]byte, 1))
			peerErr <- err
		}()
		got := make([]byte, len(payload))
		_, err = io.ReadFull(owner, got)
		require.NoError(t, err)
		require.Equal(t, payload, got)
		_, err = owner.Write(reply)
		require.NoError(t, err)
		require.NoError(t, <-peerErr)
		require.Equal(t, reply, gotReply)
		require.NoError(t, owner.Close())
		require.NoError(t, owner.Close())
		require.ErrorIs(t, <-peerErr, io.EOF)
		<-handlerDone
		_, err = raw.Read(make([]byte, 1))
		require.ErrorIs(t, err, io.ErrClosedPipe)
	})
}

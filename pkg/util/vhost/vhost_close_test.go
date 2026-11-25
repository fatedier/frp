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

package vhost

import (
	"context"
	"net"
	"testing"
	"time"
)

func newCloseTestMuxer() *Muxer {
	return &Muxer{
		timeout:        time.Second,
		registryRouter: NewRouters(),
		vhostFunc: func(c net.Conn) (net.Conn, map[string]string, error) {
			return c, map[string]string{"Host": "example.com"}, nil
		},
	}
}

func testPendingHandoffIsClosed(t *testing.T, client, server net.Conn) {
	t.Helper()

	m := newCloseTestMuxer()
	entered := make(chan struct{})
	release := make(chan struct{})
	m.SetSuccessHookFunc(func(net.Conn, map[string]string) error {
		close(entered)
		<-release
		return nil
	})
	l, err := m.Listen(context.Background(), &RouteConfig{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}

	handled := make(chan struct{})
	go func() {
		defer close(handled)
		m.handle(server)
	}()
	<-entered
	if err := client.SetReadDeadline(time.Now().Add(300 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}

	// Close the route after resolution but before the handoff. The pending
	// connection must be closed, not abandoned.
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	close(release)
	<-handled

	if _, err := client.Read(make([]byte, 1)); err == nil {
		t.Fatal("connection was not closed by the muxer")
	} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
		t.Fatalf("connection was not closed by the muxer: %v", err)
	}
}

func TestListenerCloseReleasesPendingConn(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	testPendingHandoffIsClosed(t, client, server)
}

func TestListenerCloseReleasesPendingTCPConn(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	server, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	testPendingHandoffIsClosed(t, client, server)
}

func TestListenerCloseIsIdempotent(t *testing.T) {
	m := newCloseTestMuxer()
	l, err := m.Listen(context.Background(), &RouteConfig{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Accept(); err == nil {
		t.Fatal("Accept after Close should return an error")
	}
}

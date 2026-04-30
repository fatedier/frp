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

package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestHTTPProxyHandleFragmentedConnectMethod(t *testing.T) {
	require := require.New(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(err)
	defer ln.Close()

	const payload = "ping"
	echoErr := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			echoErr <- err
			return
		}
		defer conn.Close()

		buf := make([]byte, len(payload))
		if _, err = io.ReadFull(conn, buf); err != nil {
			echoErr <- err
			return
		}
		if string(buf) != payload {
			echoErr <- fmt.Errorf("unexpected payload %q", string(buf))
			return
		}
		_, err = conn.Write([]byte("echo:" + payload))
		echoErr <- err
	}()

	hp := &HTTPProxy{
		opts: &v1.HTTPProxyPluginOptions{},
		l:    NewProxyListener(),
	}

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	go hp.Handle(context.Background(), &ConnectionInfo{
		Conn:           serverConn,
		UnderlyingConn: serverConn,
	})

	require.NoError(clientConn.SetDeadline(time.Now().Add(5 * time.Second)))

	targetAddr := ln.Addr().String()
	req := "CONNECT " + targetAddr + " HTTP/1.1\r\nHost: " + targetAddr + "\r\n\r\n"
	_, err = clientConn.Write([]byte("CON"))
	require.NoError(err)
	_, err = clientConn.Write([]byte(req[len("CON"):]))
	require.NoError(err)

	rd := bufio.NewReader(clientConn)
	status, err := rd.ReadString('\n')
	require.NoError(err)
	require.Equal("HTTP/1.1 200 OK\r\n", status)
	line, err := rd.ReadString('\n')
	require.NoError(err)
	require.Equal("\r\n", line)

	_, err = clientConn.Write([]byte(payload))
	require.NoError(err)

	got := make([]byte, len("echo:"+payload))
	_, err = io.ReadFull(rd, got)
	require.NoError(err)
	require.Equal("echo:"+payload, string(got))

	select {
	case err := <-echoErr:
		require.NoError(err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for echo server")
	}
}

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
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestHTTPPluginsCloseRejectedConnection(t *testing.T) {
	type handlerCloser interface {
		Handle(context.Context, *ConnectionInfo)
		Close() error
	}
	tests := []struct {
		name       string
		newPlugin  func() (handlerCloser, error)
		firstBytes string
	}{
		{
			name: "static file",
			newPlugin: func() (handlerCloser, error) {
				return NewStaticFilePlugin(PluginContext{}, &v1.StaticFilePluginOptions{LocalPath: t.TempDir()})
			},
		},
		{
			name: "HTTP bridge",
			newPlugin: func() (handlerCloser, error) {
				return newHTTPBridgePluginServer(http.NotFoundHandler(), false), nil
			},
		},
		{
			name: "HTTP proxy",
			newPlugin: func() (handlerCloser, error) {
				return NewHTTPProxyPlugin(PluginContext{}, &v1.HTTPProxyPluginOptions{})
			},
			// HTTPProxy reads seven bytes to distinguish CONNECT before handoff.
			firstBytes: "GET / H",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := tt.newPlugin()
			require.NoError(t, err)
			require.NoError(t, p.Close())

			conn, peer := net.Pipe()
			defer conn.Close()
			defer peer.Close()
			require.NoError(t, peer.SetReadDeadline(time.Now().Add(time.Second)))
			handled := make(chan struct{})
			go func() {
				p.Handle(context.Background(), &ConnectionInfo{Conn: conn, UnderlyingConn: conn})
				close(handled)
			}()
			if tt.firstBytes != "" {
				require.NoError(t, peer.SetWriteDeadline(time.Now().Add(time.Second)))
				_, err := io.WriteString(peer, tt.firstBytes)
				require.NoError(t, err)
			}
			select {
			case <-handled:
			case <-time.After(time.Second):
				t.Fatal("Handle did not return after the listener rejected the connection")
			}

			_, err = peer.Read(make([]byte, 1))
			require.ErrorIs(t, err, io.EOF, "a rejected connection must be closed before Handle returns")
		})
	}
}

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

package client

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

// A dedicated session is opened before any visitor connection needs it. Over websocket (frps behind a reverse
// proxy) frps sees nothing inside the websocket until the client speaks, and it closes a connection whose first
// byte does not arrive within its read timeout. The session must therefore put bytes on the wire as soon as it opens.
func TestDedicatedVisitorSessionSpeaksFirst(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	wl := netpkg.NewWebsocketListener(ln)
	defer wl.Close()

	cfg := &v1.ClientCommonConfig{}
	cfg.ServerAddr = "127.0.0.1"
	cfg.ServerPort = ln.Addr().(*net.TCPAddr).Port
	cfg.Transport.Protocol = "websocket"
	cfg.Transport.TLS.Enable = lo.ToPtr(false)
	cfg.Transport.VisitorSession = v1.VisitorSessionDedicated
	require.NoError(t, cfg.Complete())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sessions := newVisitorSessions(ctx, cfg)
	defer sessions.CloseAll()
	_ = sessions.ConnectServerFn("visitor")

	conn, err := wl.Accept()
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	require.NoError(t, err, "dedicated session stayed silent after opening")
	require.Positive(t, n)
}

func TestDedicatedVisitorSessionsUseQUICPort(t *testing.T) {
	cfg := &v1.ClientCommonConfig{}
	cfg.ServerAddr = "127.0.0.1"
	cfg.ServerPort = 7000
	cfg.Transport.Protocol = "wss"
	cfg.Transport.VisitorSession = v1.VisitorSessionDedicated
	cfg.Transport.VisitorSessionQUICPort = 7001
	require.NoError(t, cfg.Complete())

	sessions := newVisitorSessions(context.Background(), cfg)
	require.Equal(t, "quic", sessions.cfg.Transport.Protocol)
	require.Equal(t, 7001, sessions.cfg.ServerPort)
	// the control connection's configuration is left alone
	require.Equal(t, "wss", cfg.Transport.Protocol)
	require.Equal(t, 7000, cfg.ServerPort)
}

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
	"net"
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

func setPipeDeadline(t *testing.T, conns ...net.Conn) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for _, conn := range conns {
		require.NoError(t, conn.SetDeadline(deadline))
	}
}

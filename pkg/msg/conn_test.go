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

package msg

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/proto/wire"
)

func TestConnReadWriteMsg(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
	}{
		{name: "v1", protocol: wire.ProtocolV1},
		{name: "v2", protocol: wire.ProtocolV2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := net.Pipe()
			defer client.Close()
			defer server.Close()

			clientConn := NewConn(client, NewReadWriter(client, tt.protocol))
			serverConn := NewConn(server, NewReadWriter(server, tt.protocol))

			in := &Ping{PrivilegeKey: "key", Timestamp: 123}
			errCh := make(chan error, 1)
			go func() {
				errCh <- clientConn.WriteMsg(in)
			}()

			out, err := serverConn.ReadMsg()
			require.NoError(t, err)
			require.Equal(t, in, out)
			require.NoError(t, <-errCh)
		})
	}
}

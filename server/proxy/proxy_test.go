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

package proxy

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/msg"
)

func TestWorkConnStartWritesStartWorkConn(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	serverMsgConn := msg.NewConn(server, msg.NewV2ReadWriter(server))
	clientMsgConn := msg.NewConn(client, msg.NewV2ReadWriter(client))
	workConn := NewWorkConn(serverMsgConn)

	in := &msg.StartWorkConn{ProxyName: "tcp", SrcAddr: "127.0.0.1", SrcPort: 1234}
	type startResult struct {
		conn net.Conn
		err  error
	}
	resultCh := make(chan startResult, 1)
	go func() {
		conn, err := workConn.Start(in)
		resultCh <- startResult{conn: conn, err: err}
	}()

	out, err := clientMsgConn.ReadMsg()
	require.NoError(t, err)
	require.Equal(t, in, out)

	result := <-resultCh
	require.NoError(t, result.err)
	require.Same(t, serverMsgConn, result.conn)
}

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
	stdnet "net"
	"testing"

	"github.com/stretchr/testify/require"
)

// The godebug multipathtcp=0 line in go.mod keeps Go from creating MPTCP
// listeners by default, which some vendor kernels fail to accept on.
func TestMultipathTCPDisabledByDefault(t *testing.T) {
	require.False(t, (&stdnet.ListenConfig{}).MultipathTCP())
	require.False(t, (&stdnet.Dialer{}).MultipathTCP())

	ln, err := stdnet.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	// Ask for MPTCP on the client side: it is only negotiated if the
	// listener is an MPTCP socket.
	var d stdnet.Dialer
	d.SetMultipathTCP(true)
	conn, err := d.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer conn.Close()

	// Check the accepted side: on kernels older than 5.16 the client side
	// reports MPTCP for any MPTCP socket, even after a fallback to TCP.
	serverConn, err := ln.Accept()
	require.NoError(t, err)
	defer serverConn.Close()

	isMPTCP, err := serverConn.(*stdnet.TCPConn).MultipathTCP()
	require.NoError(t, err)
	require.False(t, isMPTCP, "listener created by net.Listen uses MPTCP")
}

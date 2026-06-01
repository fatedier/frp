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

package visitor

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/proto/wire"
	"github.com/fatedier/frp/pkg/util/util"
)

func TestManagerNewConnCarriesWireProtocol(t *testing.T) {
	vm := NewManager()
	listener, err := vm.Listen("sudp", "secret", []string{"*"})
	require.NoError(t, err)
	defer listener.Close()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	now := time.Now().Unix()
	errCh := make(chan error, 1)
	go func() {
		errCh <- vm.NewConn(
			"sudp",
			server,
			now,
			util.GetAuthKey("secret", now),
			false,
			false,
			"user",
			wire.ProtocolV2,
		)
	}()

	acceptedConn, err := listener.Accept()
	require.NoError(t, err)
	defer acceptedConn.Close()

	getter, ok := acceptedConn.(interface{ WireProtocol() string })
	require.True(t, ok)
	require.Equal(t, wire.ProtocolV2, getter.WireProtocol())
	require.NoError(t, <-errCh)
}

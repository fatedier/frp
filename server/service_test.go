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

package server

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriteWithDeadlineTimesOutAndClearsDeadline(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	err := writeWithDeadline(serverConn, 50*time.Millisecond, func() error {
		_, writeErr := serverConn.Write([]byte("x"))
		return writeErr
	})
	require.Error(t, err)

	var netErr net.Error
	require.True(t, errors.As(err, &netErr))
	require.True(t, netErr.Timeout())

	readCh := make(chan byte, 1)
	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 1)
		if _, readErr := clientConn.Read(buf); readErr != nil {
			errCh <- readErr
			return
		}
		readCh <- buf[0]
	}()

	_, err = serverConn.Write([]byte("y"))
	require.NoError(t, err)

	select {
	case b := <-readCh:
		require.Equal(t, byte('y'), b)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for write after deadline reset")
	}
}

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
	"bytes"
	"io"
	stdnet "net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/proto/wire"
)

func TestNewAEADCryptoReadWriterRoundTrip(t *testing.T) {
	clientConn, serverConn := stdnet.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	key := []byte("token")
	transcriptHash := bytes.Repeat([]byte{0x11}, 32)
	clientRW, err := NewAEADCryptoReadWriter(
		clientConn,
		key,
		AEADCryptoRoleClient,
		wire.AEADAlgorithmXChaCha20Poly1305,
		transcriptHash,
	)
	require.NoError(t, err)
	serverRW, err := NewAEADCryptoReadWriter(
		serverConn,
		key,
		AEADCryptoRoleServer,
		wire.AEADAlgorithmXChaCha20Poly1305,
		transcriptHash,
	)
	require.NoError(t, err)

	clientErrCh := make(chan error, 1)
	go func() {
		if _, err := clientRW.Write([]byte("ping")); err != nil {
			clientErrCh <- err
			return
		}
		buf := make([]byte, len("pong"))
		_, err := io.ReadFull(clientRW, buf)
		clientErrCh <- err
	}()

	buf := make([]byte, len("ping"))
	_, err = io.ReadFull(serverRW, buf)
	require.NoError(t, err)
	require.Equal(t, "ping", string(buf))
	_, err = serverRW.Write([]byte("pong"))
	require.NoError(t, err)
	require.NoError(t, <-clientErrCh)
}

func TestNewAEADCryptoReadWriterRejectsDifferentTranscript(t *testing.T) {
	clientConn, serverConn := stdnet.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	require.NoError(t, clientConn.SetDeadline(time.Now().Add(time.Second)))
	require.NoError(t, serverConn.SetDeadline(time.Now().Add(time.Second)))

	key := []byte("token")
	clientRW, err := NewAEADCryptoReadWriter(
		clientConn,
		key,
		AEADCryptoRoleClient,
		wire.AEADAlgorithmAES256GCM,
		bytes.Repeat([]byte{0x22}, 32),
	)
	require.NoError(t, err)
	serverRW, err := NewAEADCryptoReadWriter(
		serverConn,
		key,
		AEADCryptoRoleServer,
		wire.AEADAlgorithmAES256GCM,
		bytes.Repeat([]byte{0x33}, 32),
	)
	require.NoError(t, err)

	writeErrCh := make(chan error, 1)
	go func() {
		_, err := clientRW.Write([]byte("ping"))
		writeErrCh <- err
	}()

	buf := make([]byte, len("ping"))
	_, err = io.ReadFull(serverRW, buf)
	require.Error(t, err)
	require.NoError(t, <-writeErrCh)
}

func TestDeriveAEADControlKeysUsesDistinctDirections(t *testing.T) {
	clientToServerKey, serverToClientKey, err := deriveAEADControlKeys(
		[]byte("token"),
		wire.AEADAlgorithmXChaCha20Poly1305,
		bytes.Repeat([]byte{0x44}, 32),
	)
	require.NoError(t, err)
	require.NotEqual(t, clientToServerKey, serverToClientKey)
}

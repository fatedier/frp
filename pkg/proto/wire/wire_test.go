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

package wire

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	conn := NewConn(&buf)

	in := mustClientHello(t, BootstrapInfo{
		Transport: "tcp",
		TLS:       true,
		TCPMux:    true,
	})
	require.NoError(t, conn.WriteJSONFrame(FrameTypeClientHello, in))

	var out ClientHello
	require.NoError(t, conn.ReadJSONFrame(FrameTypeClientHello, &out))
	require.Equal(t, in, out)
}

func TestReadFrameRejectsUnsupportedFlags(t *testing.T) {
	var buf bytes.Buffer
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], FrameTypeMessage)
	binary.BigEndian.PutUint16(header[2:4], 1)
	binary.BigEndian.PutUint32(header[4:8], 0)
	buf.Write(header)

	_, err := NewConn(&buf).ReadFrame()
	require.ErrorContains(t, err, "unsupported frame flags")
}

func TestReadFrameRejectsOversizedPayload(t *testing.T) {
	var buf bytes.Buffer
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], FrameTypeMessage)
	binary.BigEndian.PutUint32(header[4:8], DefaultMaxFramePayloadSize+1)
	buf.Write(header)

	_, err := NewConn(&buf).ReadFrame()
	require.ErrorContains(t, err, "exceeds limit")
}

func TestCheckMagicV2ConsumesMagic(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()

	want := []byte("payload")
	go func() {
		defer client.Close()
		_, _ = client.Write(append([]byte(MagicV2), want...))
	}()

	out, isV2, err := CheckMagic(server)
	require.NoError(t, err)
	require.True(t, isV2)

	got := make([]byte, len(want))
	_, err = io.ReadFull(out, got)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestWriteMagicIfV2(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, WriteMagicIfV2(&buf, ProtocolV1))
	require.Empty(t, buf.Bytes())

	require.NoError(t, WriteMagicIfV2(&buf, ProtocolV2))
	require.Equal(t, []byte(MagicV2), buf.Bytes())
}

func TestCheckMagicV1PreservesReadBytes(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()

	want := []byte("legacy payload")
	go func() {
		defer client.Close()
		_, _ = client.Write(want)
	}()

	out, isV2, err := CheckMagic(server)
	require.NoError(t, err)
	require.False(t, isV2)

	got, err := io.ReadAll(out)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestValidateClientHello(t *testing.T) {
	hello := mustClientHello(t, BootstrapInfo{})
	require.NoError(t, ValidateClientHello(hello))
	require.Len(t, hello.Capabilities.Crypto.ClientRandom, CryptoRandomSize)
	require.ElementsMatch(t, []string{
		AEADAlgorithmAES256GCM,
		AEADAlgorithmXChaCha20Poly1305,
	}, hello.Capabilities.Crypto.Algorithms)

	hello.Capabilities.Message.Codecs = []string{"unknown"}
	require.ErrorContains(t, ValidateClientHello(hello), "unsupported message codec")
}

func TestValidateClientHelloRejectsInvalidCrypto(t *testing.T) {
	hello := mustClientHello(t, BootstrapInfo{})
	hello.Capabilities.Crypto.ClientRandom = hello.Capabilities.Crypto.ClientRandom[:CryptoRandomSize-1]
	require.ErrorContains(t, ValidateClientHello(hello), "invalid crypto client random length")

	hello = mustClientHello(t, BootstrapInfo{})
	hello.Capabilities.Crypto.Algorithms = []string{"unknown"}
	require.ErrorContains(t, ValidateClientHello(hello), "no supported crypto algorithm")
}

func TestPreferredAEADAlgorithms(t *testing.T) {
	require.ElementsMatch(t, []string{
		AEADAlgorithmAES256GCM,
		AEADAlgorithmXChaCha20Poly1305,
	}, PreferredAEADAlgorithms())
}

func TestNewServerHelloSelectsFirstSupportedAEADAlgorithm(t *testing.T) {
	hello := mustClientHello(t, BootstrapInfo{})
	hello.Capabilities.Crypto.Algorithms = []string{"future-aead", AEADAlgorithmXChaCha20Poly1305, AEADAlgorithmAES256GCM}

	serverHello, err := NewServerHello(hello)
	require.NoError(t, err)
	require.Equal(t, MessageCodecJSON, serverHello.Selected.Message.Codec)
	require.Equal(t, AEADAlgorithmXChaCha20Poly1305, serverHello.Selected.Crypto.Algorithm)
	require.Len(t, serverHello.Selected.Crypto.ServerRandom, CryptoRandomSize)
}

func TestNewClientCryptoContextValidatesServerHello(t *testing.T) {
	hello := mustClientHello(t, BootstrapInfo{})
	serverHello, err := NewServerHello(hello)
	require.NoError(t, err)
	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)

	ctx, err := NewClientCryptoContext(clientHelloPayload, serverHelloPayload)
	require.NoError(t, err)
	require.Equal(t, serverHello.Selected.Crypto.Algorithm, ctx.Algorithm)
	require.Len(t, ctx.TranscriptHash, 32)

	tampered := serverHello
	tampered.Selected.Crypto.ServerRandom = append([]byte(nil), serverHello.Selected.Crypto.ServerRandom...)
	tampered.Selected.Crypto.ServerRandom[0] ^= 0xff
	_, tamperedServerHelloPayload := mustCryptoTranscriptPayloads(t, hello, tampered)
	tamperedCtx, err := NewClientCryptoContext(clientHelloPayload, tamperedServerHelloPayload)
	require.NoError(t, err)
	require.NotEqual(t, ctx.TranscriptHash, tamperedCtx.TranscriptHash)
}

func TestNewCryptoContextBindsFullClientHelloPayload(t *testing.T) {
	hello := mustClientHello(t, BootstrapInfo{
		Transport: "tcp",
		TLS:       true,
		TCPMux:    true,
	})
	serverHello, err := NewServerHello(hello)
	require.NoError(t, err)
	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)

	ctx := NewCryptoContext(serverHello.Selected.Crypto.Algorithm, clientHelloPayload, serverHelloPayload)

	tamperedHello := hello
	tamperedHello.Bootstrap.TLS = false
	tamperedClientHelloPayload, _ := mustCryptoTranscriptPayloads(t, tamperedHello, serverHello)
	tamperedCtx := NewCryptoContext(serverHello.Selected.Crypto.Algorithm, tamperedClientHelloPayload, serverHelloPayload)
	require.NotEqual(t, ctx.TranscriptHash, tamperedCtx.TranscriptHash)
}

func TestNewClientCryptoContextRejectsUnknownServerSelection(t *testing.T) {
	hello := mustClientHello(t, BootstrapInfo{})
	serverHello, err := NewServerHello(hello)
	require.NoError(t, err)

	serverHello.Selected.Crypto.Algorithm = "unknown"
	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)
	_, err = NewClientCryptoContext(clientHelloPayload, serverHelloPayload)
	require.ErrorContains(t, err, "unknown selected crypto algorithm")
}

func TestNewClientCryptoContextRejectsUnadvertisedServerSelection(t *testing.T) {
	hello := mustClientHello(t, BootstrapInfo{})
	hello.Capabilities.Crypto.Algorithms = []string{AEADAlgorithmAES256GCM}
	serverHello, err := NewServerHello(hello)
	require.NoError(t, err)

	serverHello.Selected.Crypto.Algorithm = AEADAlgorithmXChaCha20Poly1305
	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)
	_, err = NewClientCryptoContext(clientHelloPayload, serverHelloPayload)
	require.ErrorContains(t, err, "selected crypto algorithm was not advertised by client")
}

func mustClientHello(t *testing.T, bootstrap BootstrapInfo) ClientHello {
	t.Helper()

	hello, err := NewClientHello(bootstrap)
	require.NoError(t, err)
	return hello
}

func mustCryptoTranscriptPayloads(t *testing.T, hello ClientHello, serverHello ServerHello) ([]byte, []byte) {
	t.Helper()

	clientHelloFrame, err := NewJSONFrame(FrameTypeClientHello, hello)
	require.NoError(t, err)
	serverHelloFrame, err := NewJSONFrame(FrameTypeServerHello, serverHello)
	require.NoError(t, err)
	return clientHelloFrame.Payload, serverHelloFrame.Payload
}

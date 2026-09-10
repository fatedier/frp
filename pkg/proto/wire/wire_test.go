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
	"crypto/ecdh"
	"encoding/binary"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	conn := NewConn(&buf)

	in, _ := mustClientHello(t, BootstrapInfo{
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
	hello, _ := mustClientHello(t, BootstrapInfo{})
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
	hello, _ := mustClientHello(t, BootstrapInfo{})
	hello.Capabilities.Crypto.ClientRandom = hello.Capabilities.Crypto.ClientRandom[:CryptoRandomSize-1]
	require.ErrorContains(t, ValidateClientHello(hello), "invalid crypto client random length")

	hello, _ = mustClientHello(t, BootstrapInfo{})
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
	hello, _ := mustClientHello(t, BootstrapInfo{})
	hello.Capabilities.Crypto.Algorithms = []string{"future-aead", AEADAlgorithmXChaCha20Poly1305, AEADAlgorithmAES256GCM}

	serverHello, _, err := NewServerHello(hello)
	require.NoError(t, err)
	require.Equal(t, MessageCodecJSON, serverHello.Selected.Message.Codec)
	require.Equal(t, UDPPacketCodecBinary, serverHello.Selected.Message.UDPPacketCodec)
	require.Equal(t, AEADAlgorithmXChaCha20Poly1305, serverHello.Selected.Crypto.Algorithm)
	require.Len(t, serverHello.Selected.Crypto.ServerRandom, CryptoRandomSize)
}

func TestUDPPacketCodecNegotiationFallbackAndValidation(t *testing.T) {
	hello, _ := mustClientHello(t, BootstrapInfo{})
	serverHello, _, err := NewServerHello(hello)
	require.NoError(t, err)
	require.Equal(t, UDPPacketCodecBinary, serverHello.Selected.Message.UDPPacketCodec)
	require.NoError(t, ValidateServerHelloForClient(hello, serverHello))

	legacyHello := hello
	legacyHello.Capabilities.Message.UDPPacketCodecs = nil
	legacyServerHello, _, err := NewServerHello(legacyHello)
	require.NoError(t, err)
	require.Empty(t, legacyServerHello.Selected.Message.UDPPacketCodec)
	require.NoError(t, ValidateServerHelloForClient(legacyHello, legacyServerHello))

	unknownOffer := hello
	unknownOffer.Capabilities.Message.UDPPacketCodecs = []string{"unknown"}
	unknownServerHello, _, err := NewServerHello(unknownOffer)
	require.NoError(t, err)
	require.Empty(t, unknownServerHello.Selected.Message.UDPPacketCodec)

	rejected := serverHello
	rejected.Selected.Message.UDPPacketCodec = "unknown"
	require.ErrorContains(t, ValidateServerHelloForClient(hello, rejected), "unsupported selected UDP packet codec")

	unadvertised := serverHello
	unadvertised.Selected.Message.UDPPacketCodec = UDPPacketCodecBinary
	require.ErrorContains(t, ValidateServerHelloForClient(legacyHello, unadvertised), "was not advertised")
}

func TestNewClientCryptoContextValidatesServerHello(t *testing.T) {
	hello, clientPriv := mustClientHello(t, BootstrapInfo{})
	serverHello, _, err := NewServerHello(hello)
	require.NoError(t, err)
	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)

	ctx, err := NewClientCryptoContext(clientPriv, clientHelloPayload, serverHelloPayload)
	require.NoError(t, err)
	require.Equal(t, serverHello.Selected.Crypto.Algorithm, ctx.Algorithm)
	require.Len(t, ctx.TranscriptHash, 32)

	tampered := serverHello
	tampered.Selected.Crypto.ServerRandom = append([]byte(nil), serverHello.Selected.Crypto.ServerRandom...)
	tampered.Selected.Crypto.ServerRandom[0] ^= 0xff
	_, tamperedServerHelloPayload := mustCryptoTranscriptPayloads(t, hello, tampered)
	tamperedCtx, err := NewClientCryptoContext(clientPriv, clientHelloPayload, tamperedServerHelloPayload)
	require.NoError(t, err)
	require.NotEqual(t, ctx.TranscriptHash, tamperedCtx.TranscriptHash)
}

func TestNewCryptoContextBindsFullClientHelloPayload(t *testing.T) {
	hello, _ := mustClientHello(t, BootstrapInfo{
		Transport: "tcp",
		TLS:       true,
		TCPMux:    true,
	})
	serverHello, _, err := NewServerHello(hello)
	require.NoError(t, err)
	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)

	ctx := NewCryptoContext(serverHello.Selected.Crypto.Algorithm, nil, clientHelloPayload, serverHelloPayload)

	tamperedHello := hello
	tamperedHello.Bootstrap.TLS = false
	tamperedClientHelloPayload, _ := mustCryptoTranscriptPayloads(t, tamperedHello, serverHello)
	tamperedCtx := NewCryptoContext(serverHello.Selected.Crypto.Algorithm, nil, tamperedClientHelloPayload, serverHelloPayload)
	require.NotEqual(t, ctx.TranscriptHash, tamperedCtx.TranscriptHash)
}

func TestNewClientCryptoContextRejectsUnknownServerSelection(t *testing.T) {
	hello, clientPriv := mustClientHello(t, BootstrapInfo{})
	serverHello, _, err := NewServerHello(hello)
	require.NoError(t, err)

	serverHello.Selected.Crypto.Algorithm = "unknown"
	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)
	_, err = NewClientCryptoContext(clientPriv, clientHelloPayload, serverHelloPayload)
	require.ErrorContains(t, err, "unknown selected crypto algorithm")
}

func TestNewClientCryptoContextRejectsUnadvertisedServerSelection(t *testing.T) {
	hello, clientPriv := mustClientHello(t, BootstrapInfo{})
	hello.Capabilities.Crypto.Algorithms = []string{AEADAlgorithmAES256GCM}
	serverHello, _, err := NewServerHello(hello)
	require.NoError(t, err)

	serverHello.Selected.Crypto.Algorithm = AEADAlgorithmXChaCha20Poly1305
	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)
	_, err = NewClientCryptoContext(clientPriv, clientHelloPayload, serverHelloPayload)
	require.ErrorContains(t, err, "selected crypto algorithm was not advertised by client")
}

func mustClientHello(t *testing.T, bootstrap BootstrapInfo) (ClientHello, *ecdh.PrivateKey) {
	t.Helper()

	hello, priv, err := NewClientHello(bootstrap)
	require.NoError(t, err)
	return hello, priv
}

func mustCryptoTranscriptPayloads(t *testing.T, hello ClientHello, serverHello ServerHello) ([]byte, []byte) {
	t.Helper()

	clientHelloFrame, err := NewJSONFrame(FrameTypeClientHello, hello)
	require.NoError(t, err)
	serverHelloFrame, err := NewJSONFrame(FrameTypeServerHello, serverHello)
	require.NoError(t, err)
	return clientHelloFrame.Payload, serverHelloFrame.Payload
}

func TestKeyAgreementProducesSharedSecretWithoutAuthKey(t *testing.T) {
	hello, clientPriv := mustClientHello(t, BootstrapInfo{})
	require.NotEmpty(t, hello.Capabilities.Crypto.ClientKeyShare)

	serverHello, serverSecret, err := NewServerHello(hello)
	require.NoError(t, err)
	require.NotEmpty(t, serverHello.Selected.Crypto.ServerKeyShare)
	require.NotEmpty(t, serverSecret)

	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)
	clientCtx, err := NewClientCryptoContext(clientPriv, clientHelloPayload, serverHelloPayload)
	require.NoError(t, err)
	require.Equal(t, serverSecret, clientCtx.SharedSecret)

	serverCtx := NewCryptoContext(serverHello.Selected.Crypto.Algorithm, serverSecret, clientHelloPayload, serverHelloPayload)

	require.NotEmpty(t, clientCtx.DeriveIKM(nil))
	require.Equal(t, serverCtx.DeriveIKM(nil), clientCtx.DeriveIKM(nil))
}

func TestKeyAgreementIsSessionUnique(t *testing.T) {
	first, firstPriv := mustClientHello(t, BootstrapInfo{})
	firstServerHello, firstSecret, err := NewServerHello(first)
	require.NoError(t, err)
	firstClientPayload, firstServerPayload := mustCryptoTranscriptPayloads(t, first, firstServerHello)
	firstCtx, err := NewClientCryptoContext(firstPriv, firstClientPayload, firstServerPayload)
	require.NoError(t, err)

	second, secondPriv := mustClientHello(t, BootstrapInfo{})
	secondServerHello, _, err := NewServerHello(second)
	require.NoError(t, err)
	secondClientPayload, secondServerPayload := mustCryptoTranscriptPayloads(t, second, secondServerHello)
	secondCtx, err := NewClientCryptoContext(secondPriv, secondClientPayload, secondServerPayload)
	require.NoError(t, err)

	require.NotEqual(t, firstSecret, secondCtx.SharedSecret)
	require.NotEqual(t, firstCtx.DeriveIKM(nil), secondCtx.DeriveIKM(nil))
}

func TestKeyAgreementBackwardCompatibleWithPeerWithoutKeyShare(t *testing.T) {
	hello, _ := mustClientHello(t, BootstrapInfo{})
	hello.Capabilities.Crypto.ClientKeyShare = nil

	serverHello, secret, err := NewServerHello(hello)
	require.NoError(t, err)
	require.Empty(t, serverHello.Selected.Crypto.ServerKeyShare)
	require.Empty(t, secret)

	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)
	ctx, err := NewClientCryptoContext(nil, clientHelloPayload, serverHelloPayload)
	require.NoError(t, err)
	require.Empty(t, ctx.SharedSecret)
	require.Equal(t, []byte("token"), ctx.DeriveIKM([]byte("token")))
}

func TestClientRejectsUnadvertisedServerKeyShare(t *testing.T) {
	hello, _ := mustClientHello(t, BootstrapInfo{})
	hello.Capabilities.Crypto.ClientKeyShare = nil

	serverHello, _, err := NewServerHello(hello)
	require.NoError(t, err)
	serverHello.Selected.Crypto.ServerKeyShare = make([]byte, 32)

	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)
	_, err = NewClientCryptoContext(nil, clientHelloPayload, serverHelloPayload)
	require.ErrorContains(t, err, "not advertised by client")
}

func TestKeyAgreementStrippedClientKeyShareLeavesNoKeyingMaterial(t *testing.T) {
	hello, clientPriv := mustClientHello(t, BootstrapInfo{})
	stripped := hello
	stripped.Capabilities.Crypto.ClientKeyShare = nil

	serverHello, serverSecret, err := NewServerHello(stripped)
	require.NoError(t, err)
	require.Empty(t, serverSecret)

	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)
	ctx, err := NewClientCryptoContext(clientPriv, clientHelloPayload, serverHelloPayload)
	require.NoError(t, err)
	require.Empty(t, ctx.SharedSecret)
	require.Empty(t, ctx.DeriveIKM(nil))
}

func TestKeyAgreementRejectsMalformedKeyShares(t *testing.T) {
	for _, size := range []int{1, 31, 33} {
		hello, _ := mustClientHello(t, BootstrapInfo{})
		hello.Capabilities.Crypto.ClientKeyShare = make([]byte, size)
		_, _, err := NewServerHello(hello)
		require.ErrorContains(t, err, "invalid client key share")
	}

	hello, clientPriv := mustClientHello(t, BootstrapInfo{})
	serverHello, _, err := NewServerHello(hello)
	require.NoError(t, err)
	serverHello.Selected.Crypto.ServerKeyShare = make([]byte, 31)
	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)
	_, err = NewClientCryptoContext(clientPriv, clientHelloPayload, serverHelloPayload)
	require.ErrorContains(t, err, "invalid server key share")
}

func TestKeyAgreementRejectsLowOrderPoint(t *testing.T) {
	hello, _ := mustClientHello(t, BootstrapInfo{})
	hello.Capabilities.Crypto.ClientKeyShare = make([]byte, 32)
	_, _, err := NewServerHello(hello)
	require.Error(t, err)

	hello, clientPriv := mustClientHello(t, BootstrapInfo{})
	serverHello, _, err := NewServerHello(hello)
	require.NoError(t, err)
	serverHello.Selected.Crypto.ServerKeyShare = make([]byte, 32)
	clientHelloPayload, serverHelloPayload := mustCryptoTranscriptPayloads(t, hello, serverHello)
	_, err = NewClientCryptoContext(clientPriv, clientHelloPayload, serverHelloPayload)
	require.ErrorContains(t, err, "X25519 exchange")
}

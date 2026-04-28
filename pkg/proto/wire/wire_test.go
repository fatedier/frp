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

	in := DefaultClientHello(BootstrapInfo{
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
	require.NoError(t, ValidateClientHello(DefaultClientHello(BootstrapInfo{})))

	hello := DefaultClientHello(BootstrapInfo{})
	hello.Capabilities.Message.Codecs = []string{"unknown"}
	require.ErrorContains(t, ValidateClientHello(hello), "unsupported message codec")
}

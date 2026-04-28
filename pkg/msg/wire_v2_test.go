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
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/proto/wire"
)

func TestV2ReadWriterRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	rw := NewV2ReadWriter(&buf)

	in := &Login{
		Version: "test-version",
		RunID:   "run-id",
		User:    "user",
	}
	require.NoError(t, rw.WriteMsg(in))

	out, err := rw.ReadMsg()
	require.NoError(t, err)
	require.Equal(t, in, out)
}

func TestNewReadWriter(t *testing.T) {
	require.IsType(t, &V1ReadWriter{}, NewReadWriter(&bytes.Buffer{}, ""))
	require.IsType(t, &V1ReadWriter{}, NewReadWriter(&bytes.Buffer{}, wire.ProtocolV1))
	require.IsType(t, &V2ReadWriter{}, NewReadWriter(&bytes.Buffer{}, wire.ProtocolV2))
}

func TestV2MessageTypeIDsAreStable(t *testing.T) {
	require.Equal(t, uint16(1), V2TypeLogin)
	require.Equal(t, uint16(2), V2TypeLoginResp)
	require.Equal(t, uint16(3), V2TypeNewProxy)
	require.Equal(t, uint16(4), V2TypeNewProxyResp)
	require.Equal(t, uint16(5), V2TypeCloseProxy)
	require.Equal(t, uint16(6), V2TypeNewWorkConn)
	require.Equal(t, uint16(7), V2TypeReqWorkConn)
	require.Equal(t, uint16(8), V2TypeStartWorkConn)
	require.Equal(t, uint16(9), V2TypeNewVisitorConn)
	require.Equal(t, uint16(10), V2TypeNewVisitorConnResp)
	require.Equal(t, uint16(11), V2TypePing)
	require.Equal(t, uint16(12), V2TypePong)
	require.Equal(t, uint16(13), V2TypeUDPPacket)
	require.Equal(t, uint16(14), V2TypeNatHoleVisitor)
	require.Equal(t, uint16(15), V2TypeNatHoleClient)
	require.Equal(t, uint16(16), V2TypeNatHoleResp)
	require.Equal(t, uint16(17), V2TypeNatHoleSid)
	require.Equal(t, uint16(18), V2TypeNatHoleReport)
}

func TestV2MessageFrameEncoding(t *testing.T) {
	frame, err := EncodeV2MessageFrame(&ReqWorkConn{})
	require.NoError(t, err)
	require.Equal(t, wire.FrameTypeMessage, frame.Type)
	require.Len(t, frame.Payload, 4)
	require.Equal(t, V2TypeReqWorkConn, binary.BigEndian.Uint16(frame.Payload[:2]))

	out, err := DecodeV2MessageFrame(frame)
	require.NoError(t, err)
	require.IsType(t, &ReqWorkConn{}, out)
}

func TestDecodeV2MessageFrameInto(t *testing.T) {
	in := &StartWorkConn{ProxyName: "tcp", SrcAddr: "127.0.0.1", SrcPort: 1234}
	frame, err := EncodeV2MessageFrame(in)
	require.NoError(t, err)

	var out StartWorkConn
	require.NoError(t, DecodeV2MessageFrameInto(frame, &out))
	require.Equal(t, *in, out)
}

func TestDecodeV2MessageFrameRejectsInvalidFrame(t *testing.T) {
	_, err := DecodeV2MessageFrame(&wire.Frame{Type: wire.FrameTypeClientHello})
	require.ErrorContains(t, err, "unexpected frame type")

	_, err = DecodeV2MessageFrame(&wire.Frame{Type: wire.FrameTypeMessage, Payload: []byte{0}})
	require.ErrorContains(t, err, "payload too short")

	payload := make([]byte, 4)
	binary.BigEndian.PutUint16(payload[:2], 65535)
	copy(payload[2:], []byte("{}"))
	_, err = DecodeV2MessageFrame(&wire.Frame{Type: wire.FrameTypeMessage, Payload: payload})
	require.ErrorContains(t, err, "unknown v2 message type")
}

func TestDecodeV2MessageFrameIntoRejectsWrongTarget(t *testing.T) {
	frame, err := EncodeV2MessageFrame(&ReqWorkConn{})
	require.NoError(t, err)

	var out StartWorkConn
	err = DecodeV2MessageFrameInto(frame, &out)
	require.ErrorContains(t, err, "unexpected message type")

	err = DecodeV2MessageFrameInto(frame, StartWorkConn{})
	require.ErrorContains(t, err, "must be a pointer")
}

func TestEncodeV2MessageFrameRejectsUnknownMessage(t *testing.T) {
	_, err := EncodeV2MessageFrame(struct{}{})
	require.ErrorContains(t, err, "unknown v2 message type")
}

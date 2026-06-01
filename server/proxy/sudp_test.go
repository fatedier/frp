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
	"bufio"
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/wire"
)

func TestSUDPBridgeTranscodesProxyV1ToVisitorV2(t *testing.T) {
	var in, out bytes.Buffer
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV1, &msg.UDPPacket{Content: []byte("proxy-to-visitor")})

	var count int64
	err := bridgeSUDPProxyToVisitor(
		msg.NewReadWriter(&in, wire.ProtocolV1),
		msg.NewReadWriter(&out, wire.ProtocolV2),
		&count,
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, int64(len("proxy-to-visitor")), count)

	frame, err := wire.NewConn(&out).ReadFrame()
	require.NoError(t, err)
	require.Equal(t, wire.FrameTypeMessage, frame.Type)
	require.GreaterOrEqual(t, len(frame.Payload), 2)
	require.Equal(t, msg.V2TypeUDPPacket, binary.BigEndian.Uint16(frame.Payload[:2]))

	var got msg.UDPPacket
	require.NoError(t, msg.DecodeV2MessageFrameInto(frame, &got))
	require.Equal(t, []byte("proxy-to-visitor"), got.Content)
}

func TestSUDPBridgeTranscodesVisitorV2ToProxyV1(t *testing.T) {
	var in, out bytes.Buffer
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV2, &msg.UDPPacket{Content: []byte("visitor-to-proxy")})

	var count int64
	err := bridgeSUDPVisitorToProxy(
		msg.NewReadWriter(&in, wire.ProtocolV2),
		msg.NewReadWriter(&out, wire.ProtocolV1),
		&count,
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, int64(len("visitor-to-proxy")), count)

	reader := bufio.NewReader(&out)
	typeByte, err := reader.ReadByte()
	require.NoError(t, err)
	require.Equal(t, msg.TypeUDPPacket, typeByte)
	require.NoError(t, reader.UnreadByte())

	var got msg.UDPPacket
	require.NoError(t, msg.ReadMsgInto(reader, &got))
	require.Equal(t, []byte("visitor-to-proxy"), got.Content)
}

func TestSUDPBridgeForwardsProxyPing(t *testing.T) {
	var in, out bytes.Buffer
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV1, &msg.Ping{})

	var count int64
	err := bridgeSUDPProxyToVisitor(
		msg.NewReadWriter(&in, wire.ProtocolV1),
		msg.NewReadWriter(&out, wire.ProtocolV2),
		&count,
		nil,
	)
	require.NoError(t, err)
	require.Zero(t, count)

	rawMsg, err := msg.NewReadWriter(&out, wire.ProtocolV2).ReadMsg()
	require.NoError(t, err)
	require.IsType(t, &msg.Ping{}, rawMsg)
}

func TestSUDPBridgeDropsVisitorPing(t *testing.T) {
	var in, out bytes.Buffer
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV2, &msg.Ping{})

	var count int64
	err := bridgeSUDPVisitorToProxy(
		msg.NewReadWriter(&in, wire.ProtocolV2),
		msg.NewReadWriter(&out, wire.ProtocolV1),
		&count,
		nil,
	)
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, out.Bytes())
}

func TestSUDPBridgeRejectsUnknownVisitorMessage(t *testing.T) {
	var in, out bytes.Buffer
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV2, &msg.Pong{})

	var count int64
	err := bridgeSUDPVisitorToProxy(
		msg.NewReadWriter(&in, wire.ProtocolV2),
		msg.NewReadWriter(&out, wire.ProtocolV1),
		&count,
		nil,
	)
	require.ErrorContains(t, err, "unexpected SUDP visitor message *msg.Pong")
	require.Zero(t, count)
	require.Empty(t, out.Bytes())
}

func TestSUDPBridgeDetectsMixedWireProtocol(t *testing.T) {
	require.False(t, isMixedWireProtocol("", wire.ProtocolV1))
	require.False(t, isMixedWireProtocol(wire.ProtocolV2, wire.ProtocolV2))
	require.True(t, isMixedWireProtocol("", wire.ProtocolV2))
	require.True(t, isMixedWireProtocol(wire.ProtocolV2, wire.ProtocolV1))
}

func writeSUDPBridgeMsg(t *testing.T, buf *bytes.Buffer, wireProtocol string, m msg.Message) {
	t.Helper()

	require.NoError(t, msg.NewReadWriter(buf, wireProtocol).WriteMsg(m))
}

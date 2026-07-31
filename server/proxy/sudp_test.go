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
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/wire"
	"github.com/fatedier/frp/pkg/util/xlog"
)

func TestSUDPBridgeTranscodesProxyV1ToVisitorV2(t *testing.T) {
	var in, out bytes.Buffer
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV1, "", &msg.UDPPacket{Content: []byte("proxy-to-visitor")})

	var count int64
	err := bridgeSUDPProxyToVisitor(
		newSUDPBridgeRW(t, &in, wire.ProtocolV1, ""),
		newSUDPBridgeRW(t, &out, wire.ProtocolV2, ""),
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
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV2, "", &msg.UDPPacket{Content: []byte("visitor-to-proxy")})

	var count int64
	err := bridgeSUDPVisitorToProxy(
		newSUDPBridgeRW(t, &in, wire.ProtocolV2, ""),
		newSUDPBridgeRW(t, &out, wire.ProtocolV1, ""),
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

func TestSUDPBridgeTranscodesProxyV2BinaryToVisitorV2JSON(t *testing.T) {
	var in, out bytes.Buffer
	packet := newSUDPBridgeUDPPacket("proxy-binary-to-json")
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV2, wire.UDPPacketCodecBinary, packet)

	var count int64
	err := bridgeSUDPProxyToVisitor(
		newSUDPBridgeRW(t, &in, wire.ProtocolV2, wire.UDPPacketCodecBinary),
		newSUDPBridgeRW(t, &out, wire.ProtocolV2, ""),
		&count,
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, int64(len(packet.Content)), count)
	requireV2UDPPacketFrame(t, &out, msg.V2TypeUDPPacket, packet)
}

func TestSUDPBridgeTranscodesVisitorV2JSONToProxyV2Binary(t *testing.T) {
	var in, out bytes.Buffer
	packet := newSUDPBridgeUDPPacket("visitor-json-to-binary")
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV2, "", packet)

	var count int64
	err := bridgeSUDPVisitorToProxy(
		newSUDPBridgeRW(t, &in, wire.ProtocolV2, ""),
		newSUDPBridgeRW(t, &out, wire.ProtocolV2, wire.UDPPacketCodecBinary),
		&count,
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, int64(len(packet.Content)), count)
	requireV2UDPPacketFrame(t, &out, msg.V2TypeUDPPacketBinary, packet)
}

func TestSUDPBridgeForwardsProxyPing(t *testing.T) {
	var in, out bytes.Buffer
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV1, "", &msg.Ping{})

	var count int64
	err := bridgeSUDPProxyToVisitor(
		newSUDPBridgeRW(t, &in, wire.ProtocolV1, ""),
		newSUDPBridgeRW(t, &out, wire.ProtocolV2, ""),
		&count,
		nil,
	)
	require.NoError(t, err)
	require.Zero(t, count)

	rawMsg, err := newSUDPBridgeRW(t, &out, wire.ProtocolV2, "").ReadMsg()
	require.NoError(t, err)
	require.IsType(t, &msg.Ping{}, rawMsg)
}

func TestSUDPBridgeDropsVisitorPing(t *testing.T) {
	var in, out bytes.Buffer
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV2, "", &msg.Ping{})

	var count int64
	err := bridgeSUDPVisitorToProxy(
		newSUDPBridgeRW(t, &in, wire.ProtocolV2, ""),
		newSUDPBridgeRW(t, &out, wire.ProtocolV1, ""),
		&count,
		nil,
	)
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, out.Bytes())
}

func TestSUDPBridgeRejectsUnknownVisitorMessage(t *testing.T) {
	var in, out bytes.Buffer
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV2, "", &msg.Pong{})

	var count int64
	err := bridgeSUDPVisitorToProxy(
		newSUDPBridgeRW(t, &in, wire.ProtocolV2, ""),
		newSUDPBridgeRW(t, &out, wire.ProtocolV1, ""),
		&count,
		nil,
	)
	require.ErrorContains(t, err, "unexpected SUDP visitor message *msg.Pong")
	require.Zero(t, count)
	require.Empty(t, out.Bytes())
}

func TestSUDPBridgeRejectsMismatchedPacketCodecOnStream(t *testing.T) {
	var in, out bytes.Buffer
	writeSUDPBridgeMsg(t, &in, wire.ProtocolV2, "", newSUDPBridgeUDPPacket("json-on-binary-stream"))

	var count int64
	err := bridgeSUDPProxyToVisitor(
		newSUDPBridgeRW(t, &in, wire.ProtocolV2, wire.UDPPacketCodecBinary),
		newSUDPBridgeRW(t, &out, wire.ProtocolV2, ""),
		&count,
		nil,
	)
	require.ErrorContains(t, err, "received JSON UDP packet after binary codec negotiation")
	require.Zero(t, count)
	require.Empty(t, out.Bytes())
}

func TestSUDPBridgeDetectsMixedWireProtocol(t *testing.T) {
	require.False(t, isMixedWireProtocol("", wire.ProtocolV1))
	require.False(t, isMixedWireProtocol(wire.ProtocolV2, wire.ProtocolV2))
	require.True(t, isMixedWireProtocol("", wire.ProtocolV2))
	require.True(t, isMixedWireProtocol(wire.ProtocolV2, wire.ProtocolV1))
}

func TestSUDPBridgeDetectsMixedPacketEncoding(t *testing.T) {
	for _, tc := range []struct {
		name       string
		leftWire   string
		leftCodec  string
		rightWire  string
		rightCodec string
		mixed      bool
	}{
		{name: "legacy v1 aliases explicit v1", leftWire: "", rightWire: wire.ProtocolV1},
		{name: "v2 json matches v2 json", leftWire: wire.ProtocolV2, rightWire: wire.ProtocolV2},
		{
			name:       "v2 binary matches v2 binary",
			leftWire:   wire.ProtocolV2,
			leftCodec:  wire.UDPPacketCodecBinary,
			rightWire:  wire.ProtocolV2,
			rightCodec: wire.UDPPacketCodecBinary,
		},
		{name: "v1 json differs from v2 json", leftWire: wire.ProtocolV1, rightWire: wire.ProtocolV2, mixed: true},
		{
			name:       "v2 json differs from v2 binary",
			leftWire:   wire.ProtocolV2,
			rightWire:  wire.ProtocolV2,
			rightCodec: wire.UDPPacketCodecBinary,
			mixed:      true,
		},
		{
			name:      "v2 binary differs from v1 json",
			leftWire:  wire.ProtocolV2,
			leftCodec: wire.UDPPacketCodecBinary,
			rightWire: wire.ProtocolV1,
			mixed:     true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mixed, err := isMixedSUDPPacketEncoding(tc.leftWire, tc.leftCodec, tc.rightWire, tc.rightCodec)
			require.NoError(t, err)
			require.Equal(t, tc.mixed, mixed)
		})
	}
}

func TestSUDPBridgeRejectsInvalidEncodingMetadata(t *testing.T) {
	for _, tc := range []struct {
		name       string
		leftWire   string
		leftCodec  string
		rightWire  string
		rightCodec string
		wantErr    string
	}{
		{
			name:      "left v1 binary",
			leftWire:  wire.ProtocolV1,
			leftCodec: wire.UDPPacketCodecBinary,
			rightWire: wire.ProtocolV1,
			wantErr:   "invalid left SUDP packet encoding",
		},
		{
			name:       "right unknown v2 codec",
			leftWire:   wire.ProtocolV2,
			rightWire:  wire.ProtocolV2,
			rightCodec: "snappy",
			wantErr:    "invalid right SUDP packet encoding",
		},
		{name: "left unknown wire", leftWire: "v3", rightWire: wire.ProtocolV2, wantErr: "unsupported wire protocol"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mixed, err := isMixedSUDPPacketEncoding(tc.leftWire, tc.leftCodec, tc.rightWire, tc.rightCodec)
			require.False(t, mixed)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestSUDPJoinUsesRawPathForSameEncodingState(t *testing.T) {
	proxyClient, proxyServer := net.Pipe()
	visitorClient, visitorServer := net.Pipe()
	t.Cleanup(func() {
		_ = proxyClient.Close()
		_ = proxyServer.Close()
		_ = visitorClient.Close()
		_ = visitorServer.Close()
	})
	deadline := time.Now().Add(3 * time.Second)
	require.NoError(t, proxyClient.SetDeadline(deadline))
	require.NoError(t, proxyServer.SetDeadline(deadline))
	require.NoError(t, visitorClient.SetDeadline(deadline))
	require.NoError(t, visitorServer.SetDeadline(deadline))

	pxy := &BaseProxy{
		configurer:     &v1.SUDPProxyConfig{},
		wireProtocol:   wire.ProtocolV2,
		udpPacketCodec: wire.UDPPacketCodecBinary,
	}
	visitorConn := &metadataConn{Conn: visitorServer, wireProtocol: wire.ProtocolV2, udpPacketCodec: wire.UDPPacketCodecBinary}
	joinDone := make(chan []error, 1)
	go func() {
		_, _, errs := pxy.joinUserConnection(proxyServer, visitorConn, string(v1.ProxyTypeSUDP), xlog.New())
		joinDone <- errs
	}()

	raw := []byte{0, 16, 0, 0, 0, 4, 0xde, 0xad, 0xbe, 0xef}
	_, err := proxyClient.Write(raw)
	require.NoError(t, err)
	got := make([]byte, len(raw))
	_, err = io.ReadFull(visitorClient, got)
	require.NoError(t, err)
	require.Equal(t, raw, got)

	_ = proxyClient.Close()
	_ = visitorClient.Close()
	<-joinDone
}

func newSUDPBridgeRW(t *testing.T, buf *bytes.Buffer, wireProtocol, udpPacketCodec string) msg.ReadWriter {
	t.Helper()
	rw, err := msg.NewUDPPacketReadWriter(buf, wireProtocol, udpPacketCodec)
	require.NoError(t, err)
	return rw
}

func writeSUDPBridgeMsg(t *testing.T, buf *bytes.Buffer, wireProtocol, udpPacketCodec string, m msg.Message) {
	t.Helper()
	require.NoError(t, newSUDPBridgeRW(t, buf, wireProtocol, udpPacketCodec).WriteMsg(m))
}

func newSUDPBridgeUDPPacket(content string) *msg.UDPPacket {
	return &msg.UDPPacket{
		Content:    []byte(content),
		RemoteAddr: &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 12345},
	}
}

func requireV2UDPPacketFrame(t *testing.T, buf *bytes.Buffer, wantType uint16, want *msg.UDPPacket) {
	t.Helper()
	frame, err := wire.NewConn(buf).ReadFrame()
	require.NoError(t, err)
	require.Equal(t, wire.FrameTypeMessage, frame.Type)
	require.GreaterOrEqual(t, len(frame.Payload), 2)
	require.Equal(t, wantType, binary.BigEndian.Uint16(frame.Payload[:2]))
	var got *msg.UDPPacket
	if wantType == msg.V2TypeUDPPacketBinary {
		got, err = msg.DecodeUDPPacketBinary(frame.Payload[2:])
	} else {
		var decoded msg.UDPPacket
		err = msg.DecodeV2MessageFrameInto(frame, &decoded)
		got = &decoded
	}
	require.NoError(t, err)
	require.Equal(t, want.Content, got.Content)
	require.Equal(t, want.RemoteAddr.String(), got.RemoteAddr.String())
}

type metadataConn struct {
	net.Conn
	wireProtocol   string
	udpPacketCodec string
}

func (c *metadataConn) WireProtocol() string {
	return c.wireProtocol
}

func (c *metadataConn) UDPPacketCodec() string {
	return c.udpPacketCodec
}

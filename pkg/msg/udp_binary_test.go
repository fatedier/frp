// Copyright 2026 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package msg

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/proto/wire"
)

func TestUDPPacketBinaryRoundTrip(t *testing.T) {
	payload := bytes.Repeat([]byte{0xa5}, 1472)
	in := &UDPPacket{
		Content: payload,
		LocalAddr: &net.UDPAddr{
			IP:   net.ParseIP("2001:db8::1"),
			Port: 1234,
			Zone: "en0",
		},
		RemoteAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.9"), Port: 54321},
	}
	body, err := EncodeUDPPacketBinary(in)
	require.NoError(t, err)
	out, err := DecodeUDPPacketBinary(body)
	require.NoError(t, err)
	require.Equal(t, in.Content, out.Content)
	require.Equal(t, in.LocalAddr.String(), out.LocalAddr.String())
	require.Equal(t, in.RemoteAddr.String(), out.RemoteAddr.String())
	body[len(body)-1] ^= 0xff
	body[25] ^= 0xff
	require.Equal(t, byte(0xa5), out.Content[len(out.Content)-1], "decoded payload must own frame bytes")
	require.Equal(t, byte(203), out.RemoteAddr.IP.To4()[0], "decoded address must own frame bytes")
}

func TestUDPPacketBinarySizesAndOptionalLocalAddress(t *testing.T) {
	for _, size := range []int{0, 32, 128, 512, 1200, 1472, 4096, 49107, 65507} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			in := &UDPPacket{
				Content:    bytes.Repeat([]byte{byte(size)}, size),
				RemoteAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.9"), Port: 54321},
			}
			body, err := EncodeUDPPacketBinary(in)
			require.NoError(t, err)
			out, err := DecodeUDPPacketBinary(body)
			require.NoError(t, err)
			require.Equal(t, len(in.Content), len(out.Content))
			if size == 0 {
				require.Empty(t, out.Content)
			} else {
				require.Equal(t, in.Content, out.Content)
			}
		})
	}
}

func TestUDPPacketBinaryMalformed(t *testing.T) {
	valid, err := EncodeUDPPacketBinary(&UDPPacket{
		Content:    []byte("payload"),
		RemoteAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.9"), Port: 54321},
	})
	require.NoError(t, err)
	tests := [][]byte{
		{0x80, 0, 0},
		{0x02, 4, 1, 2},
		{0x02, 4, 1, 2, 3, 4, 0xd4},
		append(append([]byte(nil), valid...), 0),
	}
	for _, malformed := range tests {
		_, err := DecodeUDPPacketBinary(malformed)
		require.Error(t, err)
	}
	_, err = DecodeUDPPacketBinary([]byte{0, 0, 0})
	require.ErrorContains(t, err, "missing remote address")
	payloadLengthOffset := len(valid) - len("payload") - 2
	invalidPayloadLength := append([]byte(nil), valid...)
	binary.BigEndian.PutUint16(invalidPayloadLength[payloadLengthOffset:payloadLengthOffset+2], 0xffff)
	_, err = DecodeUDPPacketBinary(invalidPayloadLength)
	require.ErrorContains(t, err, "payload length")
	truncatedPayload := append([]byte(nil), valid[:payloadLengthOffset+2]...)
	binary.BigEndian.PutUint16(truncatedPayload[payloadLengthOffset:payloadLengthOffset+2], 1)
	_, err = DecodeUDPPacketBinary(truncatedPayload)
	require.ErrorContains(t, err, "truncated UDP payload")
	_, err = DecodeUDPPacketBinary(make([]byte, wire.DefaultMaxFramePayloadSize))
	require.ErrorContains(t, err, "frame payload length")

	badIPv4Zone := []byte{2, 4, 203, 0, 113, 9, 0xd4, 0x31, 1, 'z', 0, 0}
	_, err = DecodeUDPPacketBinary(badIPv4Zone)
	require.ErrorContains(t, err, "IPv4 zone")
	badFamily := []byte{2, 9, 0, 0}
	_, err = DecodeUDPPacketBinary(badFamily)
	require.ErrorContains(t, err, "unknown address family")
	badUTF8 := make([]byte, 0, 24)
	badUTF8 = append(badUTF8, 2, 6)
	badUTF8 = append(badUTF8, make([]byte, 16)...)
	badUTF8 = append(badUTF8, 0, 1, 1, 0xff, 0, 0)
	_, err = DecodeUDPPacketBinary(badUTF8)
	require.ErrorContains(t, err, "UTF-8")
}

func TestUDPPacketBinaryEncodeRejectsInvalidPackets(t *testing.T) {
	_, err := EncodeUDPPacketBinary(&UDPPacket{})
	require.ErrorContains(t, err, "missing remote address")
	_, err = EncodeUDPPacketBinary(&UDPPacket{
		LocalAddr: &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1234},
	})
	require.ErrorContains(t, err, "missing remote address")
	_, err = EncodeUDPPacketBinary(&UDPPacket{
		Content:    make([]byte, MaxUDPPayloadSize+1),
		RemoteAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.9"), Port: 54321},
	})
	require.ErrorContains(t, err, "exceeds limit")
	_, err = EncodeUDPPacketBinary(&UDPPacket{RemoteAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.9"), Port: 1, Zone: "bad"}})
	require.ErrorContains(t, err, "IPv4 zone")
	_, err = EncodeUDPPacketBinary(&UDPPacket{RemoteAddr: &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 1, Zone: string(bytes.Repeat([]byte{'z'}, 256))}})
	require.ErrorContains(t, err, "zone exceeds")
	_, err = EncodeUDPPacketBinary(&UDPPacket{RemoteAddr: &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 1, Zone: string([]byte{0xff})}})
	require.ErrorContains(t, err, "UTF-8")
	_, err = EncodeUDPPacketBinary(&UDPPacket{RemoteAddr: &net.UDPAddr{Port: -1}})
	require.ErrorContains(t, err, "port out of range")
	_, err = EncodeUDPPacketBinary(&UDPPacket{RemoteAddr: &net.UDPAddr{Port: 65536}})
	require.ErrorContains(t, err, "port out of range")
	_, err = EncodeUDPPacketBinary(&UDPPacket{RemoteAddr: &net.UDPAddr{IP: net.IP{1, 2, 3}}})
	require.ErrorContains(t, err, "invalid IP")
	_, err = EncodeUDPPacketBinary(&UDPPacket{
		Content:    make([]byte, MaxUDPPayloadSize),
		RemoteAddr: &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Zone: string(bytes.Repeat([]byte{'z'}, 255))},
	})
	require.ErrorContains(t, err, "frame payload length")
}

func TestV2BinaryUDPPacketReadWriterPreservesOtherMessages(t *testing.T) {
	var buf bytes.Buffer
	rw := NewV2BinaryUDPPacketReadWriter(&buf)
	in := &UDPPacket{Content: []byte("udp"), RemoteAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.9"), Port: 54321}}
	require.NoError(t, rw.WriteMsg(in))
	require.NoError(t, rw.WriteMsg(&Ping{Timestamp: 7}))
	frameConn := wire.NewConn(&buf)
	frame, err := frameConn.ReadFrame()
	require.NoError(t, err)
	require.Equal(t, V2TypeUDPPacketBinary, binary.BigEndian.Uint16(frame.Payload[:2]))
	frame, err = frameConn.ReadFrame()
	require.NoError(t, err)
	require.Equal(t, V2TypePing, binary.BigEndian.Uint16(frame.Payload[:2]))
}

func TestV2BinaryUDPPacketReadWriterRoundTripAndCodecInvariant(t *testing.T) {
	in := &UDPPacket{Content: []byte("udp"), RemoteAddr: &net.UDPAddr{IP: net.ParseIP("203.0.113.9"), Port: 54321}}
	var binaryStream bytes.Buffer
	binaryWriter, err := NewUDPPacketReadWriter(&binaryStream, wire.ProtocolV2, wire.UDPPacketCodecBinary)
	require.NoError(t, err)
	require.NoError(t, binaryWriter.WriteMsg(in))
	binaryReader, err := NewUDPPacketReadWriter(&binaryStream, wire.ProtocolV2, wire.UDPPacketCodecBinary)
	require.NoError(t, err)
	out, err := binaryReader.ReadMsg()
	require.NoError(t, err)
	require.Equal(t, in.Content, out.(*UDPPacket).Content)

	for _, read := range []func(ReadWriter) error{
		func(rw ReadWriter) error {
			_, err := rw.ReadMsg()
			return err
		},
		func(rw ReadWriter) error {
			return rw.ReadMsgInto(&UDPPacket{})
		},
	} {
		var jsonStream bytes.Buffer
		require.NoError(t, NewReadWriter(&jsonStream, wire.ProtocolV2).WriteMsg(in))
		negotiatedReader, err := NewUDPPacketReadWriter(&jsonStream, wire.ProtocolV2, wire.UDPPacketCodecBinary)
		require.NoError(t, err)
		require.ErrorContains(t, read(negotiatedReader), "JSON UDP packet after binary codec negotiation")
	}

	var fallbackStream bytes.Buffer
	fallbackWriter, err := NewUDPPacketReadWriter(&fallbackStream, wire.ProtocolV2, "")
	require.NoError(t, err)
	require.NoError(t, fallbackWriter.WriteMsg(in))
	frame, err := wire.NewConn(&fallbackStream).ReadFrame()
	require.NoError(t, err)
	require.Equal(t, V2TypeUDPPacket, binary.BigEndian.Uint16(frame.Payload[:2]))
}

func TestNewUDPPacketReadWriterDefaultProtocolUsesV1(t *testing.T) {
	var stream bytes.Buffer
	rw, err := NewUDPPacketReadWriter(&stream, "", "")
	require.NoError(t, err)
	require.IsType(t, &V1ReadWriter{}, rw)
	require.NoError(t, rw.WriteMsg(&UDPPacket{Content: []byte("legacy")}))
	require.Equal(t, TypeUDPPacket, stream.Bytes()[0])
}

func TestNewUDPPacketReadWriterRejectsInvalidSelection(t *testing.T) {
	for _, tc := range []struct {
		name           string
		wireProtocol   string
		udpPacketCodec string
		errorSubstring string
	}{
		{
			name:           "binary codec over v1",
			wireProtocol:   wire.ProtocolV1,
			udpPacketCodec: wire.UDPPacketCodecBinary,
			errorSubstring: "requires wire protocol v2",
		},
		{
			name:           "binary codec over default protocol",
			udpPacketCodec: wire.UDPPacketCodecBinary,
			errorSubstring: "requires wire protocol v2",
		},
		{
			name:           "unknown v2 codec",
			wireProtocol:   wire.ProtocolV2,
			udpPacketCodec: "unknown",
			errorSubstring: "unsupported UDP packet codec",
		},
		{
			name:           "unknown wire protocol",
			wireProtocol:   "unknown",
			errorSubstring: "unsupported wire protocol",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rw, err := NewUDPPacketReadWriter(&bytes.Buffer{}, tc.wireProtocol, tc.udpPacketCodec)
			require.Nil(t, rw)
			require.ErrorContains(t, err, tc.errorSubstring)
		})
	}
}

func FuzzDecodeUDPPacketBinary(f *testing.F) {
	f.Add([]byte{0, 0, 0})
	f.Add([]byte{2, 4, 203, 0, 113, 9, 0xd4, 0x31, 0, 0, 1})
	f.Fuzz(func(t *testing.T, body []byte) {
		_, _ = DecodeUDPPacketBinary(body)
	})
}

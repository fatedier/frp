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
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"unicode/utf8"

	"github.com/fatedier/frp/pkg/proto/wire"
)

const MaxUDPPayloadSize = 65507

const (
	udpPacketFlagLocalAddr  byte = 1 << 0
	udpPacketFlagRemoteAddr byte = 1 << 1
	udpPacketValidFlags          = udpPacketFlagLocalAddr | udpPacketFlagRemoteAddr
)

type binaryUDPAddr struct {
	family byte
	ip     []byte
	port   uint16
	zone   string
}

// EncodeUDPPacketBinary encodes the body of a V2 binary UDP packet message.
// RemoteAddr is required by the UDP forwarding path.
func EncodeUDPPacketBinary(packet *UDPPacket) ([]byte, error) {
	if packet == nil {
		return nil, fmt.Errorf("nil UDP packet")
	}
	if packet.RemoteAddr == nil {
		return nil, fmt.Errorf("UDP packet missing remote address")
	}
	if len(packet.Content) > MaxUDPPayloadSize {
		return nil, fmt.Errorf("UDP payload length %d exceeds limit %d", len(packet.Content), MaxUDPPayloadSize)
	}

	var flags byte
	var localAddr, remoteAddr binaryUDPAddr
	bodyLen := 1 + 2 + len(packet.Content)
	if packet.LocalAddr != nil {
		flags |= udpPacketFlagLocalAddr
		var err error
		localAddr, err = validateBinaryUDPAddr(packet.LocalAddr)
		if err != nil {
			return nil, fmt.Errorf("local address: %w", err)
		}
		bodyLen += binaryUDPAddrLen(localAddr)
	}
	flags |= udpPacketFlagRemoteAddr
	var err error
	remoteAddr, err = validateBinaryUDPAddr(packet.RemoteAddr)
	if err != nil {
		return nil, fmt.Errorf("remote address: %w", err)
	}
	bodyLen += binaryUDPAddrLen(remoteAddr)
	if 2+bodyLen > wire.DefaultMaxFramePayloadSize {
		return nil, fmt.Errorf("v2 frame payload length %d exceeds limit %d", 2+bodyLen, wire.DefaultMaxFramePayloadSize)
	}

	body := make([]byte, bodyLen)
	body[0] = flags
	offset := 1
	if flags&udpPacketFlagLocalAddr != 0 {
		offset = putBinaryUDPAddr(body, offset, localAddr)
	}
	offset = putBinaryUDPAddr(body, offset, remoteAddr)
	binary.BigEndian.PutUint16(body[offset:offset+2], uint16(len(packet.Content)))
	offset += 2
	copy(body[offset:], packet.Content)
	return body, nil
}

// DecodeUDPPacketBinary decodes a V2 binary UDP packet body and returns data
// that does not alias the input frame buffer.
func DecodeUDPPacketBinary(body []byte) (*UDPPacket, error) {
	if len(body) < 3 {
		return nil, fmt.Errorf("UDP packet body too short: %d", len(body))
	}
	if 2+len(body) > wire.DefaultMaxFramePayloadSize {
		return nil, fmt.Errorf("v2 frame payload length %d exceeds limit %d", 2+len(body), wire.DefaultMaxFramePayloadSize)
	}

	flags := body[0]
	if flags&^udpPacketValidFlags != 0 {
		return nil, fmt.Errorf("reserved UDP packet flags set: 0x%02x", flags)
	}
	if flags&udpPacketFlagRemoteAddr == 0 {
		return nil, fmt.Errorf("UDP packet missing remote address")
	}

	packet := &UDPPacket{}
	offset := 1
	var err error
	if flags&udpPacketFlagLocalAddr != 0 {
		packet.LocalAddr, offset, err = readBinaryUDPAddr(body, offset)
		if err != nil {
			return nil, fmt.Errorf("local address: %w", err)
		}
	}
	if flags&udpPacketFlagRemoteAddr != 0 {
		packet.RemoteAddr, offset, err = readBinaryUDPAddr(body, offset)
		if err != nil {
			return nil, fmt.Errorf("remote address: %w", err)
		}
	}
	if len(body)-offset < 2 {
		return nil, fmt.Errorf("truncated UDP payload length")
	}
	payloadLen := int(binary.BigEndian.Uint16(body[offset : offset+2]))
	offset += 2
	if payloadLen > MaxUDPPayloadSize {
		return nil, fmt.Errorf("UDP payload length %d exceeds limit %d", payloadLen, MaxUDPPayloadSize)
	}
	remaining := len(body) - offset
	if remaining < payloadLen {
		return nil, fmt.Errorf("truncated UDP payload: have %d want %d", remaining, payloadLen)
	}
	if remaining > payloadLen {
		return nil, fmt.Errorf("trailing UDP packet bytes: %d", remaining-payloadLen)
	}
	packet.Content = append([]byte(nil), body[offset:offset+payloadLen]...)
	return packet, nil
}

func validateBinaryUDPAddr(addr *net.UDPAddr) (binaryUDPAddr, error) {
	if addr.Port < 0 || addr.Port > 65535 {
		return binaryUDPAddr{}, fmt.Errorf("port out of range: %d", addr.Port)
	}
	if ip := addr.IP.To4(); ip != nil {
		if addr.Zone != "" {
			return binaryUDPAddr{}, fmt.Errorf("IPv4 zone is forbidden")
		}
		return binaryUDPAddr{family: 4, ip: ip, port: uint16(addr.Port)}, nil
	}
	ip := addr.IP.To16()
	if ip == nil {
		return binaryUDPAddr{}, fmt.Errorf("invalid IP")
	}
	if len(addr.Zone) > 255 {
		return binaryUDPAddr{}, fmt.Errorf("zone exceeds 255 bytes")
	}
	if !utf8.ValidString(addr.Zone) {
		return binaryUDPAddr{}, fmt.Errorf("zone is not valid UTF-8")
	}
	return binaryUDPAddr{family: 6, ip: ip, port: uint16(addr.Port), zone: addr.Zone}, nil
}

func binaryUDPAddrLen(addr binaryUDPAddr) int {
	return 1 + len(addr.ip) + 2 + 1 + len(addr.zone)
}

func putBinaryUDPAddr(body []byte, offset int, addr binaryUDPAddr) int {
	body[offset] = addr.family
	offset++
	copy(body[offset:], addr.ip)
	offset += len(addr.ip)
	binary.BigEndian.PutUint16(body[offset:offset+2], addr.port)
	offset += 2
	body[offset] = byte(len(addr.zone))
	offset++
	copy(body[offset:], addr.zone)
	return offset + len(addr.zone)
}

func readBinaryUDPAddr(body []byte, offset int) (*net.UDPAddr, int, error) {
	if offset >= len(body) {
		return nil, offset, fmt.Errorf("truncated address family")
	}
	family := body[offset]
	offset++
	var ipLen int
	switch family {
	case 4:
		ipLen = net.IPv4len
	case 6:
		ipLen = net.IPv6len
	default:
		return nil, offset, fmt.Errorf("unknown address family %d", family)
	}
	if len(body)-offset < ipLen+3 {
		return nil, offset, fmt.Errorf("truncated address")
	}
	ip := append(net.IP(nil), body[offset:offset+ipLen]...)
	offset += ipLen
	port := binary.BigEndian.Uint16(body[offset : offset+2])
	offset += 2
	zoneLen := int(body[offset])
	offset++
	if len(body)-offset < zoneLen {
		return nil, offset, fmt.Errorf("truncated zone")
	}
	zoneBytes := body[offset : offset+zoneLen]
	if family == 4 && zoneLen != 0 {
		return nil, offset, fmt.Errorf("IPv4 zone is forbidden")
	}
	if !utf8.Valid(zoneBytes) {
		return nil, offset, fmt.Errorf("zone is not valid UTF-8")
	}
	offset += zoneLen
	return &net.UDPAddr{IP: ip, Port: int(port), Zone: string(zoneBytes)}, offset, nil
}

type V2BinaryUDPPacketReadWriter struct {
	conn *wire.Conn
}

func NewV2BinaryUDPPacketReadWriter(rw io.ReadWriter) *V2BinaryUDPPacketReadWriter {
	return &V2BinaryUDPPacketReadWriter{conn: wire.NewConn(rw)}
}

func (rw *V2BinaryUDPPacketReadWriter) ReadMsg() (Message, error) {
	frame, err := rw.conn.ReadFrame()
	if err != nil {
		return nil, err
	}
	if isV2MessageType(frame, V2TypeUDPPacketBinary) {
		return decodeV2BinaryUDPPacketFrame(frame)
	}
	if isV2MessageType(frame, V2TypeUDPPacket) {
		return nil, fmt.Errorf("received JSON UDP packet after binary codec negotiation")
	}
	return DecodeV2MessageFrame(frame)
}

func (rw *V2BinaryUDPPacketReadWriter) ReadMsgInto(out Message) error {
	frame, err := rw.conn.ReadFrame()
	if err != nil {
		return err
	}
	if packetOut, ok := out.(*UDPPacket); ok {
		if !isV2MessageType(frame, V2TypeUDPPacketBinary) {
			return unexpectedV2UDPPacketType(frame)
		}
		packet, err := decodeV2BinaryUDPPacketFrame(frame)
		if err != nil {
			return err
		}
		*packetOut = *packet
		return nil
	}
	return DecodeV2MessageFrameInto(frame, out)
}

func (rw *V2BinaryUDPPacketReadWriter) WriteMsg(message Message) error {
	var packet *UDPPacket
	switch typed := message.(type) {
	case *UDPPacket:
		packet = typed
	case UDPPacket:
		packet = &typed
	default:
		frame, err := EncodeV2MessageFrame(message)
		if err != nil {
			return err
		}
		return rw.conn.WriteFrame(frame)
	}
	body, err := EncodeUDPPacketBinary(packet)
	if err != nil {
		return err
	}
	payload := make([]byte, 2+len(body))
	binary.BigEndian.PutUint16(payload[:2], V2TypeUDPPacketBinary)
	copy(payload[2:], body)
	return rw.conn.WriteFrame(&wire.Frame{Type: wire.FrameTypeMessage, Payload: payload})
}

func decodeV2BinaryUDPPacketFrame(frame *wire.Frame) (*UDPPacket, error) {
	if frame.Type != wire.FrameTypeMessage {
		return nil, fmt.Errorf("unexpected frame type %d, want %d", frame.Type, wire.FrameTypeMessage)
	}
	if len(frame.Payload) < 2 {
		return nil, fmt.Errorf("message frame payload too short")
	}
	if binary.BigEndian.Uint16(frame.Payload[:2]) != V2TypeUDPPacketBinary {
		return nil, unexpectedV2UDPPacketType(frame)
	}
	return DecodeUDPPacketBinary(frame.Payload[2:])
}

func isV2MessageType(frame *wire.Frame, typeID uint16) bool {
	return frame.Type == wire.FrameTypeMessage && len(frame.Payload) >= 2 && binary.BigEndian.Uint16(frame.Payload[:2]) == typeID
}

func unexpectedV2UDPPacketType(frame *wire.Frame) error {
	if frame.Type != wire.FrameTypeMessage {
		return fmt.Errorf("unexpected frame type %d, want %d", frame.Type, wire.FrameTypeMessage)
	}
	if len(frame.Payload) < 2 {
		return fmt.Errorf("message frame payload too short")
	}
	typeID := binary.BigEndian.Uint16(frame.Payload[:2])
	if typeID == V2TypeUDPPacket {
		return fmt.Errorf("received JSON UDP packet after binary codec negotiation")
	}
	return fmt.Errorf("unexpected message type %d, want %d", typeID, V2TypeUDPPacketBinary)
}

// NewUDPPacketReadWriter selects the negotiated packet codec without changing
// the framing or codecs used by non-UDP messages on the work connection.
func NewUDPPacketReadWriter(rw io.ReadWriter, wireProtocol, udpPacketCodec string) (ReadWriter, error) {
	switch wireProtocol {
	case "", wire.ProtocolV1:
		if udpPacketCodec != "" {
			return nil, fmt.Errorf("UDP packet codec %q requires wire protocol v2", udpPacketCodec)
		}
		return NewV1ReadWriter(rw), nil
	case wire.ProtocolV2:
		switch udpPacketCodec {
		case "":
			return NewV2ReadWriter(rw), nil
		case wire.UDPPacketCodecBinary:
			return NewV2BinaryUDPPacketReadWriter(rw), nil
		default:
			return nil, fmt.Errorf("unsupported UDP packet codec %q", udpPacketCodec)
		}
	default:
		return nil, fmt.Errorf("unsupported wire protocol %q", wireProtocol)
	}
}

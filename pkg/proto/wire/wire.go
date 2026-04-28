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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"slices"

	libnet "github.com/fatedier/golib/net"
)

const (
	ProtocolV1 = "v1"
	ProtocolV2 = "v2"

	WireVersionV2 = 2

	FrameTypeClientHello uint16 = 1
	FrameTypeServerHello uint16 = 2
	FrameTypeMessage     uint16 = 16

	MessageCodecJSON           = "json"
	DefaultMaxFramePayloadSize = 64 * 1024

	MagicV2 = "FRP\x00\x02\r\n"
)

type Frame struct {
	Type    uint16
	Flags   uint16
	Payload []byte
}

type Conn struct {
	rw                  io.ReadWriter
	maxFramePayloadSize uint32
}

func NewConn(rw io.ReadWriter) *Conn {
	return &Conn{
		rw:                  rw,
		maxFramePayloadSize: DefaultMaxFramePayloadSize,
	}
}

func (c *Conn) ReadFrame() (*Frame, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(c.rw, header); err != nil {
		return nil, err
	}

	frameType := binary.BigEndian.Uint16(header[0:2])
	flags := binary.BigEndian.Uint16(header[2:4])
	length := binary.BigEndian.Uint32(header[4:8])
	if flags != 0 {
		return nil, fmt.Errorf("unsupported frame flags: %d", flags)
	}
	if length > c.maxFramePayloadSize {
		return nil, fmt.Errorf("frame payload length %d exceeds limit %d", length, c.maxFramePayloadSize)
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(c.rw, payload); err != nil {
		return nil, err
	}
	return &Frame{
		Type:    frameType,
		Flags:   flags,
		Payload: payload,
	}, nil
}

func (c *Conn) WriteFrame(f *Frame) error {
	if f.Flags != 0 {
		return fmt.Errorf("unsupported frame flags: %d", f.Flags)
	}
	if len(f.Payload) > int(c.maxFramePayloadSize) {
		return fmt.Errorf("frame payload length %d exceeds limit %d", len(f.Payload), c.maxFramePayloadSize)
	}

	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], f.Type)
	binary.BigEndian.PutUint16(header[2:4], f.Flags)
	binary.BigEndian.PutUint32(header[4:8], uint32(len(f.Payload)))
	if _, err := c.rw.Write(header); err != nil {
		return err
	}
	_, err := c.rw.Write(f.Payload)
	return err
}

func (c *Conn) ReadJSONFrame(frameType uint16, out any) error {
	f, err := c.ReadFrame()
	if err != nil {
		return err
	}
	if f.Type != frameType {
		return fmt.Errorf("unexpected frame type %d, want %d", f.Type, frameType)
	}
	return c.UnmarshalFrame(f, out)
}

func (c *Conn) UnmarshalFrame(f *Frame, out any) error {
	return json.Unmarshal(f.Payload, out)
}

func (c *Conn) WriteJSONFrame(frameType uint16, in any) error {
	payload, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return c.WriteFrame(&Frame{
		Type:    frameType,
		Payload: payload,
	})
}

func WriteMagic(w io.Writer) error {
	_, err := io.WriteString(w, MagicV2)
	return err
}

func WriteMagicIfV2(w io.Writer, wireProtocol string) error {
	if wireProtocol != ProtocolV2 {
		return nil
	}
	return WriteMagic(w)
}

func CheckMagic(conn net.Conn) (out net.Conn, isV2 bool, err error) {
	sharedConn, r := libnet.NewSharedConnSize(conn, len(MagicV2))
	buf := make([]byte, len(MagicV2))
	if _, err = io.ReadFull(r, buf); err != nil {
		return nil, false, err
	}
	for i := range MagicV2 {
		if buf[i] != MagicV2[i] {
			return sharedConn, false, nil
		}
	}
	return conn, true, nil
}

type BootstrapInfo struct {
	Transport string `json:"transport,omitempty"`
	TLS       bool   `json:"tls,omitempty"`
	TCPMux    bool   `json:"tcpMux,omitempty"`
}

type ClientHello struct {
	Bootstrap    BootstrapInfo      `json:"bootstrap,omitempty"`
	Capabilities ClientCapabilities `json:"capabilities,omitempty"`
}

type ClientCapabilities struct {
	Message MessageCapabilities `json:"message,omitempty"`
}

type MessageCapabilities struct {
	Codecs []string `json:"codecs,omitempty"`
}

type ServerHello struct {
	Selected ServerSelection `json:"selected,omitempty"`
	Error    string          `json:"error,omitempty"`
}

type ServerSelection struct {
	Message MessageSelection `json:"message,omitempty"`
}

type MessageSelection struct {
	Codec string `json:"codec,omitempty"`
}

func DefaultClientHello(bootstrap BootstrapInfo) ClientHello {
	return ClientHello{
		Bootstrap: bootstrap,
		Capabilities: ClientCapabilities{
			Message: MessageCapabilities{
				Codecs: []string{MessageCodecJSON},
			},
		},
	}
}

func DefaultServerHello() ServerHello {
	return ServerHello{
		Selected: ServerSelection{
			Message: MessageSelection{
				Codec: MessageCodecJSON,
			},
		},
	}
}

func Supports(list []string, value string) bool {
	return slices.Contains(list, value)
}

func ValidateClientHello(h ClientHello) error {
	if !Supports(h.Capabilities.Message.Codecs, MessageCodecJSON) {
		return fmt.Errorf("unsupported message codec")
	}
	return nil
}

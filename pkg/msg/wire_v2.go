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
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/fatedier/frp/pkg/proto/wire"
)

const (
	V2TypeLogin              uint16 = 1
	V2TypeLoginResp          uint16 = 2
	V2TypeNewProxy           uint16 = 3
	V2TypeNewProxyResp       uint16 = 4
	V2TypeCloseProxy         uint16 = 5
	V2TypeNewWorkConn        uint16 = 6
	V2TypeReqWorkConn        uint16 = 7
	V2TypeStartWorkConn      uint16 = 8
	V2TypeNewVisitorConn     uint16 = 9
	V2TypeNewVisitorConnResp uint16 = 10
	V2TypePing               uint16 = 11
	V2TypePong               uint16 = 12
	V2TypeUDPPacket          uint16 = 13
	V2TypeNatHoleVisitor     uint16 = 14
	V2TypeNatHoleClient      uint16 = 15
	V2TypeNatHoleResp        uint16 = 16
	V2TypeNatHoleSid         uint16 = 17
	V2TypeNatHoleReport      uint16 = 18
)

var v2MsgTypeMap = map[uint16]any{
	V2TypeLogin:              Login{},
	V2TypeLoginResp:          LoginResp{},
	V2TypeNewProxy:           NewProxy{},
	V2TypeNewProxyResp:       NewProxyResp{},
	V2TypeCloseProxy:         CloseProxy{},
	V2TypeNewWorkConn:        NewWorkConn{},
	V2TypeReqWorkConn:        ReqWorkConn{},
	V2TypeStartWorkConn:      StartWorkConn{},
	V2TypeNewVisitorConn:     NewVisitorConn{},
	V2TypeNewVisitorConnResp: NewVisitorConnResp{},
	V2TypePing:               Ping{},
	V2TypePong:               Pong{},
	V2TypeUDPPacket:          UDPPacket{},
	V2TypeNatHoleVisitor:     NatHoleVisitor{},
	V2TypeNatHoleClient:      NatHoleClient{},
	V2TypeNatHoleResp:        NatHoleResp{},
	V2TypeNatHoleSid:         NatHoleSid{},
	V2TypeNatHoleReport:      NatHoleReport{},
}

var v2MsgReflectTypeMap, v2MsgTypeIDMap = buildV2MsgTypeMaps()

func buildV2MsgTypeMaps() (map[uint16]reflect.Type, map[reflect.Type]uint16) {
	reflectTypeMap := make(map[uint16]reflect.Type, len(v2MsgTypeMap))
	typeIDMap := make(map[reflect.Type]uint16, len(v2MsgTypeMap))
	for typeID, m := range v2MsgTypeMap {
		t := reflect.TypeOf(m)
		reflectTypeMap[typeID] = t
		typeIDMap[t] = typeID
	}
	return reflectTypeMap, typeIDMap
}

type V2ReadWriter struct {
	conn *wire.Conn
}

func NewV2ReadWriter(rw io.ReadWriter) *V2ReadWriter {
	return NewV2ReadWriterWithConn(wire.NewConn(rw))
}

func NewV2ReadWriterWithConn(conn *wire.Conn) *V2ReadWriter {
	return &V2ReadWriter{conn: conn}
}

func (rw *V2ReadWriter) WireConn() *wire.Conn {
	return rw.conn
}

func (rw *V2ReadWriter) ReadMsg() (Message, error) {
	f, err := rw.conn.ReadFrame()
	if err != nil {
		return nil, err
	}
	return DecodeV2MessageFrame(f)
}

func (rw *V2ReadWriter) ReadMsgInto(m Message) error {
	f, err := rw.conn.ReadFrame()
	if err != nil {
		return err
	}
	return DecodeV2MessageFrameInto(f, m)
}

func (rw *V2ReadWriter) WriteMsg(m Message) error {
	f, err := EncodeV2MessageFrame(m)
	if err != nil {
		return err
	}
	return rw.conn.WriteFrame(f)
}

func DecodeV2MessageFrame(f *wire.Frame) (Message, error) {
	if f.Type != wire.FrameTypeMessage {
		return nil, fmt.Errorf("unexpected frame type %d, want %d", f.Type, wire.FrameTypeMessage)
	}
	if len(f.Payload) < 2 {
		return nil, fmt.Errorf("message frame payload too short")
	}
	typeID := binary.BigEndian.Uint16(f.Payload[:2])
	t, ok := v2MsgReflectTypeMap[typeID]
	if !ok {
		return nil, fmt.Errorf("unknown v2 message type %d", typeID)
	}
	m := reflect.New(t).Interface()
	if err := json.Unmarshal(f.Payload[2:], m); err != nil {
		return nil, err
	}
	return m, nil
}

func DecodeV2MessageFrameInto(f *wire.Frame, out Message) error {
	if f.Type != wire.FrameTypeMessage {
		return fmt.Errorf("unexpected frame type %d, want %d", f.Type, wire.FrameTypeMessage)
	}
	if len(f.Payload) < 2 {
		return fmt.Errorf("message frame payload too short")
	}

	typeID := binary.BigEndian.Uint16(f.Payload[:2])
	outType := reflect.TypeOf(out)
	if outType == nil || outType.Kind() != reflect.Pointer {
		return fmt.Errorf("message target must be a pointer")
	}
	elemType := outType.Elem()
	expectedTypeID, ok := v2MsgTypeIDMap[elemType]
	if !ok {
		return fmt.Errorf("unknown v2 message type %s", elemType.String())
	}
	if typeID != expectedTypeID {
		actualType, ok := v2MsgReflectTypeMap[typeID]
		if !ok {
			return fmt.Errorf("unknown v2 message type %d", typeID)
		}
		return fmt.Errorf("unexpected message type %s, want %s", actualType.String(), elemType.String())
	}
	return json.Unmarshal(f.Payload[2:], out)
}

func EncodeV2MessageFrame(m Message) (*wire.Frame, error) {
	t := reflect.TypeOf(m)
	if t == nil {
		return nil, fmt.Errorf("nil message")
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	typeID, ok := v2MsgTypeIDMap[t]
	if !ok {
		return nil, fmt.Errorf("unknown v2 message type %s", t.String())
	}
	content, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	payload := make([]byte, 2+len(content))
	binary.BigEndian.PutUint16(payload[:2], typeID)
	copy(payload[2:], content)
	return &wire.Frame{
		Type:    wire.FrameTypeMessage,
		Payload: payload,
	}, nil
}

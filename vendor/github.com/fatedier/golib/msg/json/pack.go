// Copyright 2018 fatedier, fatedier@gmail.com
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

package json

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
)

func (msgCtl *MsgCtl) unpack(typeByte byte, buffer []byte, msgIn Message) (msg Message, err error) {
	if msgIn == nil {
		t, ok := msgCtl.typeMap[typeByte]
		if !ok {
			err = ErrMsgType
			return
		}

		msg = reflect.New(t).Interface().(Message)
	} else {
		msg = msgIn
	}

	err = json.Unmarshal(buffer, &msg)
	return
}

func (msgCtl *MsgCtl) UnPackInto(buffer []byte, msg Message) (err error) {
	_, err = msgCtl.unpack(' ', buffer, msg)
	return
}

func (msgCtl *MsgCtl) UnPack(typeByte byte, buffer []byte) (msg Message, err error) {
	return msgCtl.unpack(typeByte, buffer, nil)
}

func (msgCtl *MsgCtl) Pack(msg Message) ([]byte, error) {
	typeByte, ok := msgCtl.typeByteMap[reflect.TypeOf(msg).Elem()]
	if !ok {
		return nil, ErrMsgType
	}

	content, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBuffer(nil)
	buffer.WriteByte(typeByte)
	binary.Write(buffer, binary.BigEndian, int64(len(content)))
	buffer.Write(content)
	return buffer.Bytes(), nil
}

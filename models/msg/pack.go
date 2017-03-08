// Copyright 2016 fatedier, fatedier@gmail.com
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
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/fatedier/frp/utils/errors"
)

func unpack(typeByte byte, buffer []byte, msgIn Message) (msg Message, err error) {
	if msgIn == nil {
		t, ok := TypeMap[typeByte]
		if !ok {
			err = fmt.Errorf("Unsupported message type %b", typeByte)
			return
		}

		msg = reflect.New(t).Interface().(Message)
	} else {
		msg = msgIn
	}

	err = json.Unmarshal(buffer, &msg)
	return
}

func UnPackInto(buffer []byte, msg Message) (err error) {
	_, err = unpack(' ', buffer, msg)
	return
}

func UnPack(typeByte byte, buffer []byte) (msg Message, err error) {
	return unpack(typeByte, buffer, nil)
}

func Pack(msg Message) ([]byte, error) {
	typeByte, ok := TypeStringMap[reflect.TypeOf(msg).Elem()]
	if !ok {
		return nil, errors.ErrMsgType
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

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
	"encoding/binary"
	"fmt"
	"io"
)

var (
	MaxMsgLength int64 = 10240
)

func readMsg(c io.Reader) (typeByte byte, buffer []byte, err error) {
	buffer = make([]byte, 1)
	_, err = c.Read(buffer)
	if err != nil {
		return
	}
	typeByte = buffer[0]
	if _, ok := TypeMap[typeByte]; !ok {
		err = fmt.Errorf("Message type error")
		return
	}

	var length int64
	err = binary.Read(c, binary.BigEndian, &length)
	if err != nil {
		return
	}
	if length > MaxMsgLength {
		err = fmt.Errorf("Message length exceed the limit")
		return
	}

	buffer = make([]byte, length)
	n, err := io.ReadFull(c, buffer)
	if err != nil {
		return
	}

	if int64(n) != length {
		err = fmt.Errorf("Message format error")
	}
	return
}

func ReadMsg(c io.Reader) (msg Message, err error) {
	typeByte, buffer, err := readMsg(c)
	if err != nil {
		return
	}
	return UnPack(typeByte, buffer)
}

func ReadMsgInto(c io.Reader, msg Message) (err error) {
	_, buffer, err := readMsg(c)
	if err != nil {
		return
	}
	return UnPackInto(buffer, msg)
}

func WriteMsg(c io.Writer, msg interface{}) (err error) {
	buffer, err := Pack(msg)
	if err != nil {
		return
	}

	if _, err = c.Write(buffer); err != nil {
		return
	}

	return nil
}

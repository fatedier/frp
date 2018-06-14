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
	"encoding/binary"
	"errors"
	"io"
)

var (
	ErrMsgType      = errors.New("message type error")
	ErrMaxMsgLength = errors.New("message length exceed the limit")
	ErrMsgLength    = errors.New("message length error")
	ErrMsgFormat    = errors.New("message format error")
)

func (msgCtl *MsgCtl) readMsg(c io.Reader) (typeByte byte, buffer []byte, err error) {
	buffer = make([]byte, 1)
	_, err = c.Read(buffer)
	if err != nil {
		return
	}
	typeByte = buffer[0]
	if _, ok := msgCtl.typeMap[typeByte]; !ok {
		err = ErrMsgType
		return
	}

	var length int64
	err = binary.Read(c, binary.BigEndian, &length)
	if err != nil {
		return
	}
	if length > msgCtl.maxMsgLength {
		err = ErrMaxMsgLength
		return
	} else if length < 0 {
		err = ErrMsgLength
		return
	}

	buffer = make([]byte, length)
	n, err := io.ReadFull(c, buffer)
	if err != nil {
		return
	}

	if int64(n) != length {
		err = ErrMsgFormat
	}
	return
}

func (msgCtl *MsgCtl) ReadMsg(c io.Reader) (msg Message, err error) {
	typeByte, buffer, err := msgCtl.readMsg(c)
	if err != nil {
		return
	}
	return msgCtl.UnPack(typeByte, buffer)
}

func (msgCtl *MsgCtl) ReadMsgInto(c io.Reader, msg Message) (err error) {
	_, buffer, err := msgCtl.readMsg(c)
	if err != nil {
		return
	}
	return msgCtl.UnPackInto(buffer, msg)
}

func (msgCtl *MsgCtl) WriteMsg(c io.Writer, msg interface{}) (err error) {
	buffer, err := msgCtl.Pack(msg)
	if err != nil {
		return
	}

	if _, err = c.Write(buffer); err != nil {
		return
	}
	return nil
}

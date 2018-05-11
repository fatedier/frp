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
	"reflect"
)

var (
	defaultMaxMsgLength int64 = 10240
)

// Message wraps socket packages for communicating between frpc and frps.
type Message interface{}

type MsgCtl struct {
	typeMap     map[byte]reflect.Type
	typeByteMap map[reflect.Type]byte

	maxMsgLength int64
}

func NewMsgCtl() *MsgCtl {
	return &MsgCtl{
		typeMap:      make(map[byte]reflect.Type),
		typeByteMap:  make(map[reflect.Type]byte),
		maxMsgLength: defaultMaxMsgLength,
	}
}

func (msgCtl *MsgCtl) RegisterMsg(typeByte byte, msg interface{}) {
	msgCtl.typeMap[typeByte] = reflect.TypeOf(msg)
	msgCtl.typeByteMap[reflect.TypeOf(msg)] = typeByte
}

func (msgCtl *MsgCtl) SetMaxMsgLength(length int64) {
	msgCtl.maxMsgLength = length
}

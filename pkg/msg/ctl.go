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

package msg

import (
	"io"

	jsonMsg "github.com/fatedier/golib/msg/json"
)

type Message = jsonMsg.Message

var (
	msgCtl    *jsonMsg.MsgCtl
	udpMsgCtl *jsonMsg.MsgCtl
)

func init() {
	msgCtl = newMsgCtl(0)
	udpMsgCtl = newMsgCtl(maxV1UDPPacketMessageSize)
}

func newMsgCtl(maxMessageLength int64) *jsonMsg.MsgCtl {
	ctl := jsonMsg.NewMsgCtl()
	if maxMessageLength > 0 {
		ctl.SetMaxMsgLength(maxMessageLength)
	}
	for typeByte, msg := range msgTypeMap {
		ctl.RegisterMsg(typeByte, msg)
	}
	return ctl
}

func ReadMsg(c io.Reader) (msg Message, err error) {
	return msgCtl.ReadMsg(c)
}

func ReadMsgInto(c io.Reader, msg Message) (err error) {
	return msgCtl.ReadMsgInto(c, msg)
}

func WriteMsg(c io.Writer, msg any) (err error) {
	return msgCtl.WriteMsg(c, msg)
}

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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcess(t *testing.T) {
	assert := assert.New(t)

	var (
		msg    Message
		resMsg Message
		err    error
	)
	// empty struct
	msg = &Ping{}
	buffer := bytes.NewBuffer(nil)
	err = WriteMsg(buffer, msg)
	assert.NoError(err)

	resMsg, err = ReadMsg(buffer)
	assert.NoError(err)
	assert.Equal(reflect.TypeOf(resMsg).Elem(), TypeMap[TypePing])

	// normal message
	msg = &StartWorkConn{
		ProxyName: "test",
	}
	buffer = bytes.NewBuffer(nil)
	err = WriteMsg(buffer, msg)
	assert.NoError(err)

	resMsg, err = ReadMsg(buffer)
	assert.NoError(err)
	assert.Equal(reflect.TypeOf(resMsg).Elem(), TypeMap[TypeStartWorkConn])

	startWorkConnMsg, ok := resMsg.(*StartWorkConn)
	assert.True(ok)
	assert.Equal("test", startWorkConnMsg.ProxyName)

	// ReadMsgInto correct
	msg = &Pong{}
	buffer = bytes.NewBuffer(nil)
	err = WriteMsg(buffer, msg)
	assert.NoError(err)

	err = ReadMsgInto(buffer, msg)
	assert.NoError(err)

	// ReadMsgInto error type
	content := []byte(`{"run_id": 123}`)
	buffer = bytes.NewBuffer(nil)
	buffer.WriteByte(TypeNewWorkConn)
	binary.Write(buffer, binary.BigEndian, int64(len(content)))
	buffer.Write(content)

	resMsg = &NewWorkConn{}
	err = ReadMsgInto(buffer, resMsg)
	assert.Error(err)

	// message format error
	buffer = bytes.NewBuffer([]byte("1234"))

	resMsg = &NewProxyResp{}
	err = ReadMsgInto(buffer, resMsg)
	assert.Error(err)

	// MaxLength, real message length is 2
	MaxMsgLength = 1
	msg = &Ping{}
	buffer = bytes.NewBuffer(nil)
	err = WriteMsg(buffer, msg)
	assert.NoError(err)

	_, err = ReadMsg(buffer)
	assert.Error(err)
	return
}

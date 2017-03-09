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

	"github.com/fatedier/frp/utils/errors"
)

type TestStruct struct{}

func TestPack(t *testing.T) {
	assert := assert.New(t)

	var (
		msg    Message
		buffer []byte
		err    error
	)

	// error type
	msg = &TestStruct{}
	buffer, err = Pack(msg)
	assert.Error(err, errors.ErrMsgType.Error())

	// correct
	msg = &Ping{}
	buffer, err = Pack(msg)
	assert.NoError(err)
	b := bytes.NewBuffer(nil)
	b.WriteByte(TypePing)
	binary.Write(b, binary.BigEndian, int64(2))
	b.WriteString("{}")
	assert.True(bytes.Equal(b.Bytes(), buffer))
}

func TestUnPack(t *testing.T) {
	assert := assert.New(t)

	var (
		msg Message
		err error
	)

	// error message type
	msg, err = UnPack('-', []byte("{}"))
	assert.Error(err)

	// correct
	msg, err = UnPack(TypePong, []byte("{}"))
	assert.NoError(err)
	assert.Equal(reflect.TypeOf(msg).Elem(), reflect.TypeOf(Pong{}))
}

func TestUnPackInto(t *testing.T) {
	assert := assert.New(t)

	var err error

	// correct type
	pongMsg := &Pong{}
	err = UnPackInto([]byte("{}"), pongMsg)
	assert.NoError(err)

	// wrong type
	loginMsg := &Login{}
	err = UnPackInto([]byte(`{"version": 123}`), loginMsg)
	assert.Error(err)
}

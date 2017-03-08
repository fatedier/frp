// Copyright 2017 fatedier, fatedier@gmail.com
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

package tcp

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fatedier/frp/utils/crypto"
)

func TestJoin(t *testing.T) {
	assert := assert.New(t)

	var (
		n   int
		err error
	)
	text1 := "A document that gives tips for writing clear, idiomatic Go code. A must read for any new Go programmer. It augments the tour and the language specification, both of which should be read first."
	text2 := "A document that specifies the conditions under which reads of a variable in one goroutine can be guaranteed to observe values produced by writes to the same variable in a different goroutine."

	// Forward bytes directly.
	pr, pw := io.Pipe()
	pr2, pw2 := io.Pipe()
	pr3, pw3 := io.Pipe()
	pr4, pw4 := io.Pipe()

	conn1 := WrapReadWriteCloser(pr, pw2)
	conn2 := WrapReadWriteCloser(pr2, pw)
	conn3 := WrapReadWriteCloser(pr3, pw4)
	conn4 := WrapReadWriteCloser(pr4, pw3)

	go func() {
		Join(conn2, conn3)
	}()

	buf1 := make([]byte, 1024)
	buf2 := make([]byte, 1024)

	conn1.Write([]byte(text1))
	conn4.Write([]byte(text2))

	n, err = conn4.Read(buf1)
	assert.NoError(err)
	assert.Equal(text1, string(buf1[:n]))

	n, err = conn1.Read(buf2)
	assert.NoError(err)
	assert.Equal(text2, string(buf2[:n]))

	conn1.Close()
	conn2.Close()
	conn3.Close()
	conn4.Close()
}

func TestJoinEncrypt(t *testing.T) {
	assert := assert.New(t)

	var (
		n   int
		err error
	)
	text1 := "1234567890"
	text2 := "abcdefghij"
	key := "authkey"

	// Forward enrypted bytes.
	pr, pw := io.Pipe()
	pr2, pw2 := io.Pipe()
	pr3, pw3 := io.Pipe()
	pr4, pw4 := io.Pipe()
	pr5, pw5 := io.Pipe()
	pr6, pw6 := io.Pipe()

	conn1 := WrapReadWriteCloser(pr, pw2)
	conn2 := WrapReadWriteCloser(pr2, pw)
	conn3 := WrapReadWriteCloser(pr3, pw4)
	conn4 := WrapReadWriteCloser(pr4, pw3)
	conn5 := WrapReadWriteCloser(pr5, pw6)
	conn6 := WrapReadWriteCloser(pr6, pw5)

	r1, err := crypto.NewReader(conn3, []byte(key))
	assert.NoError(err)
	w1, err := crypto.NewWriter(conn3, []byte(key))
	assert.NoError(err)

	r2, err := crypto.NewReader(conn4, []byte(key))
	assert.NoError(err)
	w2, err := crypto.NewWriter(conn4, []byte(key))
	assert.NoError(err)

	go Join(conn2, WrapReadWriteCloser(r1, w1))
	go Join(WrapReadWriteCloser(r2, w2), conn5)

	buf := make([]byte, 128)

	conn1.Write([]byte(text1))
	conn6.Write([]byte(text2))

	n, err = conn6.Read(buf)
	assert.NoError(err)
	assert.Equal(text1, string(buf[:n]))

	n, err = conn1.Read(buf)
	assert.NoError(err)
	assert.Equal(text2, string(buf[:n]))

	conn1.Close()
	conn2.Close()
	conn3.Close()
	conn4.Close()
	conn5.Close()
	conn6.Close()
}

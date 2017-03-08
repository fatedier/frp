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
)

func TestWithCompression(t *testing.T) {
	assert := assert.New(t)

	// Forward compression bytes.
	pr, pw := io.Pipe()
	pr2, pw2 := io.Pipe()

	conn1 := WrapReadWriteCloser(pr, pw2)
	conn2 := WrapReadWriteCloser(pr2, pw)

	compressionStream1 := WithCompression(conn1)
	compressionStream2 := WithCompression(conn2)

	var (
		n   int
		err error
	)

	text := "1234567812345678"
	buf := make([]byte, 256)

	go compressionStream1.Write([]byte(text))
	n, err = compressionStream2.Read(buf)
	assert.NoError(err)
	assert.Equal(text, string(buf[:n]))

	go compressionStream2.Write([]byte(text))
	n, err = compressionStream1.Read(buf)
	assert.NoError(err)
	assert.Equal(text, string(buf[:n]))
}

func TestWithEncryption(t *testing.T) {
	assert := assert.New(t)
	var (
		n   int
		err error
	)
	text1 := "Go is expressive, concise, clean, and efficient. Its concurrency mechanisms make it easy to write programs that get the most out of multicore and networked machines, while its novel type system enables flexible and modular program construction. Go compiles quickly to machine code yet has the convenience of garbage collection and the power of run-time reflection. It's a fast, statically typed, compiled language that feels like a dynamically typed, interpreted language."
	text2 := "An interactive introduction to Go in three sections. The first section covers basic syntax and data structures; the second discusses methods and interfaces; and the third introduces Go's concurrency primitives. Each section concludes with a few exercises so you can practice what you've learned. You can take the tour online or install it locally with"
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

	encryptStream1, err := WithEncryption(conn3, []byte(key))
	assert.NoError(err)
	encryptStream2, err := WithEncryption(conn4, []byte(key))
	assert.NoError(err)

	go Join(conn2, encryptStream1)
	go Join(encryptStream2, conn5)

	buf := make([]byte, 1024)

	conn1.Write([]byte(text1))
	conn6.Write([]byte(text2))

	n, err = conn6.Read(buf)
	assert.NoError(err)
	assert.Equal(text1, string(buf[:n]))

	n, err = conn1.Read(buf)
	assert.NoError(err)
}

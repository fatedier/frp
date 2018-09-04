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

package net

import (
	"bytes"
	"io"
	"net"
)

type SharedConn struct {
	net.Conn
	buf *bytes.Buffer
}

// the bytes you read in io.Reader, will be reserved in SharedConn
func NewSharedConn(conn net.Conn) (*SharedConn, io.Reader) {
	sc := &SharedConn{
		Conn: conn,
		buf:  bytes.NewBuffer(make([]byte, 0, 1024)),
	}
	return sc, io.TeeReader(conn, sc.buf)
}

func NewSharedConnSize(conn net.Conn, bufSize int) (*SharedConn, io.Reader) {
	sc := &SharedConn{
		Conn: conn,
		buf:  bytes.NewBuffer(make([]byte, 0, bufSize)),
	}
	return sc, io.TeeReader(conn, sc.buf)
}

// Not thread safety.
func (sc *SharedConn) Read(p []byte) (n int, err error) {
	if sc.buf == nil {
		return sc.Conn.Read(p)
	}
	n, err = sc.buf.Read(p)
	if err == io.EOF {
		sc.buf = nil
		var n2 int
		n2, err = sc.Conn.Read(p[n:])
		n += n2
	}
	return
}

func (sc *SharedConn) ResetBuf(buffer []byte) (err error) {
	sc.buf.Reset()
	_, err = sc.buf.Write(buffer)
	return err
}

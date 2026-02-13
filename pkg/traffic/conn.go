// Copyright 2024 The frp Authors
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

package traffic

import (
	"io"
	"net"
)

// CountedConn wraps a net.Conn to count traffic.
type CountedConn struct {
	net.Conn
	counter   *TokenTrafficCounter
	proxyName string
}

// NewCountedConn creates a new counted connection.
func NewCountedConn(conn net.Conn, counter *TokenTrafficCounter, proxyName string) *CountedConn {
	return &CountedConn{
		Conn:      conn,
		counter:   counter,
		proxyName: proxyName,
	}
}

func (c *CountedConn) Read(p []byte) (n int, err error) {
	n, err = c.Conn.Read(p)
	if n > 0 && c.counter != nil {
		c.counter.AddTraffic(c.proxyName, int64(n), 0)
	}
	return
}

func (c *CountedConn) Write(p []byte) (n int, err error) {
	n, err = c.Conn.Write(p)
	if n > 0 && c.counter != nil {
		c.counter.AddTraffic(c.proxyName, 0, int64(n))
	}
	return
}

// Close closes the connection and flushes traffic stats.
func (c *CountedConn) Close() error {
	if c.counter != nil {
		c.counter.Flush(c.proxyName)
	}
	return c.Conn.Close()
}

// CountedReadWriteCloser wraps an io.ReadWriteCloser to count traffic.
type CountedReadWriteCloser struct {
	io.ReadWriteCloser
	counter   *TokenTrafficCounter
	proxyName string
}

// NewCountedReadWriteCloser creates a new counted ReadWriteCloser.
func NewCountedReadWriteCloser(rwc io.ReadWriteCloser, counter *TokenTrafficCounter, proxyName string) *CountedReadWriteCloser {
	return &CountedReadWriteCloser{
		ReadWriteCloser: rwc,
		counter:         counter,
		proxyName:       proxyName,
	}
}

func (c *CountedReadWriteCloser) Read(p []byte) (n int, err error) {
	n, err = c.ReadWriteCloser.Read(p)
	if n > 0 && c.counter != nil {
		c.counter.AddTraffic(c.proxyName, int64(n), 0)
	}
	return
}

func (c *CountedReadWriteCloser) Write(p []byte) (n int, err error) {
	n, err = c.ReadWriteCloser.Write(p)
	if n > 0 && c.counter != nil {
		c.counter.AddTraffic(c.proxyName, 0, int64(n))
	}
	return
}

func (c *CountedReadWriteCloser) Close() error {
	if c.counter != nil {
		c.counter.Flush(c.proxyName)
	}
	return c.ReadWriteCloser.Close()
}

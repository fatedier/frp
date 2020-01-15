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

package net

import (
	"fmt"
	"net"
	"sync"

	"github.com/fatedier/golib/errors"
)

// Custom listener
type CustomListener struct {
	acceptCh chan net.Conn
	closed   bool
	mu       sync.Mutex
}

func NewCustomListener() *CustomListener {
	return &CustomListener{
		acceptCh: make(chan net.Conn, 64),
	}
}

func (l *CustomListener) Accept() (net.Conn, error) {
	conn, ok := <-l.acceptCh
	if !ok {
		return nil, fmt.Errorf("listener closed")
	}
	return conn, nil
}

func (l *CustomListener) PutConn(conn net.Conn) error {
	err := errors.PanicToError(func() {
		select {
		case l.acceptCh <- conn:
		default:
			conn.Close()
		}
	})
	return err
}

func (l *CustomListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		close(l.acceptCh)
		l.closed = true
	}
	return nil
}

func (l *CustomListener) Addr() net.Addr {
	return (*net.TCPAddr)(nil)
}

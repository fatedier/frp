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

// InternalListener is a listener that can be used to accept connections from
// other goroutines.
type InternalListener struct {
	acceptCh chan net.Conn
	closed   bool
	mu       sync.Mutex
}

func NewInternalListener() *InternalListener {
	return &InternalListener{
		acceptCh: make(chan net.Conn, 128),
	}
}

func (l *InternalListener) Accept() (net.Conn, error) {
	conn, ok := <-l.acceptCh
	if !ok {
		return nil, fmt.Errorf("listener closed")
	}
	return conn, nil
}

func (l *InternalListener) PutConn(conn net.Conn) error {
	err := errors.PanicToError(func() {
		select {
		case l.acceptCh <- conn:
		default:
			conn.Close()
		}
	})
	if err != nil {
		return fmt.Errorf("put conn error: listener is closed")
	}
	return nil
}

func (l *InternalListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		close(l.acceptCh)
		l.closed = true
	}
	return nil
}

func (l *InternalListener) Addr() net.Addr {
	return &InternalAddr{}
}

type InternalAddr struct{}

func (ia *InternalAddr) Network() string {
	return "internal"
}

func (ia *InternalAddr) String() string {
	return "internal"
}

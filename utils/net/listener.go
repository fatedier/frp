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

	"github.com/fatedier/frp/utils/log"

	"github.com/fatedier/golib/errors"
)

type Listener interface {
	Accept() (Conn, error)
	Close() error
	log.Logger
}

type LogListener struct {
	l net.Listener
	net.Listener
	log.Logger
}

func WrapLogListener(l net.Listener) Listener {
	return &LogListener{
		l:        l,
		Listener: l,
		Logger:   log.NewPrefixLogger(""),
	}
}

func (logL *LogListener) Accept() (Conn, error) {
	c, err := logL.l.Accept()
	return WrapConn(c), err
}

// Custom listener
type CustomListener struct {
	conns  chan Conn
	closed bool
	mu     sync.Mutex

	log.Logger
}

func NewCustomListener() *CustomListener {
	return &CustomListener{
		conns:  make(chan Conn, 64),
		Logger: log.NewPrefixLogger(""),
	}
}

func (l *CustomListener) Accept() (Conn, error) {
	conn, ok := <-l.conns
	if !ok {
		return nil, fmt.Errorf("listener closed")
	}
	conn.AddLogPrefix(l.GetPrefixStr())
	return conn, nil
}

func (l *CustomListener) PutConn(conn Conn) error {
	err := errors.PanicToError(func() {
		select {
		case l.conns <- conn:
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
		close(l.conns)
		l.closed = true
	}
	return nil
}

func (l *CustomListener) Addr() net.Addr {
	return (*net.TCPAddr)(nil)
}

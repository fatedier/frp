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

package plugin

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/fatedier/golib/errors"
)

// Creators is used for create plugins to handle connections.
var creators = make(map[string]CreatorFn)

// params has prefix "plugin_"
type CreatorFn func(params map[string]string) (Plugin, error)

func Register(name string, fn CreatorFn) {
	creators[name] = fn
}

func Create(name string, params map[string]string) (p Plugin, err error) {
	if fn, ok := creators[name]; ok {
		p, err = fn(params)
	} else {
		err = fmt.Errorf("plugin [%s] is not registered", name)
	}
	return
}

type Plugin interface {
	Name() string

	// extraBufToLocal will send to local connection first, then join conn with local connection
	Handle(conn io.ReadWriteCloser, realConn net.Conn, extraBufToLocal []byte)
	Close() error
}

type Listener struct {
	conns  chan net.Conn
	closed bool
	mu     sync.Mutex
}

func NewProxyListener() *Listener {
	return &Listener{
		conns: make(chan net.Conn, 64),
	}
}

func (l *Listener) Accept() (net.Conn, error) {
	conn, ok := <-l.conns
	if !ok {
		return nil, fmt.Errorf("listener closed")
	}
	return conn, nil
}

func (l *Listener) PutConn(conn net.Conn) error {
	err := errors.PanicToError(func() {
		l.conns <- conn
	})
	return err
}

func (l *Listener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		close(l.conns)
		l.closed = true
	}
	return nil
}

func (l *Listener) Addr() net.Addr {
	return (*net.TCPAddr)(nil)
}

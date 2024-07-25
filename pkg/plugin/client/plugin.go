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
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/fatedier/golib/errors"
	pp "github.com/pires/go-proxyproto"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// Creators is used for create plugins to handle connections.
var creators = make(map[string]CreatorFn)

// params has prefix "plugin_"
type CreatorFn func(options v1.ClientPluginOptions) (Plugin, error)

func Register(name string, fn CreatorFn) {
	if _, exist := creators[name]; exist {
		panic(fmt.Sprintf("plugin [%s] is already registered", name))
	}
	creators[name] = fn
}

func Create(name string, options v1.ClientPluginOptions) (p Plugin, err error) {
	if fn, ok := creators[name]; ok {
		p, err = fn(options)
	} else {
		err = fmt.Errorf("plugin [%s] is not registered", name)
	}
	return
}

type ExtraInfo struct {
	ProxyProtocolHeader *pp.Header
	SrcAddr             net.Addr
	DstAddr             net.Addr
}

type Plugin interface {
	Name() string

	Handle(ctx context.Context, conn io.ReadWriteCloser, realConn net.Conn, extra *ExtraInfo)
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

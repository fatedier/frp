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

package vhost

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/fatedier/frp/src/utils/conn"
)

type muxFunc func(*conn.Conn) (net.Conn, string, error)
type hostRewriteFunc func(*conn.Conn, string) (net.Conn, error)

type VhostMuxer struct {
	listener    *conn.Listener
	timeout     time.Duration
	vhostFunc   muxFunc
	rewriteFunc hostRewriteFunc
	registryMap map[string]*Listener
	mutex       sync.RWMutex
}

func NewVhostMuxer(listener *conn.Listener, vhostFunc muxFunc, rewriteFunc hostRewriteFunc, timeout time.Duration) (mux *VhostMuxer, err error) {
	mux = &VhostMuxer{
		listener:    listener,
		timeout:     timeout,
		vhostFunc:   vhostFunc,
		rewriteFunc: rewriteFunc,
		registryMap: make(map[string]*Listener),
	}
	go mux.run()
	return mux, nil
}

// listen for a new domain name, if rewriteHost is not empty  and rewriteFunc is not nil, then rewrite the host header to rewriteHost
func (v *VhostMuxer) Listen(name string, rewriteHost string) (l *Listener, err error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	if _, exist := v.registryMap[name]; exist {
		return nil, fmt.Errorf("domain name %s is already bound", name)
	}

	l = &Listener{
		name:        name,
		rewriteHost: rewriteHost,
		mux:         v,
		accept:      make(chan *conn.Conn),
	}
	v.registryMap[name] = l
	return l, nil
}

func (v *VhostMuxer) getListener(name string) (l *Listener, exist bool) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	// first we check the full hostname
	// if not exist, then check the wildcard_domain such as *.example.com
	l, exist = v.registryMap[name]
	if exist {
		return l, exist
	}
	domainSplit := strings.Split(name, ".")
	if len(domainSplit) < 3 {
		return l, false
	}
	domainSplit[0] = "*"
	name = strings.Join(domainSplit, ".")
	l, exist = v.registryMap[name]
	return l, exist
}

func (v *VhostMuxer) unRegister(name string) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	delete(v.registryMap, name)
}

func (v *VhostMuxer) run() {
	for {
		conn, err := v.listener.Accept()
		if err != nil {
			return
		}
		go v.handle(conn)
	}
}

func (v *VhostMuxer) handle(c *conn.Conn) {
	if err := c.SetDeadline(time.Now().Add(v.timeout)); err != nil {
		c.Close()
		return
	}

	sConn, name, err := v.vhostFunc(c)
	if err != nil {
		c.Close()
		return
	}

	name = strings.ToLower(name)
	// get listener by hostname
	l, ok := v.getListener(name)
	if !ok {
		c.Close()
		return
	}

	if err = sConn.SetDeadline(time.Time{}); err != nil {
		c.Close()
		return
	}
	c.SetTcpConn(sConn)

	l.accept <- c
}

type Listener struct {
	name        string
	rewriteHost string
	mux         *VhostMuxer // for closing VhostMuxer
	accept      chan *conn.Conn
}

func (l *Listener) Accept() (*conn.Conn, error) {
	conn, ok := <-l.accept
	if !ok {
		return nil, fmt.Errorf("Listener closed")
	}

	// if rewriteFunc is exist and rewriteHost is set
	// rewrite http requests with a modified host header
	if l.mux.rewriteFunc != nil && l.rewriteHost != "" {
		sConn, err := l.mux.rewriteFunc(conn, l.rewriteHost)
		if err != nil {
			return nil, fmt.Errorf("http host header rewrite failed")
		}
		conn.SetTcpConn(sConn)
	}
	return conn, nil
}

func (l *Listener) Close() error {
	l.mux.unRegister(l.name)
	close(l.accept)
	return nil
}

func (l *Listener) Name() string {
	return l.name
}

type sharedConn struct {
	net.Conn
	sync.Mutex
	buff *bytes.Buffer
}

// the bytes you read in io.Reader, will be reserved in sharedConn
func newShareConn(conn net.Conn) (*sharedConn, io.Reader) {
	sc := &sharedConn{
		Conn: conn,
		buff: bytes.NewBuffer(make([]byte, 0, 1024)),
	}
	return sc, io.TeeReader(conn, sc.buff)
}

func (sc *sharedConn) Read(p []byte) (n int, err error) {
	sc.Lock()
	if sc.buff == nil {
		sc.Unlock()
		return sc.Conn.Read(p)
	}
	n, err = sc.buff.Read(p)

	if err == io.EOF {
		sc.buff = nil
		var n2 int
		n2, err = sc.Conn.Read(p[n:])

		n += n2
	}
	sc.Unlock()
	return
}

func (sc *sharedConn) WriteBuff(buffer []byte) (err error) {
	sc.buff.Reset()
	_, err = sc.buff.Write(buffer)
	return err
}

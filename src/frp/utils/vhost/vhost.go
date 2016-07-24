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

	"frp/utils/conn"
)

type muxFunc func(*conn.Conn) (net.Conn, string, error)

type VhostMuxer struct {
	listener    *conn.Listener
	timeout     time.Duration
	vhostFunc   muxFunc
	registryMap map[string]*Listener
	mutex       sync.RWMutex
}

func NewVhostMuxer(listener *conn.Listener, vhostFunc muxFunc, timeout time.Duration) (mux *VhostMuxer, err error) {
	mux = &VhostMuxer{
		listener:    listener,
		timeout:     timeout,
		vhostFunc:   vhostFunc,
		registryMap: make(map[string]*Listener),
	}
	go mux.run()
	return mux, nil
}

func (v *VhostMuxer) Listen(name, proxytype, clientIp string, clientPort, serverPort int64) (l *Listener, err error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	if _, exist := v.registryMap[name]; exist {
		return nil, fmt.Errorf("name %s is already bound", name)
	}

	l = &Listener{
		name:       name,
		mux:        v,
		accept:     make(chan *conn.Conn),
		proxyType:  proxytype,
		clientIp:   clientIp,
		clientPort: clientPort,
		serverPort: serverPort,
	}
	v.registryMap[name] = l
	return l, nil
}

func (v *VhostMuxer) getListener(name string) (l *Listener, exist bool) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
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
		return
	}

	sConn, name, err := v.vhostFunc(c)
	if err != nil {
		return
	}

	name = strings.ToLower(name)
	l, ok := v.getListener(name)
	if !ok {
		return
	}

	if err = sConn.SetDeadline(time.Time{}); err != nil {
		return
	}
	c.SetTcpConn(sConn)

	l.accept <- c
}

type Listener struct {
	name       string
	mux        *VhostMuxer // for closing VhostMuxer
	accept     chan *conn.Conn
	proxyType  string //suppor http host rewrite
	clientIp   string
	clientPort int64
	serverPort int64
}

func (l *Listener) Accept() (*conn.Conn, error) {
	conn, ok := <-l.accept
	if !ok {
		return nil, fmt.Errorf("Listener closed")
	}
	if net.ParseIP(l.clientIp) == nil && l.proxyType == "http" {
		if (l.name != l.clientIp) || (l.serverPort != l.clientPort) {
			clientHost := l.clientIp
			if l.clientPort != 80 {
				strPort := fmt.Sprintf(":%d", l.clientPort)
				clientHost += strPort
			}
			retConn, err := HostNameRewrite(conn, clientHost)
			if err != nil {
				return nil, fmt.Errorf("http host rewrite failed")
			}
			conn.SetTcpConn(retConn)
		}
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

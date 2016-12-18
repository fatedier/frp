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
	//"strings"
	"sync"
	"time"

	"github.com/fatedier/frp/src/utils/conn"
	"github.com/fatedier/frp/src/utils/log"
)

type muxFunc func(*conn.Conn) (net.Conn, string, error)
type hostRewriteFunc func(*conn.Conn, string) (net.Conn, error)

type VhostMuxer struct {
	listener       *conn.Listener
	timeout        time.Duration
	vhostFunc      muxFunc
	rewriteFunc    hostRewriteFunc
	registryRouter *VhostRouters
	mutex          sync.RWMutex
}

func NewVhostMuxer(listener *conn.Listener, vhostFunc muxFunc, rewriteFunc hostRewriteFunc, timeout time.Duration) (mux *VhostMuxer, err error) {
	mux = &VhostMuxer{
		listener:       listener,
		timeout:        timeout,
		vhostFunc:      vhostFunc,
		rewriteFunc:    rewriteFunc,
		registryRouter: NewVhostRouters(),
	}
	go mux.run()
	return mux, nil
}

// listen for a new domain name, if rewriteHost is not empty  and rewriteFunc is not nil, then rewrite the host header to rewriteHost
func (v *VhostMuxer) Listen(name, domain string, rewriteHost string) (l *Listener) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	locations := []string{""}

	l = &Listener{
		name:        name,
		domain:      domain,
		locations:   locations,
		rewriteHost: rewriteHost,
		mux:         v,
		accept:      make(chan *conn.Conn),
	}

	v.registryRouter.add(name, domain, locations, l)
	return l
}

func (v *VhostMuxer) ListenByRouter(name string, domains []string, locations []string, rewriteHost string) (ls []*Listener) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	ls = make([]*Listener, 0)
	for _, domain := range domains {
		l := &Listener{
			name:        name,
			domain:      domain,
			locations:   locations,
			rewriteHost: rewriteHost,
			mux:         v,
			accept:      make(chan *conn.Conn),
		}
		v.registryRouter.add(name, domain, locations, l)
		ls = append(ls, l)
	}

	return ls
}

func (v *VhostMuxer) getListener(rname string) (l *Listener, exist bool) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	var frcname string
	vr, found := v.registryRouter.get(rname)
	if found {
		frcname = vr.name
	} else {
		log.Warn("can't found the router for %s", rname)
		return
	}

	log.Debug("get frcname %s for %s", frcname, rname)
	return vr.listener, true
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

	//name = strings.ToLower(name)
	l, ok := v.getListener(name)
	if !ok {
		return
	}

	if err = sConn.SetDeadline(time.Time{}); err != nil {
		log.Error("set dead line err: %v", err)
		return
	}
	c.SetTcpConn(sConn)

	log.Debug("handle request: %s", c.GetRemoteAddr())
	l.accept <- c
}

type Listener struct {
	name        string
	domain      string
	locations   []string
	rewriteHost string
	mux         *VhostMuxer // for closing VhostMuxer
	accept      chan *conn.Conn
}

func (l *Listener) Accept() (*conn.Conn, error) {
	log.Debug("[%s][%s] now to accept ...", l.name, l.domain)
	conn, ok := <-l.accept
	if !ok {
		return nil, fmt.Errorf("Listener closed")
	}
	log.Debug("[%s][%s] accept something ...", l.name, l.domain)

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
	l.mux.registryRouter.del(l)
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

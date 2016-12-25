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

type muxFunc func(*conn.Conn) (net.Conn, map[string]string, error)
type httpAuthFunc func(*conn.Conn, string, string, string) (bool, error)
type hostRewriteFunc func(*conn.Conn, string) (net.Conn, error)

type VhostMuxer struct {
	listener       *conn.Listener
	timeout        time.Duration
	vhostFunc      muxFunc
	authFunc       httpAuthFunc
	rewriteFunc    hostRewriteFunc
	registryRouter *VhostRouters
	mutex          sync.RWMutex
}

func NewVhostMuxer(listener *conn.Listener, vhostFunc muxFunc, authFunc httpAuthFunc, rewriteFunc hostRewriteFunc, timeout time.Duration) (mux *VhostMuxer, err error) {
	mux = &VhostMuxer{
		listener:       listener,
		timeout:        timeout,
		vhostFunc:      vhostFunc,
		authFunc:       authFunc,
		rewriteFunc:    rewriteFunc,
		registryRouter: NewVhostRouters(),
	}
	go mux.run()
	return mux, nil
}

// listen for a new domain name, if rewriteHost is not empty  and rewriteFunc is not nil, then rewrite the host header to rewriteHost
func (v *VhostMuxer) Listen(name, location, rewriteHost, userName, passWord string) (l *Listener, err error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	l = &Listener{
		name:        name,
		rewriteHost: rewriteHost,
		userName:    userName,
		passWord:    passWord,
		mux:         v,
		accept:      make(chan *conn.Conn),
	}
	v.registryRouter.Add(name, location, l)
	return l, nil
}

func (v *VhostMuxer) getListener(name, path string) (l *Listener, exist bool) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	// first we check the full hostname
	// if not exist, then check the wildcard_domain such as *.example.com
	vr, found := v.registryRouter.Get(name, path)
	if found {
		return vr.listener, true
	}

	domainSplit := strings.Split(name, ".")
	if len(domainSplit) < 3 {
		return l, false
	}
	domainSplit[0] = "*"
	name = strings.Join(domainSplit, ".")

	vr, found = v.registryRouter.Get(name, path)
	if !found {
		return
	}

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
		c.Close()
		return
	}

	sConn, reqInfoMap, err := v.vhostFunc(c)
	if err != nil {
		c.Close()
		return
	}

	name := strings.ToLower(reqInfoMap["Host"])
	path := strings.ToLower(reqInfoMap["Path"])
	l, ok := v.getListener(name, path)
	if !ok {
		c.Close()
		return
	}

	// if authFunc is exist and userName/password is set
	// verify user access
	if l.mux.authFunc != nil &&
		l.userName != "" && l.passWord != "" {
		bAccess, err := l.mux.authFunc(c, l.userName, l.passWord, reqInfoMap["Authorization"])
		if bAccess == false || err != nil {
			res := noAuthResponse()
			res.Write(c.TcpConn)
			c.Close()
			return
		}
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
	userName    string
	passWord    string
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
	l.mux.registryRouter.Del(l)
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
	sc.Unlock()
	n, err = sc.buff.Read(p)

	if err == io.EOF {
		sc.Lock()
		sc.buff = nil
		sc.Unlock()
		var n2 int
		n2, err = sc.Conn.Read(p[n:])

		n += n2
	}
	return
}

func (sc *sharedConn) WriteBuff(buffer []byte) (err error) {
	sc.buff.Reset()
	_, err = sc.buff.Write(buffer)
	return err
}

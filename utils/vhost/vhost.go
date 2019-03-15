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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"

	"github.com/fatedier/golib/errors"
)

type muxFunc func(frpNet.Conn) (frpNet.Conn, map[string]string, error)
type httpAuthFunc func(frpNet.Conn, string, string, string) (bool, error)
type hostRewriteFunc func(frpNet.Conn, string) (frpNet.Conn, error)

type VhostMuxer struct {
	listener       frpNet.Listener
	timeout        time.Duration
	vhostFunc      muxFunc
	authFunc       httpAuthFunc
	rewriteFunc    hostRewriteFunc
	registryRouter *VhostRouters
	mutex          sync.RWMutex
}

func NewVhostMuxer(listener frpNet.Listener, vhostFunc muxFunc, authFunc httpAuthFunc, rewriteFunc hostRewriteFunc, timeout time.Duration) (mux *VhostMuxer, err error) {
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

type CreateConnFunc func() (frpNet.Conn, error)

type VhostRouteConfig struct {
	Domain      string
	Location    string
	RewriteHost string
	Username    string
	Password    string
	Headers     map[string]string

	CreateConnFn CreateConnFunc
}

// listen for a new domain name, if rewriteHost is not empty  and rewriteFunc is not nil
// then rewrite the host header to rewriteHost
func (v *VhostMuxer) Listen(cfg *VhostRouteConfig) (l *Listener, err error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	_, ok := v.registryRouter.Exist(cfg.Domain, cfg.Location)
	if ok {
		return nil, fmt.Errorf("hostname [%s] location [%s] is already registered", cfg.Domain, cfg.Location)
	}

	l = &Listener{
		name:        cfg.Domain,
		location:    cfg.Location,
		rewriteHost: cfg.RewriteHost,
		userName:    cfg.Username,
		passWord:    cfg.Password,
		mux:         v,
		accept:      make(chan frpNet.Conn),
		Logger:      log.NewPrefixLogger(""),
	}
	v.registryRouter.Add(cfg.Domain, cfg.Location, l)
	return l, nil
}

func (v *VhostMuxer) getListener(name, path string) (l *Listener, exist bool) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	// first we check the full hostname
	// if not exist, then check the wildcard_domain such as *.example.com
	vr, found := v.registryRouter.Get(name, path)
	if found {
		return vr.payload.(*Listener), true
	}

	domainSplit := strings.Split(name, ".")
	if len(domainSplit) < 3 {
		return
	}

	for {
		if len(domainSplit) < 3 {
			return
		}

		domainSplit[0] = "*"
		name = strings.Join(domainSplit, ".")

		vr, found = v.registryRouter.Get(name, path)
		if found {
			return vr.payload.(*Listener), true
		}
		domainSplit = domainSplit[1:]
	}
	return
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

func (v *VhostMuxer) handle(c frpNet.Conn) {
	if err := c.SetDeadline(time.Now().Add(v.timeout)); err != nil {
		c.Close()
		return
	}

	sConn, reqInfoMap, err := v.vhostFunc(c)
	if err != nil {
		log.Warn("get hostname from http/https request error: %v", err)
		c.Close()
		return
	}

	name := strings.ToLower(reqInfoMap["Host"])
	path := strings.ToLower(reqInfoMap["Path"])
	l, ok := v.getListener(name, path)
	if !ok {
		res := notFoundResponse()
		res.Write(c)
		log.Debug("http request for host [%s] path [%s] not found", name, path)
		c.Close()
		return
	}

	// if authFunc is exist and userName/password is set
	// then verify user access
	if l.mux.authFunc != nil && l.userName != "" && l.passWord != "" {
		bAccess, err := l.mux.authFunc(c, l.userName, l.passWord, reqInfoMap["Authorization"])
		if bAccess == false || err != nil {
			l.Debug("check http Authorization failed")
			res := noAuthResponse()
			res.Write(c)
			c.Close()
			return
		}
	}

	if err = sConn.SetDeadline(time.Time{}); err != nil {
		c.Close()
		return
	}
	c = sConn

	l.Debug("get new http request host [%s] path [%s]", name, path)
	err = errors.PanicToError(func() {
		l.accept <- c
	})
	if err != nil {
		l.Warn("listener is already closed, ignore this request")
	}
}

type Listener struct {
	name        string
	location    string
	rewriteHost string
	userName    string
	passWord    string
	mux         *VhostMuxer // for closing VhostMuxer
	accept      chan frpNet.Conn
	log.Logger
}

func (l *Listener) Accept() (frpNet.Conn, error) {
	conn, ok := <-l.accept
	if !ok {
		return nil, fmt.Errorf("Listener closed")
	}

	// if rewriteFunc is exist
	// rewrite http requests with a modified host header
	// if l.rewriteHost is empty, nothing to do
	if l.mux.rewriteFunc != nil {
		sConn, err := l.mux.rewriteFunc(conn, l.rewriteHost)
		if err != nil {
			l.Warn("host header rewrite failed: %v", err)
			return nil, fmt.Errorf("host header rewrite failed")
		}
		l.Debug("rewrite host to [%s] success", l.rewriteHost)
		conn = sConn
	}

	for _, prefix := range l.GetAllPrefix() {
		conn.AddLogPrefix(prefix)
	}
	return conn, nil
}

func (l *Listener) Close() error {
	l.mux.registryRouter.Del(l.name, l.location)
	close(l.accept)
	return nil
}

func (l *Listener) Name() string {
	return l.name
}

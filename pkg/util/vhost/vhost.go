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
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/fatedier/frp/pkg/util/log"
	frpNet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"

	"github.com/fatedier/golib/errors"
)

type RouteInfo string

const (
	RouteInfoURL    RouteInfo = "url"
	RouteInfoHost   RouteInfo = "host"
	RouteInfoRemote RouteInfo = "remote"
)

type muxFunc func(net.Conn) (net.Conn, map[string]string, error)
type httpAuthFunc func(net.Conn, string, string, string) (bool, error)
type hostRewriteFunc func(net.Conn, string) (net.Conn, error)
type successFunc func(net.Conn) error

type Muxer struct {
	listener       net.Listener
	timeout        time.Duration
	vhostFunc      muxFunc
	authFunc       httpAuthFunc
	successFunc    successFunc
	rewriteFunc    hostRewriteFunc
	registryRouter *Routers
}

func NewMuxer(listener net.Listener, vhostFunc muxFunc, authFunc httpAuthFunc, successFunc successFunc, rewriteFunc hostRewriteFunc, timeout time.Duration) (mux *Muxer, err error) {
	mux = &Muxer{
		listener:       listener,
		timeout:        timeout,
		vhostFunc:      vhostFunc,
		authFunc:       authFunc,
		successFunc:    successFunc,
		rewriteFunc:    rewriteFunc,
		registryRouter: NewRouters(),
	}
	go mux.run()
	return mux, nil
}

type CreateConnFunc func(remoteAddr string) (net.Conn, error)

// RouteConfig is the params used to match HTTP requests
type RouteConfig struct {
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
func (v *Muxer) Listen(ctx context.Context, cfg *RouteConfig) (l *Listener, err error) {
	l = &Listener{
		name:        cfg.Domain,
		location:    cfg.Location,
		rewriteHost: cfg.RewriteHost,
		userName:    cfg.Username,
		passWord:    cfg.Password,
		mux:         v,
		accept:      make(chan net.Conn),
		ctx:         ctx,
	}
	err = v.registryRouter.Add(cfg.Domain, cfg.Location, l)
	if err != nil {
		return
	}
	return l, nil
}

func (v *Muxer) getListener(name, path string) (l *Listener, exist bool) {
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
}

func (v *Muxer) run() {
	for {
		conn, err := v.listener.Accept()
		if err != nil {
			return
		}
		go v.handle(conn)
	}
}

func (v *Muxer) handle(c net.Conn) {
	if err := c.SetDeadline(time.Now().Add(v.timeout)); err != nil {
		c.Close()
		return
	}

	sConn, reqInfoMap, err := v.vhostFunc(c)
	if err != nil {
		log.Debug("get hostname from http/https request error: %v", err)
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

	xl := xlog.FromContextSafe(l.ctx)
	if v.successFunc != nil {
		if err := v.successFunc(c); err != nil {
			xl.Info("success func failure on vhost connection: %v", err)
			c.Close()
			return
		}
	}

	// if authFunc is exist and userName/password is set
	// then verify user access
	if l.mux.authFunc != nil && l.userName != "" && l.passWord != "" {
		bAccess, err := l.mux.authFunc(c, l.userName, l.passWord, reqInfoMap["Authorization"])
		if bAccess == false || err != nil {
			xl.Debug("check http Authorization failed")
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

	xl.Debug("get new http request host [%s] path [%s]", name, path)
	err = errors.PanicToError(func() {
		l.accept <- c
	})
	if err != nil {
		xl.Warn("listener is already closed, ignore this request")
	}
}

type Listener struct {
	name        string
	location    string
	rewriteHost string
	userName    string
	passWord    string
	mux         *Muxer // for closing Muxer
	accept      chan net.Conn
	ctx         context.Context
}

func (l *Listener) Accept() (net.Conn, error) {
	xl := xlog.FromContextSafe(l.ctx)
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
			xl.Warn("host header rewrite failed: %v", err)
			return nil, fmt.Errorf("host header rewrite failed")
		}
		xl.Debug("rewrite host to [%s] success", l.rewriteHost)
		conn = sConn
	}
	return frpNet.NewContextConn(l.ctx, conn), nil
}

func (l *Listener) Close() error {
	l.mux.registryRouter.Del(l.name, l.location)
	close(l.accept)
	return nil
}

func (l *Listener) Name() string {
	return l.name
}

func (l *Listener) Addr() net.Addr {
	return (*net.TCPAddr)(nil)
}

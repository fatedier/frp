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

	"github.com/fatedier/golib/errors"

	"github.com/fatedier/frp/pkg/util/log"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type RouteInfo string

const (
	RouteInfoKey RouteInfo = "routeInfo"
)

type RequestRouteInfo struct {
	URL        string
	Host       string
	HTTPUser   string
	RemoteAddr string
	URLHost    string
	Endpoint   string
}

type (
	muxFunc         func(net.Conn) (net.Conn, map[string]string, error)
	authFunc        func(conn net.Conn, username, password string, reqInfoMap map[string]string) (bool, error)
	hostRewriteFunc func(net.Conn, string) (net.Conn, error)
	successHookFunc func(net.Conn, map[string]string) error
	failHookFunc    func(net.Conn)
)

// Muxer is a functional component used for https and tcpmux proxies.
// It accepts connections and extracts vhost information from the beginning of the connection data.
// It then routes the connection to its appropriate listener.
type Muxer struct {
	listener net.Listener
	timeout  time.Duration

	vhostFunc      muxFunc
	checkAuth      authFunc
	successHook    successHookFunc
	failHook       failHookFunc
	rewriteHost    hostRewriteFunc
	registryRouter *Routers
}

func NewMuxer(
	listener net.Listener,
	vhostFunc muxFunc,
	timeout time.Duration,
) (mux *Muxer, err error) {
	mux = &Muxer{
		listener:       listener,
		timeout:        timeout,
		vhostFunc:      vhostFunc,
		registryRouter: NewRouters(),
	}
	go mux.run()
	return mux, nil
}

func (v *Muxer) SetCheckAuthFunc(f authFunc) *Muxer {
	v.checkAuth = f
	return v
}

func (v *Muxer) SetSuccessHookFunc(f successHookFunc) *Muxer {
	v.successHook = f
	return v
}

func (v *Muxer) SetFailHookFunc(f failHookFunc) *Muxer {
	v.failHook = f
	return v
}

func (v *Muxer) SetRewriteHostFunc(f hostRewriteFunc) *Muxer {
	v.rewriteHost = f
	return v
}

type ChooseEndpointFunc func() (string, error)

type CreateConnFunc func(remoteAddr string) (net.Conn, error)

type CreateConnByEndpointFunc func(endpoint, remoteAddr string) (net.Conn, error)

// RouteConfig is the params used to match HTTP requests
type RouteConfig struct {
	Domain          string
	Location        string
	RewriteHost     string
	Username        string
	Password        string
	Headers         map[string]string
	RouteByHTTPUser string

	CreateConnFn           CreateConnFunc
	ChooseEndpointFn       ChooseEndpointFunc
	CreateConnByEndpointFn CreateConnByEndpointFunc
}

// listen for a new domain name, if rewriteHost is not empty and rewriteHost func is not nil,
// then rewrite the host header to rewriteHost
func (v *Muxer) Listen(ctx context.Context, cfg *RouteConfig) (l *Listener, err error) {
	l = &Listener{
		name:            cfg.Domain,
		location:        cfg.Location,
		routeByHTTPUser: cfg.RouteByHTTPUser,
		rewriteHost:     cfg.RewriteHost,
		username:        cfg.Username,
		password:        cfg.Password,
		mux:             v,
		accept:          make(chan net.Conn),
		ctx:             ctx,
	}
	err = v.registryRouter.Add(cfg.Domain, cfg.Location, cfg.RouteByHTTPUser, l)
	if err != nil {
		return
	}
	return l, nil
}

func (v *Muxer) getListener(name, path, httpUser string) (*Listener, bool) {
	findRouter := func(inName, inPath, inHTTPUser string) (*Listener, bool) {
		vr, ok := v.registryRouter.Get(inName, inPath, inHTTPUser)
		if ok {
			return vr.payload.(*Listener), true
		}
		// Try to check if there is one proxy that doesn't specify routerByHTTPUser, it means match all.
		vr, ok = v.registryRouter.Get(inName, inPath, "")
		if ok {
			return vr.payload.(*Listener), true
		}
		return nil, false
	}

	// first we check the full hostname
	// if not exist, then check the wildcard_domain such as *.example.com
	l, ok := findRouter(name, path, httpUser)
	if ok {
		return l, true
	}

	domainSplit := strings.Split(name, ".")
	for {
		if len(domainSplit) < 3 {
			break
		}

		domainSplit[0] = "*"
		name = strings.Join(domainSplit, ".")

		l, ok = findRouter(name, path, httpUser)
		if ok {
			return l, true
		}
		domainSplit = domainSplit[1:]
	}
	// Finally, try to check if there is one proxy that domain is "*" means match all domains.
	l, ok = findRouter("*", path, httpUser)
	if ok {
		return l, true
	}
	return nil, false
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
		_ = c.Close()
		return
	}

	sConn, reqInfoMap, err := v.vhostFunc(c)
	if err != nil {
		log.Debugf("get hostname from http/https request error: %v", err)
		_ = c.Close()
		return
	}

	name := strings.ToLower(reqInfoMap["Host"])
	path := strings.ToLower(reqInfoMap["Path"])
	httpUser := reqInfoMap["HTTPUser"]
	l, ok := v.getListener(name, path, httpUser)
	if !ok {
		log.Debugf("http request for host [%s] path [%s] httpUser [%s] not found", name, path, httpUser)
		v.failHook(sConn)
		return
	}

	xl := xlog.FromContextSafe(l.ctx)
	if v.successHook != nil {
		if err := v.successHook(c, reqInfoMap); err != nil {
			xl.Infof("success func failure on vhost connection: %v", err)
			_ = c.Close()
			return
		}
	}

	// if checkAuth func is exist and username/password is set
	// then verify user access
	if l.mux.checkAuth != nil && l.username != "" {
		ok, err := l.mux.checkAuth(c, l.username, l.password, reqInfoMap)
		if !ok || err != nil {
			xl.Debugf("auth failed for user: %s", l.username)
			_ = c.Close()
			return
		}
	}

	if err = sConn.SetDeadline(time.Time{}); err != nil {
		_ = c.Close()
		return
	}
	c = sConn

	xl.Debugf("new request host [%s] path [%s] httpUser [%s]", name, path, httpUser)
	err = errors.PanicToError(func() {
		l.accept <- c
	})
	if err != nil {
		xl.Warnf("listener is already closed, ignore this request")
	}
}

type Listener struct {
	name            string
	location        string
	routeByHTTPUser string
	rewriteHost     string
	username        string
	password        string
	mux             *Muxer // for closing Muxer
	accept          chan net.Conn
	ctx             context.Context
}

func (l *Listener) Accept() (net.Conn, error) {
	xl := xlog.FromContextSafe(l.ctx)
	conn, ok := <-l.accept
	if !ok {
		return nil, fmt.Errorf("Listener closed")
	}

	// if rewriteHost func is exist
	// rewrite http requests with a modified host header
	// if l.rewriteHost is empty, nothing to do
	if l.mux.rewriteHost != nil {
		sConn, err := l.mux.rewriteHost(conn, l.rewriteHost)
		if err != nil {
			xl.Warnf("host header rewrite failed: %v", err)
			return nil, fmt.Errorf("host header rewrite failed")
		}
		xl.Debugf("rewrite host to [%s] success", l.rewriteHost)
		conn = sConn
	}
	return netpkg.NewContextConn(l.ctx, conn), nil
}

func (l *Listener) Close() error {
	l.mux.registryRouter.Del(l.name, l.location, l.routeByHTTPUser)
	close(l.accept)
	return nil
}

func (l *Listener) Name() string {
	return l.name
}

func (l *Listener) Addr() net.Addr {
	return (*net.TCPAddr)(nil)
}

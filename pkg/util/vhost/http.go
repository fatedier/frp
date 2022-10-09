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

package vhost

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	frpIo "github.com/fatedier/golib/io"
	"github.com/fatedier/golib/pool"

	frpLog "github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/util"
)

var ErrNoRouteFound = errors.New("no route found")

type HTTPReverseProxyOptions struct {
	ResponseHeaderTimeoutS int64
}

type HTTPReverseProxy struct {
	proxy       *httputil.ReverseProxy
	vhostRouter *Routers

	responseHeaderTimeout time.Duration
}

func NewHTTPReverseProxy(option HTTPReverseProxyOptions, vhostRouter *Routers) *HTTPReverseProxy {
	if option.ResponseHeaderTimeoutS <= 0 {
		option.ResponseHeaderTimeoutS = 60
	}
	rp := &HTTPReverseProxy{
		responseHeaderTimeout: time.Duration(option.ResponseHeaderTimeoutS) * time.Second,
		vhostRouter:           vhostRouter,
	}
	proxy := &httputil.ReverseProxy{
		// Modify incoming requests by route policies.
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			url := req.Context().Value(RouteInfoURL).(string)
			routeByHTTPUser := req.Context().Value(RouteInfoHTTPUser).(string)
			oldHost, _ := util.CanonicalHost(req.Context().Value(RouteInfoHost).(string))
			rc := rp.GetRouteConfig(oldHost, url, routeByHTTPUser)
			if rc != nil {
				if rc.RewriteHost != "" {
					req.Host = rc.RewriteHost
				}
				// Set {domain}.{location}.{routeByHTTPUser} as URL host here to let http transport reuse connections.
				// TODO(fatedier): use proxy name instead?
				req.URL.Host = rc.Domain + "." +
					base64.StdEncoding.EncodeToString([]byte(rc.Location)) + "." +
					base64.StdEncoding.EncodeToString([]byte(rc.RouteByHTTPUser))

				for k, v := range rc.Headers {
					req.Header.Set(k, v)
				}
			} else {
				req.URL.Host = req.Host
			}
		},
		// Create a connection to one proxy routed by route policy.
		Transport: &http.Transport{
			ResponseHeaderTimeout: rp.responseHeaderTimeout,
			IdleConnTimeout:       60 * time.Second,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				url := ctx.Value(RouteInfoURL).(string)
				host, _ := util.CanonicalHost(ctx.Value(RouteInfoHost).(string))
				routerByHTTPUser := ctx.Value(RouteInfoHTTPUser).(string)
				remote := ctx.Value(RouteInfoRemote).(string)
				return rp.CreateConnection(host, url, routerByHTTPUser, remote)
			},
			Proxy: func(req *http.Request) (*url.URL, error) {
				// Use proxy mode if there is host in HTTP first request line.
				// GET http://example.com/ HTTP/1.1
				// Host: example.com
				//
				// Normal:
				// GET / HTTP/1.1
				// Host: example.com
				urlHost := req.Context().Value(RouteInfoURLHost).(string)
				if urlHost != "" {
					return req.URL, nil
				}
				return nil, nil
			},
		},
		BufferPool: newWrapPool(),
		ErrorLog:   log.New(newWrapLogger(), "", 0),
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			frpLog.Warn("do http proxy request [host: %s] error: %v", req.Host, err)
			rw.WriteHeader(http.StatusNotFound)
			_, _ = rw.Write(getNotFoundPageContent())
		},
	}
	rp.proxy = proxy
	return rp
}

// Register register the route config to reverse proxy
// reverse proxy will use CreateConnFn from routeCfg to create a connection to the remote service
func (rp *HTTPReverseProxy) Register(routeCfg RouteConfig) error {
	err := rp.vhostRouter.Add(routeCfg.Domain, routeCfg.Location, routeCfg.RouteByHTTPUser, &routeCfg)
	if err != nil {
		return err
	}
	return nil
}

// UnRegister unregister route config by domain and location
func (rp *HTTPReverseProxy) UnRegister(routeCfg RouteConfig) {
	rp.vhostRouter.Del(routeCfg.Domain, routeCfg.Location, routeCfg.RouteByHTTPUser)
}

func (rp *HTTPReverseProxy) GetRouteConfig(domain, location, routeByHTTPUser string) *RouteConfig {
	vr, ok := rp.getVhost(domain, location, routeByHTTPUser)
	if ok {
		frpLog.Debug("get new HTTP request host [%s] path [%s] httpuser [%s]", domain, location, routeByHTTPUser)
		return vr.payload.(*RouteConfig)
	}
	return nil
}

func (rp *HTTPReverseProxy) GetRealHost(domain, location, routeByHTTPUser string) (host string) {
	vr, ok := rp.getVhost(domain, location, routeByHTTPUser)
	if ok {
		host = vr.payload.(*RouteConfig).RewriteHost
	}
	return
}

func (rp *HTTPReverseProxy) GetHeaders(domain, location, routeByHTTPUser string) (headers map[string]string) {
	vr, ok := rp.getVhost(domain, location, routeByHTTPUser)
	if ok {
		headers = vr.payload.(*RouteConfig).Headers
	}
	return
}

// CreateConnection create a new connection by route config
func (rp *HTTPReverseProxy) CreateConnection(domain, location, routeByHTTPUser string, remoteAddr string) (net.Conn, error) {
	vr, ok := rp.getVhost(domain, location, routeByHTTPUser)
	if ok {
		fn := vr.payload.(*RouteConfig).CreateConnFn
		if fn != nil {
			return fn(remoteAddr)
		}
	}
	return nil, fmt.Errorf("%v: %s %s %s", ErrNoRouteFound, domain, location, routeByHTTPUser)
}

func (rp *HTTPReverseProxy) CheckAuth(domain, location, routeByHTTPUser, user, passwd string) bool {
	vr, ok := rp.getVhost(domain, location, routeByHTTPUser)
	if ok {
		checkUser := vr.payload.(*RouteConfig).Username
		checkPasswd := vr.payload.(*RouteConfig).Password
		if (checkUser != "" || checkPasswd != "") && (checkUser != user || checkPasswd != passwd) {
			return false
		}
	}
	return true
}

// getVhost trys to get vhost router by route policy.
func (rp *HTTPReverseProxy) getVhost(domain, location, routeByHTTPUser string) (*Router, bool) {
	findRouter := func(inDomain, inLocation, inRouteByHTTPUser string) (*Router, bool) {
		vr, ok := rp.vhostRouter.Get(inDomain, inLocation, inRouteByHTTPUser)
		if ok {
			return vr, ok
		}
		// Try to check if there is one proxy that doesn't specify routerByHTTPUser, it means match all.
		vr, ok = rp.vhostRouter.Get(inDomain, inLocation, "")
		if ok {
			return vr, ok
		}
		return nil, false
	}

	// First we check the full hostname
	// if not exist, then check the wildcard_domain such as *.example.com
	vr, ok := findRouter(domain, location, routeByHTTPUser)
	if ok {
		return vr, ok
	}

	// e.g. domain = test.example.com, try to match wildcard domains.
	// *.example.com
	// *.com
	domainSplit := strings.Split(domain, ".")
	for {
		if len(domainSplit) < 3 {
			break
		}

		domainSplit[0] = "*"
		domain = strings.Join(domainSplit, ".")
		vr, ok = findRouter(domain, location, routeByHTTPUser)
		if ok {
			return vr, true
		}
		domainSplit = domainSplit[1:]
	}

	// Finally, try to check if there is one proxy that domain is "*" means match all domains.
	vr, ok = findRouter("*", location, routeByHTTPUser)
	if ok {
		return vr, true
	}
	return nil, false
}

func (rp *HTTPReverseProxy) connectHandler(rw http.ResponseWriter, req *http.Request) {
	hj, ok := rw.(http.Hijacker)
	if !ok {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	client, _, err := hj.Hijack()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	url := req.Context().Value(RouteInfoURL).(string)
	routeByHTTPUser := req.Context().Value(RouteInfoHTTPUser).(string)
	domain, _ := util.CanonicalHost(req.Context().Value(RouteInfoHost).(string))
	remoteAddr := req.Context().Value(RouteInfoRemote).(string)

	remote, err := rp.CreateConnection(domain, url, routeByHTTPUser, remoteAddr)
	if err != nil {
		_ = notFoundResponse().Write(client)
		client.Close()
		return
	}
	_ = req.Write(remote)
	go frpIo.Join(remote, client)
}

func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

func (rp *HTTPReverseProxy) injectRequestInfoToCtx(req *http.Request) *http.Request {
	newctx := req.Context()
	newctx = context.WithValue(newctx, RouteInfoURL, req.URL.Path)
	newctx = context.WithValue(newctx, RouteInfoHost, req.Host)
	newctx = context.WithValue(newctx, RouteInfoURLHost, req.URL.Host)

	user := ""
	// If url host isn't empty, it's a proxy request. Get http user from Proxy-Authorization header.
	if req.URL.Host != "" {
		proxyAuth := req.Header.Get("Proxy-Authorization")
		if proxyAuth != "" {
			user, _, _ = parseBasicAuth(proxyAuth)
		}
	}
	if user == "" {
		user, _, _ = req.BasicAuth()
	}
	newctx = context.WithValue(newctx, RouteInfoHTTPUser, user)
	newctx = context.WithValue(newctx, RouteInfoRemote, req.RemoteAddr)
	return req.Clone(newctx)
}

func (rp *HTTPReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	domain, _ := util.CanonicalHost(req.Host)
	location := req.URL.Path
	user, passwd, _ := req.BasicAuth()
	if !rp.CheckAuth(domain, location, user, user, passwd) {
		rw.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	newreq := rp.injectRequestInfoToCtx(req)
	if req.Method == http.MethodConnect {
		rp.connectHandler(rw, newreq)
	} else {
		rp.proxy.ServeHTTP(rw, newreq)
	}
}

type wrapPool struct{}

func newWrapPool() *wrapPool { return &wrapPool{} }

func (p *wrapPool) Get() []byte { return pool.GetBuf(32 * 1024) }

func (p *wrapPool) Put(buf []byte) { pool.PutBuf(buf) }

type wrapLogger struct{}

func newWrapLogger() *wrapLogger { return &wrapLogger{} }

func (l *wrapLogger) Write(p []byte) (n int, err error) {
	frpLog.Warn("%s", string(bytes.TrimRight(p, "\n")))
	return len(p), nil
}

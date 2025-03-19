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
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	libio "github.com/fatedier/golib/io"
	"github.com/fatedier/golib/pool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
)

var ErrNoRouteFound = errors.New("no route found")

type HTTPReverseProxyOptions struct {
	ResponseHeaderTimeoutS int64
}

type HTTPReverseProxy struct {
	proxy       http.Handler
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
		Rewrite: func(r *httputil.ProxyRequest) {
			r.Out.Header["X-Forwarded-For"] = r.In.Header["X-Forwarded-For"]
			r.SetXForwarded()
			req := r.Out
			req.URL.Scheme = "http"
			reqRouteInfo := req.Context().Value(RouteInfoKey).(*RequestRouteInfo)
			originalHost, _ := httppkg.CanonicalHost(reqRouteInfo.Host)

			rc := req.Context().Value(RouteConfigKey).(*RouteConfig)
			if rc != nil {
				if rc.RewriteHost != "" {
					req.Host = rc.RewriteHost
				}

				var endpoint string
				if rc.ChooseEndpointFn != nil {
					// ignore error here, it will use CreateConnFn instead later
					endpoint, _ = rc.ChooseEndpointFn()
					reqRouteInfo.Endpoint = endpoint
					log.Tracef("choose endpoint name [%s] for http request host [%s] path [%s] httpuser [%s]",
						endpoint, originalHost, reqRouteInfo.URL, reqRouteInfo.HTTPUser)
				}
				// Set {domain}.{location}.{routeByHTTPUser}.{endpoint} as URL host here to let http transport reuse connections.
				req.URL.Host = rc.Domain + "." +
					base64.StdEncoding.EncodeToString([]byte(rc.Location)) + "." +
					base64.StdEncoding.EncodeToString([]byte(rc.RouteByHTTPUser)) + "." +
					base64.StdEncoding.EncodeToString([]byte(endpoint))

				for k, v := range rc.Headers {
					req.Header.Set(k, v)
				}
			} else {
				req.URL.Host = req.Host
			}

			for k, v := range req.Header {
				if strings.Contains(k, "Websocket") {
					delete(req.Header, k)
					req.Header[strings.ReplaceAll(k, "Websocket", "WebSocket")] = v
				}
			}
		},
		ModifyResponse: func(r *http.Response) error {
			rc := r.Request.Context().Value(RouteConfigKey).(*RouteConfig)
			if rc != nil {
				for k, v := range rc.ResponseHeaders {
					r.Header.Set(k, v)
				}
			}
			return nil
		},
		// Create a connection to one proxy routed by route policy.
		Transport: &http.Transport{
			ResponseHeaderTimeout: rp.responseHeaderTimeout,
			IdleConnTimeout:       60 * time.Second,
			MaxIdleConnsPerHost:   5,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return rp.CreateConnection(ctx.Value(RouteInfoKey).(*RequestRouteInfo), true)
			},
			Proxy: func(req *http.Request) (*url.URL, error) {
				// Use proxy mode if there is host in HTTP first request line.
				// GET http://example.com/ HTTP/1.1
				// Host: example.com
				//
				// Normal:
				// GET / HTTP/1.1
				// Host: example.com
				urlHost := req.Context().Value(RouteInfoKey).(*RequestRouteInfo).URLHost
				if urlHost != "" {
					return req.URL, nil
				}
				return nil, nil
			},
		},
		BufferPool: pool.NewBuffer(32 * 1024),
		ErrorLog:   stdlog.New(log.NewWriteLogger(log.WarnLevel, 2), "", 0),
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			log.Logf(log.WarnLevel, 1, "do http proxy request [host: %s] error: %v", req.Host, err)
			if err != nil {
				if e, ok := err.(net.Error); ok && e.Timeout() {
					rw.WriteHeader(http.StatusGatewayTimeout)
					return
				}
			}
			rw.WriteHeader(http.StatusNotFound)
			_, _ = rw.Write(getNotFoundPageContent())
		},
	}
	rp.proxy = h2c.NewHandler(proxy, &http2.Server{})
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
		log.Debugf("get new HTTP request host [%s] path [%s] httpuser [%s]", domain, location, routeByHTTPUser)
		return vr.payload.(*RouteConfig)
	}
	return nil
}

// CreateConnection create a new connection by route config
func (rp *HTTPReverseProxy) CreateConnection(reqRouteInfo *RequestRouteInfo, byEndpoint bool) (net.Conn, error) {
	host, _ := httppkg.CanonicalHost(reqRouteInfo.Host)
	vr, ok := rp.getVhost(host, reqRouteInfo.URL, reqRouteInfo.HTTPUser)
	if ok {
		if byEndpoint {
			fn := vr.payload.(*RouteConfig).CreateConnByEndpointFn
			if fn != nil {
				return fn(reqRouteInfo.Endpoint, reqRouteInfo.RemoteAddr)
			}
		}
		fn := vr.payload.(*RouteConfig).CreateConnFn
		if fn != nil {
			return fn(reqRouteInfo.RemoteAddr)
		}
	}
	return nil, fmt.Errorf("%v: %s %s %s", ErrNoRouteFound, host, reqRouteInfo.URL, reqRouteInfo.HTTPUser)
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

// getVhost tries to get vhost router by route policy.
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

	remote, err := rp.CreateConnection(req.Context().Value(RouteInfoKey).(*RequestRouteInfo), false)
	if err != nil {
		_ = NotFoundResponse().Write(client)
		client.Close()
		return
	}
	_ = req.Write(remote)
	go libio.Join(remote, client)
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

	reqRouteInfo := &RequestRouteInfo{
		URL:        req.URL.Path,
		Host:       req.Host,
		HTTPUser:   user,
		RemoteAddr: req.RemoteAddr,
		URLHost:    req.URL.Host,
	}

	originalHost, _ := httppkg.CanonicalHost(reqRouteInfo.Host)
	rc := rp.GetRouteConfig(originalHost, reqRouteInfo.URL, reqRouteInfo.HTTPUser)

	newctx := req.Context()
	newctx = context.WithValue(newctx, RouteInfoKey, reqRouteInfo)
	newctx = context.WithValue(newctx, RouteConfigKey, rc)
	return req.Clone(newctx)
}

func (rp *HTTPReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	domain, _ := httppkg.CanonicalHost(req.Host)
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

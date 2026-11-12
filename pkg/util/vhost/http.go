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
	"strconv"
	"time"

	libio "github.com/fatedier/golib/io"
	"github.com/fatedier/golib/pool"

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
				// Keep connection reuse within the same route registration and endpoint.
				req.URL.Host = rc.Domain + "." +
					base64.StdEncoding.EncodeToString([]byte(rc.Location)) + "." +
					base64.StdEncoding.EncodeToString([]byte(rc.RouteByHTTPUser)) + "." +
					base64.StdEncoding.EncodeToString([]byte(endpoint)) + "." +
					strconv.FormatUint(reqRouteInfo.routeID, 10)

				for k, v := range rc.Headers {
					req.Header.Set(k, v)
				}
			} else {
				req.URL.Host = req.Host
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
				return createConnection(ctx.Value(RouteInfoKey).(*RequestRouteInfo), ctx.Value(RouteConfigKey).(*RouteConfig), true)
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

// createConnection uses the same route configuration that authorized the request.
func createConnection(reqRouteInfo *RequestRouteInfo, rc *RouteConfig, byEndpoint bool) (net.Conn, error) {
	if rc != nil {
		if byEndpoint {
			fn := rc.CreateConnByEndpointFn
			if fn != nil {
				return fn(reqRouteInfo.Endpoint, reqRouteInfo.RemoteAddr)
			}
		}
		fn := rc.CreateConnFn
		if fn != nil {
			return fn(reqRouteInfo.RemoteAddr)
		}
	}
	host, _ := httppkg.CanonicalHost(reqRouteInfo.Host)
	return nil, fmt.Errorf("%v: %s %s %s", ErrNoRouteFound, host, reqRouteInfo.URL, reqRouteInfo.HTTPUser)
}

func checkRouteAuthByRequest(req *http.Request, rc *RouteConfig) bool {
	if rc == nil {
		return true
	}
	if rc.Username == "" && rc.Password == "" {
		return true
	}

	if req.URL.Host != "" {
		proxyAuth := req.Header.Get("Proxy-Authorization")
		if proxyAuth == "" {
			return false
		}
		user, passwd, ok := httppkg.ParseBasicAuth(proxyAuth)
		return ok && user == rc.Username && passwd == rc.Password
	}

	user, passwd, ok := req.BasicAuth()
	return ok && user == rc.Username && passwd == rc.Password
}

func (rp *HTTPReverseProxy) connectHandler(rw http.ResponseWriter, req *http.Request) {
	if req.ContentLength > 0 || len(req.TransferEncoding) > 0 {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	hj, ok := rw.(http.Hijacker)
	if !ok {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Forward only CONNECT headers; the original Body must not be used after Hijack.
	outreq := req.Clone(req.Context())
	outreq.Body = nil
	outreq.GetBody = nil
	outreq.ContentLength = 0
	outreq.TransferEncoding = nil
	outreq.Trailer = nil
	outreq.Header.Del("Content-Length")
	outreq.Header.Del("Transfer-Encoding")
	outreq.Header.Del("Trailer")

	client, buffered, err := hj.Hijack()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	remote, err := createConnection(
		req.Context().Value(RouteInfoKey).(*RequestRouteInfo), req.Context().Value(RouteConfigKey).(*RouteConfig), false)
	if err != nil {
		_ = NotFoundResponse().Write(client)
		client.Close()
		return
	}
	if err := outreq.Write(remote); err != nil {
		_ = remote.Close()
		_ = client.Close()
		return
	}
	go libio.Join(remote, libio.WrapReadWriteCloser(buffered.Reader, client, client.Close))
}

func getRequestRouteUser(req *http.Request) string {
	if req.URL.Host != "" {
		proxyAuth := req.Header.Get("Proxy-Authorization")
		if proxyAuth == "" {
			// Preserve legacy proxy-mode routing when clients send only Authorization,
			// so requests still hit the matched route and return 407 instead of 404.
			// Auth validation intentionally does not share this fallback.
			user, _, _ := req.BasicAuth()
			return user
		}
		user, _, _ := httppkg.ParseBasicAuth(proxyAuth)
		return user
	}
	user, _, _ := req.BasicAuth()
	return user
}

func (rp *HTTPReverseProxy) injectRequestInfoToCtx(req *http.Request) *http.Request {
	user := getRequestRouteUser(req)

	reqRouteInfo := &RequestRouteInfo{
		URL:        req.URL.Path,
		Host:       req.Host,
		HTTPUser:   user,
		RemoteAddr: req.RemoteAddr,
		URLHost:    req.URL.Host,
	}

	originalHost, _ := httppkg.CanonicalHost(reqRouteInfo.Host)
	var rc *RouteConfig
	if vr, ok := rp.vhostRouter.getByRoute(originalHost, reqRouteInfo.URL, reqRouteInfo.HTTPUser); ok {
		rc = vr.payload.(*RouteConfig)
		reqRouteInfo.routeID = vr.id
		log.Debugf("get new http request host [%s] path [%s] httpuser [%s]", originalHost, reqRouteInfo.URL, reqRouteInfo.HTTPUser)
	}

	newctx := req.Context()
	newctx = context.WithValue(newctx, RouteInfoKey, reqRouteInfo)
	newctx = context.WithValue(newctx, RouteConfigKey, rc)
	return req.Clone(newctx)
}

func (rp *HTTPReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	newreq := rp.injectRequestInfoToCtx(req)
	rc := newreq.Context().Value(RouteConfigKey).(*RouteConfig)
	if !checkRouteAuthByRequest(req, rc) {
		if req.URL.Host != "" {
			rw.Header().Set("Proxy-Authenticate", `Basic realm="Restricted"`)
			http.Error(rw, http.StatusText(http.StatusProxyAuthRequired), http.StatusProxyAuthRequired)
		} else {
			rw.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
		return
	}

	if req.Method == http.MethodConnect {
		rp.connectHandler(rw, newreq)
	} else {
		rp.proxy.ServeHTTP(rw, newreq)
	}
}

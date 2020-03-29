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
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	frpLog "github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/util"

	"github.com/fatedier/golib/pool"
)

var (
	ErrNoDomain = errors.New("no such domain")
)

type HttpReverseProxyOptions struct {
	ResponseHeaderTimeoutS int64
}

type HttpReverseProxy struct {
	proxy       *ReverseProxy
	vhostRouter *VhostRouters

	responseHeaderTimeout time.Duration
}

func NewHttpReverseProxy(option HttpReverseProxyOptions, vhostRouter *VhostRouters) *HttpReverseProxy {
	if option.ResponseHeaderTimeoutS <= 0 {
		option.ResponseHeaderTimeoutS = 60
	}
	rp := &HttpReverseProxy{
		responseHeaderTimeout: time.Duration(option.ResponseHeaderTimeoutS) * time.Second,
		vhostRouter:           vhostRouter,
	}
	proxy := &ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			url := req.Context().Value("url").(string)
			oldHost := util.GetHostFromAddr(req.Context().Value("host").(string))
			host := rp.GetRealHost(oldHost, url)
			if host != "" {
				req.Host = host
			}
			req.URL.Host = req.Host

			headers := rp.GetHeaders(oldHost, url)
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		},
		Transport: &http.Transport{
			ResponseHeaderTimeout: rp.responseHeaderTimeout,
			DisableKeepAlives:     true,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				url := ctx.Value("url").(string)
				host := util.GetHostFromAddr(ctx.Value("host").(string))
				remote := ctx.Value("remote").(string)
				return rp.CreateConnection(host, url, remote)
			},
		},
		BufferPool: newWrapPool(),
		ErrorLog:   log.New(newWrapLogger(), "", 0),
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			frpLog.Warn("do http proxy request error: %v", err)
			rw.WriteHeader(http.StatusNotFound)
			rw.Write(getNotFoundPageContent())
		},
	}
	rp.proxy = proxy
	return rp
}

// Register register the route config to reverse proxy
// reverse proxy will use CreateConnFn from routeCfg to create a connection to the remote service
func (rp *HttpReverseProxy) Register(routeCfg VhostRouteConfig) error {
	err := rp.vhostRouter.Add(routeCfg.Domain, routeCfg.Location, &routeCfg)
	if err != nil {
		return err
	}
	return nil
}

// UnRegister unregister route config by domain and location
func (rp *HttpReverseProxy) UnRegister(domain string, location string) {
	rp.vhostRouter.Del(domain, location)
}

func (rp *HttpReverseProxy) GetRealHost(domain string, location string) (host string) {
	vr, ok := rp.getVhost(domain, location)
	if ok {
		host = vr.payload.(*VhostRouteConfig).RewriteHost
	}
	return
}

func (rp *HttpReverseProxy) GetHeaders(domain string, location string) (headers map[string]string) {
	vr, ok := rp.getVhost(domain, location)
	if ok {
		headers = vr.payload.(*VhostRouteConfig).Headers
	}
	return
}

// CreateConnection create a new connection by route config
func (rp *HttpReverseProxy) CreateConnection(domain string, location string, remoteAddr string) (net.Conn, error) {
	vr, ok := rp.getVhost(domain, location)
	if ok {
		fn := vr.payload.(*VhostRouteConfig).CreateConnFn
		if fn != nil {
			return fn(remoteAddr)
		}
	}
	return nil, fmt.Errorf("%v: %s %s", ErrNoDomain, domain, location)
}

func (rp *HttpReverseProxy) CheckAuth(domain, location, user, passwd string) bool {
	vr, ok := rp.getVhost(domain, location)
	if ok {
		checkUser := vr.payload.(*VhostRouteConfig).Username
		checkPasswd := vr.payload.(*VhostRouteConfig).Password
		if (checkUser != "" || checkPasswd != "") && (checkUser != user || checkPasswd != passwd) {
			return false
		}
	}
	return true
}

// getVhost get vhost router by domain and location
func (rp *HttpReverseProxy) getVhost(domain string, location string) (vr *VhostRouter, ok bool) {
	// first we check the full hostname
	// if not exist, then check the wildcard_domain such as *.example.com
	vr, ok = rp.vhostRouter.Get(domain, location)
	if ok {
		return
	}

	domainSplit := strings.Split(domain, ".")
	if len(domainSplit) < 3 {
		return nil, false
	}

	for {
		if len(domainSplit) < 3 {
			return nil, false
		}

		domainSplit[0] = "*"
		domain = strings.Join(domainSplit, ".")
		vr, ok = rp.vhostRouter.Get(domain, location)
		if ok {
			return vr, true
		}
		domainSplit = domainSplit[1:]
	}
}

func (rp *HttpReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	domain := util.GetHostFromAddr(req.Host)
	location := req.URL.Path
	user, passwd, _ := req.BasicAuth()
	if !rp.CheckAuth(domain, location, user, passwd) {
		rw.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	rp.proxy.ServeHTTP(rw, req)
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

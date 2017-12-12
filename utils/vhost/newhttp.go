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
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	frpLog "github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/pool"
)

var (
	responseHeaderTimeout = time.Duration(30) * time.Second

	ErrRouterConfigConflict = errors.New("router config conflict")
	ErrNoDomain             = errors.New("no such domain")
)

func getHostFromAddr(addr string) (host string) {
	strs := strings.Split(addr, ":")
	if len(strs) > 1 {
		host = strs[0]
	} else {
		host = addr
	}
	return
}

type CreateConnFunc func() (frpNet.Conn, error)

type ProxyOption struct {
	RewriteHost string
	DialFunc    CreateConnFunc
}

type HttpReverseProxy struct {
	proxy *ReverseProxy
	tr    *http.Transport

	vhostRouter *VhostRouters

	cfgMu sync.RWMutex
}

func NewHttpReverseProxy() *HttpReverseProxy {
	rp := &HttpReverseProxy{
		vhostRouter: NewVhostRouters(),
	}
	proxy := &ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			url := req.Context().Value("url").(string)
			host := getHostFromAddr(req.Context().Value("host").(string))
			host = rp.GetRealHost(host, url)
			if host != "" {
				req.Host = host
			}
			req.URL.Host = req.Host
		},
		Transport: &http.Transport{
			ResponseHeaderTimeout: responseHeaderTimeout,
			DisableKeepAlives:     true,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				url := ctx.Value("url").(string)
				host := getHostFromAddr(ctx.Value("host").(string))
				return rp.CreateConnection(host, url)
			},
		},
		BufferPool: newWrapPool(),
		ErrorLog:   log.New(newWrapLogger(), "", 0),
	}
	rp.proxy = proxy
	return rp
}

func (rp *HttpReverseProxy) Register(domain string, location string, rewriteHost string, fn CreateConnFunc) error {
	rp.cfgMu.Lock()
	defer rp.cfgMu.Unlock()
	_, ok := rp.vhostRouter.Exist(domain, location)
	if ok {
		return ErrRouterConfigConflict
	} else {
		payload := &ProxyOption{
			RewriteHost: rewriteHost,
			DialFunc:    fn,
		}
		rp.vhostRouter.Add(domain, location, payload)
	}
	return nil
}

func (rp *HttpReverseProxy) UnRegister(domain string, location string) {
	rp.cfgMu.Lock()
	defer rp.cfgMu.Unlock()
	rp.vhostRouter.Del(domain, location)
}

func (rp *HttpReverseProxy) GetRealHost(domain string, location string) (host string) {
	vr, ok := rp.getVhost(domain, location)
	if ok {
		host = vr.payload.(*ProxyOption).RewriteHost
	}
	return
}

func (rp *HttpReverseProxy) CreateConnection(domain string, location string) (net.Conn, error) {
	vr, ok := rp.getVhost(domain, location)
	if ok {
		fn := vr.payload.(*ProxyOption).DialFunc
		if fn != nil {
			return fn()
		}
	}
	return nil, ErrNoDomain
}

func (rp *HttpReverseProxy) getVhost(domain string, location string) (vr *VhostRouter, ok bool) {
	rp.cfgMu.RLock()
	defer rp.cfgMu.RUnlock()

	// first we check the full hostname
	// if not exist, then check the wildcard_domain such as *.example.com
	vr, ok = rp.vhostRouter.Get(domain, location)
	if ok {
		return
	}

	domainSplit := strings.Split(domain, ".")
	if len(domainSplit) < 3 {
		return vr, false
	}
	domainSplit[0] = "*"
	domain = strings.Join(domainSplit, ".")
	vr, ok = rp.vhostRouter.Get(domain, location)
	return
}

func (rp *HttpReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
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

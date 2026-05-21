// Copyright 2026 The frp Authors
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

//go:build !frps

package client

import (
	"context"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/fatedier/golib/pool"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/plugin/client/internal/httpsserver"
	"github.com/fatedier/frp/pkg/util/log"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

const httpBridgeReadHeaderTimeout = 60 * time.Second

func rewriteHTTPPluginRequest(
	req *http.Request,
	scheme string,
	localAddr string,
	hostHeaderRewrite string,
	requestHeaders v1.HeaderOperations,
) {
	req.URL.Scheme = scheme
	req.URL.Host = localAddr
	if hostHeaderRewrite != "" {
		req.Host = hostHeaderRewrite
	}
	for k, v := range requestHeaders.Set {
		req.Header.Set(k, v)
	}
}

type httpBridgePlugin struct {
	l *Listener
	s *http.Server

	useSourceRemoteAddr bool
}

func newHTTPBridgePluginServer(handler http.Handler, useSourceRemoteAddr bool) *httpBridgePlugin {
	listener := NewProxyListener()
	p := &httpBridgePlugin{
		l:                   listener,
		useSourceRemoteAddr: useSourceRemoteAddr,
	}
	p.s = &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: httpBridgeReadHeaderTimeout,
	}
	go func() {
		_ = p.s.Serve(listener)
	}()
	return p
}

func newHTTPSBridgePluginServer(
	handler http.Handler,
	crtPath string,
	keyPath string,
	enableHTTP2 *bool,
	useSourceRemoteAddr bool,
) (*httpBridgePlugin, error) {
	listener := NewProxyListener()
	server, err := httpsserver.New(handler, crtPath, keyPath, enableHTTP2)
	if err != nil {
		return nil, err
	}
	p := &httpBridgePlugin{
		l:                   listener,
		s:                   server,
		useSourceRemoteAddr: useSourceRemoteAddr,
	}
	go func() {
		_ = p.s.ServeTLS(listener, "", "")
	}()
	return p, nil
}

func newHTTPBridgeReverseProxy(
	rewrite func(*httputil.ProxyRequest),
	transport http.RoundTripper,
) *httputil.ReverseProxy {
	rp := &httputil.ReverseProxy{
		Rewrite:    rewrite,
		BufferPool: pool.NewBuffer(32 * 1024),
		ErrorLog:   stdlog.New(log.NewWriteLogger(log.WarnLevel, 2), "", 0),
	}
	if transport != nil {
		rp.Transport = transport
	}
	return rp
}

func (p *httpBridgePlugin) Handle(_ context.Context, connInfo *ConnectionInfo) {
	wrapConn := netpkg.WrapReadWriteCloserToConn(connInfo.Conn, connInfo.UnderlyingConn)
	if p.useSourceRemoteAddr && connInfo.SrcAddr != nil {
		wrapConn.SetRemoteAddr(connInfo.SrcAddr)
	}
	_ = p.l.PutConn(wrapConn)
}

func (p *httpBridgePlugin) Close() error {
	err := p.s.Close()
	_ = p.l.Close()
	return err
}

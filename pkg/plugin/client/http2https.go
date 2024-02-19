// Copyright 2019 fatedier, fatedier@gmail.com
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

package plugin

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httputil"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

func init() {
	Register(v1.PluginHTTP2HTTPS, NewHTTP2HTTPSPlugin)
}

type HTTP2HTTPSPlugin struct {
	opts *v1.HTTP2HTTPSPluginOptions

	l *Listener
	s *http.Server
}

func NewHTTP2HTTPSPlugin(options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.HTTP2HTTPSPluginOptions)

	listener := NewProxyListener()

	p := &HTTP2HTTPSPlugin{
		opts: opts,
		l:    listener,
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	rp := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			req := r.Out
			req.URL.Scheme = "https"
			req.URL.Host = p.opts.LocalAddr
			if p.opts.HostHeaderRewrite != "" {
				req.Host = p.opts.HostHeaderRewrite
			}
			for k, v := range p.opts.RequestHeaders.Set {
				req.Header.Set(k, v)
			}
		},
		Transport: tr,
	}

	p.s = &http.Server{
		Handler:           rp,
		ReadHeaderTimeout: 0,
	}

	go func() {
		_ = p.s.Serve(listener)
	}()

	return p, nil
}

func (p *HTTP2HTTPSPlugin) Handle(conn io.ReadWriteCloser, realConn net.Conn, _ *ExtraInfo) {
	wrapConn := netpkg.WrapReadWriteCloserToConn(conn, realConn)
	_ = p.l.PutConn(wrapConn)
}

func (p *HTTP2HTTPSPlugin) Name() string {
	return v1.PluginHTTP2HTTPS
}

func (p *HTTP2HTTPSPlugin) Close() error {
	return p.s.Close()
}

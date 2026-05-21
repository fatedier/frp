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

package client

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func init() {
	Register(v1.PluginHTTP2HTTPS, NewHTTP2HTTPSPlugin)
}

type HTTP2HTTPSPlugin struct {
	opts *v1.HTTP2HTTPSPluginOptions

	*httpBridgePlugin
}

func NewHTTP2HTTPSPlugin(_ PluginContext, options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.HTTP2HTTPSPluginOptions)

	p := &HTTP2HTTPSPlugin{
		opts: opts,
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	rp := newHTTPBridgeReverseProxy(
		func(r *httputil.ProxyRequest) {
			r.Out.Header["X-Forwarded-For"] = r.In.Header["X-Forwarded-For"]
			r.Out.Header["X-Forwarded-Host"] = r.In.Header["X-Forwarded-Host"]
			r.Out.Header["X-Forwarded-Proto"] = r.In.Header["X-Forwarded-Proto"]
			req := r.Out
			rewriteHTTPPluginRequest(req, "https", p.opts.LocalAddr, p.opts.HostHeaderRewrite, p.opts.RequestHeaders)
		},
		tr,
	)
	p.httpBridgePlugin = newHTTPBridgePluginServer(rp, false)

	return p, nil
}

func (p *HTTP2HTTPSPlugin) Name() string {
	return v1.PluginHTTP2HTTPS
}

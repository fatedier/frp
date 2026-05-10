// Copyright 2024 The frp Authors
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
	"net/http/httputil"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func init() {
	Register(v1.PluginHTTP2HTTP, NewHTTP2HTTPPlugin)
}

type HTTP2HTTPPlugin struct {
	opts *v1.HTTP2HTTPPluginOptions

	*httpBridgePlugin
}

func NewHTTP2HTTPPlugin(_ PluginContext, options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.HTTP2HTTPPluginOptions)

	p := &HTTP2HTTPPlugin{
		opts: opts,
	}

	rp := newHTTPBridgeReverseProxy(
		func(r *httputil.ProxyRequest) {
			req := r.Out
			rewriteHTTPPluginRequest(req, "http", p.opts.LocalAddr, p.opts.HostHeaderRewrite, p.opts.RequestHeaders)
		},
		nil,
	)
	p.httpBridgePlugin = newHTTPBridgePluginServer(rp, false)

	return p, nil
}

func (p *HTTP2HTTPPlugin) Name() string {
	return v1.PluginHTTP2HTTP
}

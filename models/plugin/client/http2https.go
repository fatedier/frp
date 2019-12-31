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

package plugin

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"

	frpNet "github.com/fatedier/frp/utils/net"
)

const PluginHTTP2HTTPS = "http2https"

func init() {
	Register(PluginHTTP2HTTPS, NewHTTP2HTTPSPlugin)
}

type HTTP2HTTPSPlugin struct {
	hostHeaderRewrite string
	localAddr         string
	headers           map[string]string

	l *Listener
	s *http.Server
}

func NewHTTP2HTTPSPlugin(params map[string]string) (Plugin, error) {
	localAddr := params["plugin_local_addr"]
	hostHeaderRewrite := params["plugin_host_header_rewrite"]
	headers := make(map[string]string)
	for k, v := range params {
		if !strings.HasPrefix(k, "plugin_header_") {
			continue
		}
		if k = strings.TrimPrefix(k, "plugin_header_"); k != "" {
			headers[k] = v
		}
	}

	if localAddr == "" {
		return nil, fmt.Errorf("plugin_local_addr is required")
	}

	listener := NewProxyListener()

	p := &HTTPS2HTTPPlugin{
		localAddr:         localAddr,
		hostHeaderRewrite: hostHeaderRewrite,
		headers:           headers,
		l:                 listener,
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "https"
			req.URL.Host = p.localAddr
			if p.hostHeaderRewrite != "" {
				req.Host = p.hostHeaderRewrite
			}
			for k, v := range p.headers {
				req.Header.Set(k, v)
			}
		},
		Transport: tr,
	}

	p.s = &http.Server{
		Handler: rp,
	}

	go p.s.Serve(listener)

	return p, nil
}

func (p *HTTP2HTTPSPlugin) Handle(conn io.ReadWriteCloser, realConn net.Conn, extraBufToLocal []byte) {
	wrapConn := frpNet.WrapReadWriteCloserToConn(conn, realConn)
	p.l.PutConn(wrapConn)
}

func (p *HTTP2HTTPSPlugin) Name() string {
	return PluginHTTP2HTTPS
}

func (p *HTTP2HTTPSPlugin) Close() error {
	if err := p.s.Close(); err != nil {
		return err
	}
	return nil
}

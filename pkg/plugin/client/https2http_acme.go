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

	frpNet "github.com/fatedier/frp/pkg/util/net"
	"golang.org/x/crypto/acme/autocert"
)

const PluginHTTPS2HTTPACME = "https2http_acme"

func init() {
	Register(PluginHTTPS2HTTPACME, NewHTTPS2HTTPACMEPlugin)
}

type HTTPS2HTTPACMEPlugin struct {
	hostHeaderRewrite string
	localAddr         string
	headers           map[string]string

	l *Listener
	s *http.Server
	m *autocert.Manager
}

func NewHTTPS2HTTPACMEPlugin(params map[string]string) (Plugin, error) {
	email := params["plugin_email"]
	certsPath := params["plugin_certs_path"]
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

	if certsPath == "" {
		return nil, fmt.Errorf("plugin_certs_path is required")
	}
	if localAddr == "" {
		return nil, fmt.Errorf("plugin_local_addr is required")
	}

	listener := NewProxyListener()

	manager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(certsPath),
		Email:  email,
	}

	p := &HTTPS2HTTPACMEPlugin{
		localAddr:         localAddr,
		hostHeaderRewrite: hostHeaderRewrite,
		headers:           headers,
		l:                 listener,
		m:                 &manager,
	}

	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = p.localAddr
			if p.hostHeaderRewrite != "" {
				req.Host = p.hostHeaderRewrite
			}
			for k, v := range p.headers {
				req.Header.Set(k, v)
			}
		},
	}

	p.s = &http.Server{
		Handler: rp,
	}

	ln := tls.NewListener(listener, p.m.TLSConfig())

	go p.s.Serve(ln)
	return p, nil
}

func (p *HTTPS2HTTPACMEPlugin) Handle(conn io.ReadWriteCloser, realConn net.Conn, extraBufToLocal []byte) {
	wrapConn := frpNet.WrapReadWriteCloserToConn(conn, realConn)
	p.l.PutConn(wrapConn)
}

func (p *HTTPS2HTTPACMEPlugin) Name() string {
	return PluginHTTPS2HTTPACME
}

func (p *HTTPS2HTTPACMEPlugin) Close() error {
	if err := p.s.Close(); err != nil {
		return err
	}
	return nil
}

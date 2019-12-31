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

const PluginHTTPS2HTTP = "https2http"

func init() {
	Register(PluginHTTPS2HTTP, NewHTTPS2HTTPPlugin)
}

type HTTPS2HTTPPlugin struct {
	crtPath           string
	keyPath           string
	hostHeaderRewrite string
	localAddr         string
	headers           map[string]string

	l *Listener
	s *http.Server
}

func NewHTTPS2HTTPPlugin(params map[string]string) (Plugin, error) {
	crtPath := params["plugin_crt_path"]
	keyPath := params["plugin_key_path"]
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

	if crtPath == "" {
		return nil, fmt.Errorf("plugin_crt_path is required")
	}
	if keyPath == "" {
		return nil, fmt.Errorf("plugin_key_path is required")
	}
	if localAddr == "" {
		return nil, fmt.Errorf("plugin_local_addr is required")
	}

	listener := NewProxyListener()

	p := &HTTPS2HTTPPlugin{
		crtPath:           crtPath,
		keyPath:           keyPath,
		localAddr:         localAddr,
		hostHeaderRewrite: hostHeaderRewrite,
		headers:           headers,
		l:                 listener,
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

	tlsConfig, err := p.genTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("gen TLS config error: %v", err)
	}
	ln := tls.NewListener(listener, tlsConfig)

	go p.s.Serve(ln)
	return p, nil
}

func (p *HTTPS2HTTPPlugin) genTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(p.crtPath, p.keyPath)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	return config, nil
}

func (p *HTTPS2HTTPPlugin) Handle(conn io.ReadWriteCloser, realConn net.Conn, extraBufToLocal []byte) {
	wrapConn := frpNet.WrapReadWriteCloserToConn(conn, realConn)
	p.l.PutConn(wrapConn)
}

func (p *HTTPS2HTTPPlugin) Name() string {
	return PluginHTTPS2HTTP
}

func (p *HTTPS2HTTPPlugin) Close() error {
	if err := p.s.Close(); err != nil {
		return err
	}
	return nil
}

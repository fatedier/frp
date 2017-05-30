// Copyright 2017 frp team
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
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/fatedier/frp/models/proto/tcp"
	"github.com/fatedier/frp/utils/errors"
	frpNet "github.com/fatedier/frp/utils/net"
)

const PluginHttpProxy = "http_proxy"

func init() {
	Register(PluginHttpProxy, NewHttpProxyPlugin)
}

type Listener struct {
	conns  chan net.Conn
	closed bool
	mu     sync.Mutex
}

func NewProxyListener() *Listener {
	return &Listener{
		conns: make(chan net.Conn, 64),
	}
}

func (l *Listener) Accept() (net.Conn, error) {
	conn, ok := <-l.conns
	if !ok {
		return nil, fmt.Errorf("listener closed")
	}
	return conn, nil
}

func (l *Listener) PutConn(conn net.Conn) error {
	err := errors.PanicToError(func() {
		l.conns <- conn
	})
	return err
}

func (l *Listener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		close(l.conns)
	}
	return nil
}

func (l *Listener) Addr() net.Addr {
	return (*net.TCPAddr)(nil)
}

type HttpProxy struct {
	l          *Listener
	s          *http.Server
	AuthUser   string
	AuthPasswd string
}

func NewHttpProxyPlugin(params map[string]string) (Plugin, error) {
	user := params["plugin_http_user"]
	passwd := params["plugin_http_passwd"]
	listener := NewProxyListener()

	hp := &HttpProxy{
		l:          listener,
		AuthUser:   user,
		AuthPasswd: passwd,
	}

	hp.s = &http.Server{
		Handler: hp,
	}

	go hp.s.Serve(listener)
	return hp, nil
}

func (hp *HttpProxy) Name() string {
	return PluginHttpProxy
}

func (hp *HttpProxy) Handle(conn io.ReadWriteCloser) {
	var wrapConn net.Conn
	if realConn, ok := conn.(net.Conn); ok {
		wrapConn = realConn
	} else {
		wrapConn = frpNet.WrapReadWriteCloserToConn(conn)
	}

	hp.l.PutConn(wrapConn)
	return
}

func (hp *HttpProxy) Close() error {
	hp.s.Close()
	hp.l.Close()
	return nil
}

func (hp *HttpProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if ok := hp.Auth(rw, req); !ok {
		rw.Header().Set("Proxy-Authenticate", "Basic")
		rw.WriteHeader(http.StatusProxyAuthRequired)
		return
	}

	if req.Method == "CONNECT" {
		hp.ConnectHandler(rw, req)
	} else {
		hp.HttpHandler(rw, req)
	}
}

func (hp *HttpProxy) HttpHandler(rw http.ResponseWriter, req *http.Request) {
	removeProxyHeaders(req)

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	copyHeaders(rw.Header(), resp.Header)
	rw.WriteHeader(resp.StatusCode)

	_, err = io.Copy(rw, resp.Body)
	if err != nil && err != io.EOF {
		return
	}
}

func (hp *HttpProxy) ConnectHandler(rw http.ResponseWriter, req *http.Request) {
	hj, ok := rw.(http.Hijacker)
	if !ok {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	client, _, err := hj.Hijack()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	remote, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		http.Error(rw, "Failed", http.StatusBadRequest)
		client.Close()
		return
	}
	client.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))

	go tcp.Join(remote, client)
}

func (hp *HttpProxy) Auth(rw http.ResponseWriter, req *http.Request) bool {
	if hp.AuthUser == "" && hp.AuthPasswd == "" {
		return true
	}

	s := strings.SplitN(req.Header.Get("Proxy-Authorization"), " ", 2)
	if len(s) != 2 {
		return false
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return false
	}

	if pair[0] != hp.AuthUser || pair[1] != hp.AuthPasswd {
		return false
	}
	return true
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func removeProxyHeaders(req *http.Request) {
	req.RequestURI = ""
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Connection")
	req.Header.Del("Proxy-Authenticate")
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("TE")
	req.Header.Del("Trailers")
	req.Header.Del("Transfer-Encoding")
	req.Header.Del("Upgrade")
}

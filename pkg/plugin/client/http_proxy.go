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

//go:build !frps

package plugin

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	libio "github.com/fatedier/golib/io"
	libnet "github.com/fatedier/golib/net"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/util"
)

func init() {
	Register(v1.PluginHTTPProxy, NewHTTPProxyPlugin)
}

type HTTPProxy struct {
	opts *v1.HTTPProxyPluginOptions

	l *Listener
	s *http.Server
}

func NewHTTPProxyPlugin(options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.HTTPProxyPluginOptions)
	listener := NewProxyListener()

	hp := &HTTPProxy{
		l:    listener,
		opts: opts,
	}

	hp.s = &http.Server{
		Handler:           hp,
		ReadHeaderTimeout: 60 * time.Second,
	}

	go func() {
		_ = hp.s.Serve(listener)
	}()
	return hp, nil
}

func (hp *HTTPProxy) Name() string {
	return v1.PluginHTTPProxy
}

func (hp *HTTPProxy) Handle(_ context.Context, conn io.ReadWriteCloser, realConn net.Conn, _ *ExtraInfo) {
	wrapConn := netpkg.WrapReadWriteCloserToConn(conn, realConn)

	sc, rd := libnet.NewSharedConn(wrapConn)
	firstBytes := make([]byte, 7)
	_, err := rd.Read(firstBytes)
	if err != nil {
		wrapConn.Close()
		return
	}

	if strings.ToUpper(string(firstBytes)) == "CONNECT" {
		bufRd := bufio.NewReader(sc)
		request, err := http.ReadRequest(bufRd)
		if err != nil {
			wrapConn.Close()
			return
		}
		hp.handleConnectReq(request, libio.WrapReadWriteCloser(bufRd, wrapConn, wrapConn.Close))
		return
	}

	_ = hp.l.PutConn(sc)
}

func (hp *HTTPProxy) Close() error {
	hp.s.Close()
	hp.l.Close()
	return nil
}

func (hp *HTTPProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if ok := hp.Auth(req); !ok {
		rw.Header().Set("Proxy-Authenticate", "Basic")
		rw.WriteHeader(http.StatusProxyAuthRequired)
		return
	}

	if req.Method == http.MethodConnect {
		// deprecated
		// Connect request is handled in Handle function.
		hp.ConnectHandler(rw, req)
	} else {
		hp.HTTPHandler(rw, req)
	}
}

func (hp *HTTPProxy) HTTPHandler(rw http.ResponseWriter, req *http.Request) {
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

// deprecated
// Hijack needs to SetReadDeadline on the Conn of the request, but if we use stream compression here,
// we may always get i/o timeout error.
func (hp *HTTPProxy) ConnectHandler(rw http.ResponseWriter, req *http.Request) {
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
	_, _ = client.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

	go libio.Join(remote, client)
}

func (hp *HTTPProxy) Auth(req *http.Request) bool {
	if hp.opts.HTTPUser == "" && hp.opts.HTTPPassword == "" {
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

	if !util.ConstantTimeEqString(pair[0], hp.opts.HTTPUser) ||
		!util.ConstantTimeEqString(pair[1], hp.opts.HTTPPassword) {
		time.Sleep(200 * time.Millisecond)
		return false
	}
	return true
}

func (hp *HTTPProxy) handleConnectReq(req *http.Request, rwc io.ReadWriteCloser) {
	defer rwc.Close()
	if ok := hp.Auth(req); !ok {
		res := getBadResponse()
		_ = res.Write(rwc)
		if res.Body != nil {
			res.Body.Close()
		}
		return
	}

	remote, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		res := &http.Response{
			StatusCode: 400,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
		}
		_ = res.Write(rwc)
		return
	}
	_, _ = rwc.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

	libio.Join(remote, rwc)
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

func getBadResponse() *http.Response {
	header := make(map[string][]string)
	header["Proxy-Authenticate"] = []string{"Basic"}
	header["Connection"] = []string{"close"}
	res := &http.Response{
		Status:     "407 Not authorized",
		StatusCode: 407,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
	}
	return res
}

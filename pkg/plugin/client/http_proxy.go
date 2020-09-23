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
	"bufio"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"strings"

	frpNet "github.com/fatedier/frp/pkg/util/net"

	frpIo "github.com/fatedier/golib/io"
	gnet "github.com/fatedier/golib/net"
)

const PluginHTTPProxy = "http_proxy"

func init() {
	Register(PluginHTTPProxy, NewHTTPProxyPlugin)
}

type HTTPProxy struct {
	l          *Listener
	s          *http.Server
	AuthUser   string
	AuthPasswd string
}

func NewHTTPProxyPlugin(params map[string]string) (Plugin, error) {
	user := params["plugin_http_user"]
	passwd := params["plugin_http_passwd"]
	listener := NewProxyListener()

	hp := &HTTPProxy{
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

func (hp *HTTPProxy) Name() string {
	return PluginHTTPProxy
}

func (hp *HTTPProxy) Handle(conn io.ReadWriteCloser, realConn net.Conn, extraBufToLocal []byte) {
	wrapConn := frpNet.WrapReadWriteCloserToConn(conn, realConn)

	sc, rd := gnet.NewSharedConn(wrapConn)
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
		hp.handleConnectReq(request, frpIo.WrapReadWriteCloser(bufRd, wrapConn, wrapConn.Close))
		return
	}

	hp.l.PutConn(sc)
	return
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
	client.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

	go frpIo.Join(remote, client)
}

func (hp *HTTPProxy) Auth(req *http.Request) bool {
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

func (hp *HTTPProxy) handleConnectReq(req *http.Request, rwc io.ReadWriteCloser) {
	defer rwc.Close()
	if ok := hp.Auth(req); !ok {
		res := getBadResponse()
		res.Write(rwc)
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
		res.Write(rwc)
		return
	}
	rwc.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

	frpIo.Join(remote, rwc)
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

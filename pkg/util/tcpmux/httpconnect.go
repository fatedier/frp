// Copyright 2020 guylewin, guy@lewin.co.il
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

package tcpmux

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	libnet "github.com/fatedier/golib/net"

	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/vhost"
)

type HTTPConnectTCPMuxer struct {
	*vhost.Muxer

	// If passthrough is set to true, the CONNECT request will be forwarded to the backend service.
	// Otherwise, it will return an OK response to the client and forward the remaining content to the backend service.
	passthrough bool
}

func NewHTTPConnectTCPMuxer(listener net.Listener, passthrough bool, timeout time.Duration) (*HTTPConnectTCPMuxer, error) {
	ret := &HTTPConnectTCPMuxer{passthrough: passthrough}
	mux, err := vhost.NewMuxer(listener, ret.getHostFromHTTPConnect, timeout)
	if err != nil {
		return nil, err
	}
	mux.SetCheckAuthFunc(ret.auth).
		SetSuccessHookFunc(ret.sendConnectResponse).
		SetFailHookFunc(vhostFailed)
	ret.Muxer = mux
	return ret, nil
}

func (muxer *HTTPConnectTCPMuxer) readHTTPConnectRequest(bufioReader *bufio.Reader) (host, httpUser, httpPwd string, err error) {
	req, err := http.ReadRequest(bufioReader)
	if err != nil {
		return
	}

	if req.Method != "CONNECT" {
		err = fmt.Errorf("connections to tcp vhost must be of method CONNECT")
		return
	}

	host, _ = httppkg.CanonicalHost(req.Host)
	proxyAuth := req.Header.Get("Proxy-Authorization")
	if proxyAuth != "" {
		httpUser, httpPwd, _ = httppkg.ParseBasicAuth(proxyAuth)
	}
	return
}

func (muxer *HTTPConnectTCPMuxer) sendConnectResponse(c net.Conn, _ map[string]string) error {
	if muxer.passthrough {
		return nil
	}
	res := httppkg.OkResponse()
	if res.Body != nil {
		defer res.Body.Close()
	}
	return res.Write(c)
}

func (muxer *HTTPConnectTCPMuxer) auth(c net.Conn, username, password string, reqInfo map[string]string) (bool, error) {
	reqUsername := reqInfo["HTTPUser"]
	reqPassword := reqInfo["HTTPPwd"]
	if username == reqUsername && password == reqPassword {
		return true, nil
	}

	resp := httppkg.ProxyUnauthorizedResponse()
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	_ = resp.Write(c)
	return false, nil
}

func vhostFailed(c net.Conn) {
	res := vhost.NotFoundResponse()
	if res.Body != nil {
		defer res.Body.Close()
	}
	_ = res.Write(c)
	_ = c.Close()
}

func (muxer *HTTPConnectTCPMuxer) getHostFromHTTPConnect(c net.Conn) (net.Conn, map[string]string, error) {
	reqInfoMap := make(map[string]string, 0)
	sc, rd := libnet.NewSharedConn(c)

	bufioReader := bufio.NewReader(rd)
	host, httpUser, httpPwd, err := muxer.readHTTPConnectRequest(bufioReader)
	if err != nil {
		return nil, reqInfoMap, err
	}

	reqInfoMap["Host"] = host
	reqInfoMap["Scheme"] = "tcp"
	reqInfoMap["HTTPUser"] = httpUser
	reqInfoMap["HTTPPwd"] = httpPwd

	if !muxer.passthrough {
		// Drop the CONNECT request but keep tunnel data that was read along with it.
		buffered, _ := bufioReader.Peek(bufioReader.Buffered())
		if err := sc.ResetBuf(buffered); err != nil {
			return nil, reqInfoMap, err
		}
	}
	return sc, reqInfoMap, nil
}

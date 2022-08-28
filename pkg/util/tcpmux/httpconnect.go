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
	"io"
	"net"
	"net/http"
	"time"

	gnet "github.com/fatedier/golib/net"

	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/vhost"
)

type HTTPConnectTCPMuxer struct {
	*vhost.Muxer

	passthrough  bool
	authRequired bool // Not supported until we really need this.
}

func NewHTTPConnectTCPMuxer(listener net.Listener, passthrough bool, timeout time.Duration) (*HTTPConnectTCPMuxer, error) {
	ret := &HTTPConnectTCPMuxer{passthrough: passthrough, authRequired: false}
	mux, err := vhost.NewMuxer(listener, ret.getHostFromHTTPConnect, nil, ret.sendConnectResponse, nil, timeout)
	ret.Muxer = mux
	return ret, err
}

func (muxer *HTTPConnectTCPMuxer) readHTTPConnectRequest(rd io.Reader) (host string, httpUser string, err error) {
	bufioReader := bufio.NewReader(rd)

	req, err := http.ReadRequest(bufioReader)
	if err != nil {
		return
	}

	if req.Method != "CONNECT" {
		err = fmt.Errorf("connections to tcp vhost must be of method CONNECT")
		return
	}

	host, _ = util.CanonicalHost(req.Host)
	proxyAuth := req.Header.Get("Proxy-Authorization")
	if proxyAuth != "" {
		httpUser, _, _ = util.ParseBasicAuth(proxyAuth)
	}
	return
}

func (muxer *HTTPConnectTCPMuxer) sendConnectResponse(c net.Conn, reqInfo map[string]string) error {
	if muxer.passthrough {
		return nil
	}
	res := util.OkResponse()
	if res.Body != nil {
		defer res.Body.Close()
	}
	return res.Write(c)
}

func (muxer *HTTPConnectTCPMuxer) getHostFromHTTPConnect(c net.Conn) (net.Conn, map[string]string, error) {
	reqInfoMap := make(map[string]string, 0)
	sc, rd := gnet.NewSharedConn(c)

	host, httpUser, err := muxer.readHTTPConnectRequest(rd)
	if err != nil {
		return nil, reqInfoMap, err
	}

	reqInfoMap["Host"] = host
	reqInfoMap["Scheme"] = "tcp"
	reqInfoMap["HTTPUser"] = httpUser

	outConn := c
	if muxer.passthrough {
		outConn = sc
		if muxer.authRequired && httpUser == "" {
			resp := util.ProxyUnauthorizedResponse()
			if resp.Body != nil {
				defer resp.Body.Close()
			}
			_ = resp.Write(c)
			outConn = c
		}
	}
	return outConn, reqInfoMap, nil
}

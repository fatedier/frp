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

	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/vhost"
)

type HTTPConnectTCPMuxer struct {
	*vhost.Muxer
}

func NewHTTPConnectTCPMuxer(listener net.Listener, timeout time.Duration) (*HTTPConnectTCPMuxer, error) {
	mux, err := vhost.NewMuxer(listener, getHostFromHTTPConnect, nil, sendHTTPOk, nil, timeout)
	return &HTTPConnectTCPMuxer{mux}, err
}

func readHTTPConnectRequest(rd io.Reader) (host string, err error) {
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
	return
}

func sendHTTPOk(c net.Conn) error {
	return util.OkResponse().Write(c)
}

func getHostFromHTTPConnect(c net.Conn) (_ net.Conn, _ map[string]string, err error) {
	reqInfoMap := make(map[string]string, 0)
	host, err := readHTTPConnectRequest(c)
	if err != nil {
		return nil, reqInfoMap, err
	}
	reqInfoMap["Host"] = host
	reqInfoMap["Scheme"] = "tcp"
	return c, reqInfoMap, nil
}

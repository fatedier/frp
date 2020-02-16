// Copyright 2019 guylewin, guy@lewin.co.il
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

package vhost

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type TcpMuxer struct {
	*VhostMuxer
}

func NewTcpMuxer(listener net.Listener, timeout time.Duration) (*TcpMuxer, error) {
	mux, err := NewVhostMuxer(listener, getTcpServiceName, nil, sendHttpOk, timeout)
	return &TcpMuxer{mux}, err
}

func readHttpConnectRequest(rd io.Reader) (host string, err error) {
	bufioReader := bufio.NewReader(rd)

	req, err := http.ReadRequest(bufioReader)
	if err != nil {
		return
	}

	if req.Method != "CONNECT" {
		err = fmt.Errorf("connections to tcp vhost must be of method CONNECT")
		return
	}

	host = getHostFromAddr(req.Host)
	return
}

func sendHttpOk(c net.Conn, _ string) (_ net.Conn, err error) {
	okResp := okResponse()
	err = okResp.Write(c)
	return c, err
}

func getTcpServiceName(c net.Conn) (_ net.Conn, _ map[string]string, err error) {
	reqInfoMap := make(map[string]string, 0)
	host, err := readHttpConnectRequest(c)
	if err != nil {
		return nil, reqInfoMap, err
	}
	reqInfoMap["Host"] = host
	reqInfoMap["Scheme"] = "tcp"
	return c, reqInfoMap, nil
}

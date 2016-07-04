// Copyright 2016 fatedier, fatedier@gmail.com
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
	"net"
	"net/http"
	"strings"
	"time"

	"frp/utils/conn"
)

type HttpMuxer struct {
	*VhostMuxer
}

func GetHttpHostname(c *conn.Conn) (_ net.Conn, routerName string, err error) {
	sc, rd := newShareConn(c.TcpConn)

	request, err := http.ReadRequest(bufio.NewReader(rd))
	if err != nil {
		return sc, "", err
	}
	tmpArr := strings.Split(request.Host, ":")
	routerName = tmpArr[0]
	request.Body.Close()
	return sc, routerName, nil
}

func NewHttpMuxer(listener *conn.Listener, timeout time.Duration) (*HttpMuxer, error) {
	mux, err := NewVhostMuxer(listener, GetHttpHostname, timeout)
	return &HttpMuxer{mux}, err
}

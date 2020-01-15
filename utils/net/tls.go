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

package net

import (
	"crypto/tls"
	"net"
	"time"

	gnet "github.com/fatedier/golib/net"
)

var (
	FRP_TLS_HEAD_BYTE = 0x17
)

func WrapTLSClientConn(c net.Conn, tlsConfig *tls.Config) (out net.Conn) {
	c.Write([]byte{byte(FRP_TLS_HEAD_BYTE)})
	out = tls.Client(c, tlsConfig)
	return
}

func CheckAndEnableTLSServerConnWithTimeout(c net.Conn, tlsConfig *tls.Config, timeout time.Duration) (out net.Conn, err error) {
	sc, r := gnet.NewSharedConnSize(c, 2)
	buf := make([]byte, 1)
	var n int
	c.SetReadDeadline(time.Now().Add(timeout))
	n, err = r.Read(buf)
	c.SetReadDeadline(time.Time{})
	if err != nil {
		return
	}

	if n == 1 && int(buf[0]) == FRP_TLS_HEAD_BYTE {
		out = tls.Server(c, tlsConfig)
	} else {
		out = sc
	}
	return
}

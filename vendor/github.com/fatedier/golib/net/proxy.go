// Copyright 2018 fatedier, fatedier@gmail.com
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
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"
)

type ProxyAuth struct {
	Enable   bool
	Username string
	Passwd   string
}

func DialTcpByProxy(proxyStr string, addr string) (c net.Conn, err error) {
	if proxyStr == "" {
		return net.Dial("tcp", addr)
	}

	var proxyUrl *url.URL
	if proxyUrl, err = url.Parse(proxyStr); err != nil {
		return
	}

	auth := &ProxyAuth{}
	if proxyUrl.User != nil {
		auth.Enable = true
		auth.Username = proxyUrl.User.Username()
		auth.Passwd, _ = proxyUrl.User.Password()
	}

	switch proxyUrl.Scheme {
	case "http":
		return DialTcpByHttpProxy(proxyUrl.Host, addr, auth)
	case "socks5":
		return DialTcpBySocks5Proxy(proxyUrl.Host, addr, auth)
	default:
		err = fmt.Errorf("Proxy URL scheme must be http or socks5, not [%s]", proxyUrl.Scheme)
		return
	}
}

func DialTcpByHttpProxy(proxyHost string, dstAddr string, auth *ProxyAuth) (c net.Conn, err error) {
	if c, err = net.Dial("tcp", proxyHost); err != nil {
		return
	}

	req, err := http.NewRequest("CONNECT", "http://"+dstAddr, nil)
	if err != nil {
		return
	}
	if auth.Enable {
		req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth.Username+":"+auth.Passwd)))
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Write(c)

	resp, err := http.ReadResponse(bufio.NewReader(c), req)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("DialTcpByHttpProxy error, StatusCode [%d]", resp.StatusCode)
		return
	}
	return
}

func DialTcpBySocks5Proxy(proxyHost string, dstAddr string, auth *ProxyAuth) (c net.Conn, err error) {
	var s5Auth *proxy.Auth
	if auth.Enable {
		s5Auth = &proxy.Auth{
			User:     auth.Username,
			Password: auth.Passwd,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", proxyHost, s5Auth, nil)
	if err != nil {
		return nil, err
	}

	if c, err = dialer.Dial("tcp", dstAddr); err != nil {
		return
	}
	return
}

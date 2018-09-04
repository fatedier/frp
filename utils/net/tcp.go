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

package net

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/fatedier/frp/utils/log"
)

type TcpListener struct {
	net.Addr
	listener  net.Listener
	accept    chan Conn
	closeFlag bool
	log.Logger
}

func ListenTcp(bindAddr string, bindPort int) (l *TcpListener, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", bindAddr, bindPort))
	if err != nil {
		return l, err
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return l, err
	}

	l = &TcpListener{
		Addr:      listener.Addr(),
		listener:  listener,
		accept:    make(chan Conn),
		closeFlag: false,
		Logger:    log.NewPrefixLogger(""),
	}

	go func() {
		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				if l.closeFlag {
					close(l.accept)
					return
				}
				continue
			}

			c := NewTcpConn(conn)
			l.accept <- c
		}
	}()
	return l, err
}

// Wait util get one new connection or listener is closed
// if listener is closed, err returned.
func (l *TcpListener) Accept() (Conn, error) {
	conn, ok := <-l.accept
	if !ok {
		return conn, fmt.Errorf("channel for tcp listener closed")
	}
	return conn, nil
}

func (l *TcpListener) Close() error {
	if !l.closeFlag {
		l.closeFlag = true
		l.listener.Close()
	}
	return nil
}

// Wrap for TCPConn.
type TcpConn struct {
	net.Conn
	log.Logger
}

func NewTcpConn(conn *net.TCPConn) (c *TcpConn) {
	c = &TcpConn{
		Conn:   conn,
		Logger: log.NewPrefixLogger(""),
	}
	return
}

func ConnectTcpServer(addr string) (c Conn, err error) {
	servertAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}
	conn, err := net.DialTCP("tcp", nil, servertAddr)
	if err != nil {
		return
	}
	c = NewTcpConn(conn)
	return
}

// ConnectTcpServerByHttpProxy try to connect remote server by http proxy.
// If httpProxy is empty, it will connect server directly.
func ConnectTcpServerByHttpProxy(httpProxy string, serverAddr string) (c Conn, err error) {
	if httpProxy == "" {
		return ConnectTcpServer(serverAddr)
	}

	var proxyUrl *url.URL
	if proxyUrl, err = url.Parse(httpProxy); err != nil {
		return
	}

	var proxyAuth string
	if proxyUrl.User != nil {
		username := proxyUrl.User.Username()
		passwd, _ := proxyUrl.User.Password()
		proxyAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+passwd))
	}

	if proxyUrl.Scheme != "http" {
		err = fmt.Errorf("Proxy URL scheme must be http, not [%s]", proxyUrl.Scheme)
		return
	}

	if c, err = ConnectTcpServer(proxyUrl.Host); err != nil {
		return
	}

	req, err := http.NewRequest("CONNECT", "http://"+serverAddr, nil)
	if err != nil {
		return
	}
	if proxyAuth != "" {
		req.Header.Set("Proxy-Authorization", proxyAuth)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Write(c)

	resp, err := http.ReadResponse(bufio.NewReader(c), req)
	if err != nil {
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("ConnectTcpServer using proxy error, StatusCode [%d]", resp.StatusCode)
		return
	}

	return
}

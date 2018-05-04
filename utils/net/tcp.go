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

	"golang.org/x/net/proxy"
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

func NewTcpConn(conn net.Conn) (c *TcpConn) {
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

// ConnectTcpServerByProxy try to connect remote server by proxy.
func ConnectTcpServerByProxy(proxyStr string, serverAddr string) (c Conn, err error) {
	if proxyStr == "" {
		return ConnectTcpServer(serverAddr)
	}

	var (
		proxyUrl *url.URL
		username string
		passwd   string
	)
	if proxyUrl, err = url.Parse(proxyStr); err != nil {
		return
	}
	if proxyUrl.User != nil {
		username = proxyUrl.User.Username()
		passwd, _ = proxyUrl.User.Password()
	}

	switch proxyUrl.Scheme {
	case "http":
		return ConnectTcpServerByHttpProxy(proxyUrl, username, passwd, serverAddr)
	case "socks5":
		return ConnectTcpServerBySocks5Proxy(proxyUrl, username, passwd, serverAddr)
	default:
		err = fmt.Errorf("Proxy URL scheme must be http or socks5, not [%s]", proxyUrl.Scheme)
		return
	}
}

// ConnectTcpServerByHttpProxy try to connect remote server by http proxy.
func ConnectTcpServerByHttpProxy(proxyUrl *url.URL, user string, passwd string, serverAddr string) (c Conn, err error) {
	var proxyAuth string
	if proxyUrl.User != nil {
		proxyAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+passwd))
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

func ConnectTcpServerBySocks5Proxy(proxyUrl *url.URL, user string, passwd string, serverAddr string) (c Conn, err error) {
	var auth *proxy.Auth
	if proxyUrl.User != nil {
		auth = &proxy.Auth{
			User:     user,
			Password: passwd,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", proxyUrl.Host, auth, nil)
	if err != nil {
		return nil, err
	}

	var conn net.Conn
	if conn, err = dialer.Dial("tcp", serverAddr); err != nil {
		return
	}
	c = NewTcpConn(conn)
	return
}

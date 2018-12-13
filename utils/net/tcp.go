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
	"fmt"
	"net"

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

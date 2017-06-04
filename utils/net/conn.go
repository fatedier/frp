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
	"io"
	"net"
	"time"

	"github.com/fatedier/frp/utils/log"

	kcp "github.com/xtaci/kcp-go"
)

// Conn is the interface of connections used in frp.
type Conn interface {
	net.Conn
	log.Logger
}

type WrapLogConn struct {
	net.Conn
	log.Logger
}

func WrapConn(c net.Conn) Conn {
	return &WrapLogConn{
		Conn:   c,
		Logger: log.NewPrefixLogger(""),
	}
}

type WrapReadWriteCloserConn struct {
	io.ReadWriteCloser
	log.Logger
}

func (conn *WrapReadWriteCloserConn) LocalAddr() net.Addr {
	return (*net.TCPAddr)(nil)
}

func (conn *WrapReadWriteCloserConn) RemoteAddr() net.Addr {
	return (*net.TCPAddr)(nil)
}

func (conn *WrapReadWriteCloserConn) SetDeadline(t time.Time) error {
	return nil
}

func (conn *WrapReadWriteCloserConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (conn *WrapReadWriteCloserConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func WrapReadWriteCloserToConn(rwc io.ReadWriteCloser) Conn {
	return &WrapReadWriteCloserConn{
		ReadWriteCloser: rwc,
		Logger:          log.NewPrefixLogger(""),
	}
}

type Listener interface {
	Accept() (Conn, error)
	Close() error
	log.Logger
}

type LogListener struct {
	l net.Listener
	net.Listener
	log.Logger
}

func WrapLogListener(l net.Listener) Listener {
	return &LogListener{
		l:        l,
		Listener: l,
		Logger:   log.NewPrefixLogger(""),
	}
}

func (logL *LogListener) Accept() (Conn, error) {
	c, err := logL.l.Accept()
	return WrapConn(c), err
}

func ConnectServer(protocol string, addr string) (c Conn, err error) {
	switch protocol {
	case "tcp":
		return ConnectTcpServer(addr)
	case "kcp":
		kcpConn, errRet := kcp.DialWithOptions(addr, nil, 10, 3)
		if errRet != nil {
			err = errRet
			return
		}
		kcpConn.SetStreamMode(true)
		kcpConn.SetWriteDelay(true)
		kcpConn.SetNoDelay(1, 20, 2, 1)
		kcpConn.SetWindowSize(128, 512)
		kcpConn.SetMtu(1350)
		kcpConn.SetACKNoDelay(false)
		kcpConn.SetReadBuffer(4194304)
		kcpConn.SetWriteBuffer(4194304)
		c = WrapConn(kcpConn)
		return
	default:
		return nil, fmt.Errorf("unsupport protocol: %s", protocol)
	}
}

func ConnectServerByHttpProxy(httpProxy string, protocol string, addr string) (c Conn, err error) {
	switch protocol {
	case "tcp":
		return ConnectTcpServerByHttpProxy(httpProxy, addr)
	case "kcp":
		// http proxy is not supported for kcp
		return ConnectServer(protocol, addr)
	default:
		return nil, fmt.Errorf("unsupport protocol: %s", protocol)
	}
}

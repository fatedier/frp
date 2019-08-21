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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/fatedier/frp/utils/log"

	gnet "github.com/fatedier/golib/net"
	kcp "github.com/fatedier/kcp-go"
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

	underConn net.Conn
}

func WrapReadWriteCloserToConn(rwc io.ReadWriteCloser, underConn net.Conn) Conn {
	return &WrapReadWriteCloserConn{
		ReadWriteCloser: rwc,
		Logger:          log.NewPrefixLogger(""),
		underConn:       underConn,
	}
}

func (conn *WrapReadWriteCloserConn) LocalAddr() net.Addr {
	if conn.underConn != nil {
		return conn.underConn.LocalAddr()
	}
	return (*net.TCPAddr)(nil)
}

func (conn *WrapReadWriteCloserConn) RemoteAddr() net.Addr {
	if conn.underConn != nil {
		return conn.underConn.RemoteAddr()
	}
	return (*net.TCPAddr)(nil)
}

func (conn *WrapReadWriteCloserConn) SetDeadline(t time.Time) error {
	if conn.underConn != nil {
		return conn.underConn.SetDeadline(t)
	}
	return &net.OpError{Op: "set", Net: "wrap", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (conn *WrapReadWriteCloserConn) SetReadDeadline(t time.Time) error {
	if conn.underConn != nil {
		return conn.underConn.SetReadDeadline(t)
	}
	return &net.OpError{Op: "set", Net: "wrap", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (conn *WrapReadWriteCloserConn) SetWriteDeadline(t time.Time) error {
	if conn.underConn != nil {
		return conn.underConn.SetWriteDeadline(t)
	}
	return &net.OpError{Op: "set", Net: "wrap", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

type CloseNotifyConn struct {
	net.Conn
	log.Logger

	// 1 means closed
	closeFlag int32

	closeFn func()
}

// closeFn will be only called once
func WrapCloseNotifyConn(c net.Conn, closeFn func()) Conn {
	return &CloseNotifyConn{
		Conn:    c,
		Logger:  log.NewPrefixLogger(""),
		closeFn: closeFn,
	}
}

func (cc *CloseNotifyConn) Close() (err error) {
	pflag := atomic.SwapInt32(&cc.closeFlag, 1)
	if pflag == 0 {
		err = cc.Close()
		if cc.closeFn != nil {
			cc.closeFn()
		}
	}
	return
}

type StatsConn struct {
	Conn

	closed     int64 // 1 means closed
	totalRead  int64
	totalWrite int64
	statsFunc  func(totalRead, totalWrite int64)
}

func WrapStatsConn(conn Conn, statsFunc func(total, totalWrite int64)) *StatsConn {
	return &StatsConn{
		Conn:      conn,
		statsFunc: statsFunc,
	}
}

func (statsConn *StatsConn) Read(p []byte) (n int, err error) {
	n, err = statsConn.Conn.Read(p)
	statsConn.totalRead += int64(n)
	return
}

func (statsConn *StatsConn) Write(p []byte) (n int, err error) {
	n, err = statsConn.Conn.Write(p)
	statsConn.totalWrite += int64(n)
	return
}

func (statsConn *StatsConn) Close() (err error) {
	old := atomic.SwapInt64(&statsConn.closed, 1)
	if old != 1 {
		err = statsConn.Conn.Close()
		if statsConn.statsFunc != nil {
			statsConn.statsFunc(statsConn.totalRead, statsConn.totalWrite)
		}
	}
	return
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

func ConnectServerByProxy(proxyUrl string, protocol string, addr string) (c Conn, err error) {
	switch protocol {
	case "tcp":
		var conn net.Conn
		if conn, err = gnet.DialTcpByProxy(proxyUrl, addr); err != nil {
			return
		}
		return WrapConn(conn), nil
	case "kcp":
		// http proxy is not supported for kcp
		return ConnectServer(protocol, addr)
	case "websocket":
		return ConnectWebsocketServer(addr)
	default:
		return nil, fmt.Errorf("unsupport protocol: %s", protocol)
	}
}

func ConnectServerByProxyWithTLS(proxyUrl string, protocol string, addr string, tlsConfig *tls.Config) (c Conn, err error) {
	c, err = ConnectServerByProxy(proxyUrl, protocol, addr)
	if err != nil {
		return
	}

	if tlsConfig == nil {
		return
	}

	c = WrapTLSClientConn(c, tlsConfig)
	return
}

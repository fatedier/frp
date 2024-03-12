// Copyright 2017 fatedier, fatedier@gmail.com
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

	kcp "github.com/xtaci/kcp-go/v5"
)

type KCPListener struct {
	listener  net.Listener
	acceptCh  chan net.Conn
	closeFlag bool
}

func ListenKcp(address string) (l *KCPListener, err error) {
	listener, err := kcp.ListenWithOptions(address, nil, 10, 3)
	if err != nil {
		return l, err
	}
	_ = listener.SetReadBuffer(4194304)
	_ = listener.SetWriteBuffer(4194304)

	l = &KCPListener{
		listener:  listener,
		acceptCh:  make(chan net.Conn),
		closeFlag: false,
	}

	go func() {
		for {
			conn, err := listener.AcceptKCP()
			if err != nil {
				if l.closeFlag {
					close(l.acceptCh)
					return
				}
				continue
			}
			conn.SetStreamMode(true)
			conn.SetWriteDelay(true)
			conn.SetNoDelay(1, 20, 2, 1)
			conn.SetMtu(1350)
			conn.SetWindowSize(1024, 1024)
			conn.SetACKNoDelay(false)

			l.acceptCh <- conn
		}
	}()
	return l, err
}

func (l *KCPListener) Accept() (net.Conn, error) {
	conn, ok := <-l.acceptCh
	if !ok {
		return conn, fmt.Errorf("channel for kcp listener closed")
	}
	return conn, nil
}

func (l *KCPListener) Close() error {
	if !l.closeFlag {
		l.closeFlag = true
		l.listener.Close()
	}
	return nil
}

func (l *KCPListener) Addr() net.Addr {
	return l.listener.Addr()
}

func NewKCPConnFromUDP(conn *net.UDPConn, connected bool, raddr string) (net.Conn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", raddr)
	if err != nil {
		return nil, err
	}
	var pConn net.PacketConn = conn
	if connected {
		pConn = &ConnectedUDPConn{conn}
	}
	kcpConn, err := kcp.NewConn3(1, udpAddr, nil, 10, 3, pConn)
	if err != nil {
		return nil, err
	}
	kcpConn.SetStreamMode(true)
	kcpConn.SetWriteDelay(true)
	kcpConn.SetNoDelay(1, 20, 2, 1)
	kcpConn.SetMtu(1350)
	kcpConn.SetWindowSize(1024, 1024)
	kcpConn.SetACKNoDelay(false)
	return kcpConn, nil
}

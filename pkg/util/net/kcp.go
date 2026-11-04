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
	"errors"
	"net"
	"sync"

	kcp "github.com/xtaci/kcp-go/v5"
)

var ErrKCPListenerClosed = errors.New("kcp listener closed")

type KCPListener struct {
	listener net.Listener
	acceptCh chan net.Conn
	done     chan struct{}

	closeOnce sync.Once
}

func ListenKcp(address string) (l *KCPListener, err error) {
	listener, err := kcp.ListenWithOptions(address, nil, 10, 3)
	if err != nil {
		return l, err
	}
	_ = listener.SetReadBuffer(4194304)
	_ = listener.SetWriteBuffer(4194304)

	l = &KCPListener{
		listener: listener,
		acceptCh: make(chan net.Conn),
		done:     make(chan struct{}),
	}

	go func() {
		for {
			conn, err := listener.AcceptKCP()
			if err != nil {
				select {
				case <-l.done:
					return
				default:
					continue
				}
			}
			conn.SetStreamMode(true)
			conn.SetWriteDelay(true)
			conn.SetNoDelay(1, 20, 2, 1)
			conn.SetMtu(1350)
			conn.SetWindowSize(1024, 1024)
			conn.SetACKNoDelay(false)

			select {
			case <-l.done:
				conn.Close()
				return
			case l.acceptCh <- conn:
				// A ready handoff may win when Close races with it. Once delivered,
				// the connection is owned by the caller of Accept.
			}
		}
	}()
	return l, err
}

func (l *KCPListener) Accept() (net.Conn, error) {
	select {
	case <-l.done:
		return nil, ErrKCPListenerClosed
	default:
	}
	select {
	case <-l.done:
		return nil, ErrKCPListenerClosed
	case conn := <-l.acceptCh:
		return conn, nil
	}
}

func (l *KCPListener) Close() error {
	l.closeOnce.Do(func() {
		close(l.done)
		l.listener.Close()
	})
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

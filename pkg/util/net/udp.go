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
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/fatedier/golib/pool"
)

type UDPPacket struct {
	Buf        []byte
	LocalAddr  net.Addr
	RemoteAddr net.Addr
}

type FakeUDPConn struct {
	l *UDPListener

	localAddr  net.Addr
	remoteAddr net.Addr
	packets    chan []byte
	closeFlag  bool

	lastActive time.Time
	mu         sync.RWMutex
}

func NewFakeUDPConn(l *UDPListener, laddr, raddr net.Addr) *FakeUDPConn {
	fc := &FakeUDPConn{
		l:          l,
		localAddr:  laddr,
		remoteAddr: raddr,
		packets:    make(chan []byte, 20),
	}

	go func() {
		for {
			time.Sleep(5 * time.Second)
			fc.mu.RLock()
			if time.Since(fc.lastActive) > 10*time.Second {
				fc.mu.RUnlock()
				fc.Close()
				break
			}
			fc.mu.RUnlock()
		}
	}()
	return fc
}

func (c *FakeUDPConn) putPacket(content []byte) {
	defer func() {
		_ = recover()
	}()

	select {
	case c.packets <- content:
	default:
	}
}

func (c *FakeUDPConn) Read(b []byte) (n int, err error) {
	content, ok := <-c.packets
	if !ok {
		return 0, io.EOF
	}
	c.mu.Lock()
	c.lastActive = time.Now()
	c.mu.Unlock()

	if len(b) < len(content) {
		n = len(b)
	} else {
		n = len(content)
	}
	copy(b, content)
	return n, nil
}

func (c *FakeUDPConn) Write(b []byte) (n int, err error) {
	c.mu.RLock()
	if c.closeFlag {
		c.mu.RUnlock()
		return 0, io.ErrClosedPipe
	}
	c.mu.RUnlock()

	packet := &UDPPacket{
		Buf:        b,
		LocalAddr:  c.localAddr,
		RemoteAddr: c.remoteAddr,
	}
	_ = c.l.writeUDPPacket(packet)

	c.mu.Lock()
	c.lastActive = time.Now()
	c.mu.Unlock()
	return len(b), nil
}

func (c *FakeUDPConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closeFlag {
		c.closeFlag = true
		close(c.packets)
	}
	return nil
}

func (c *FakeUDPConn) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closeFlag
}

func (c *FakeUDPConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *FakeUDPConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *FakeUDPConn) SetDeadline(_ time.Time) error {
	return nil
}

func (c *FakeUDPConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (c *FakeUDPConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

type UDPListener struct {
	addr      net.Addr
	acceptCh  chan net.Conn
	writeCh   chan *UDPPacket
	readConn  net.Conn
	closeFlag bool

	fakeConns map[string]*FakeUDPConn
}

func ListenUDP(bindAddr string, bindPort int) (l *UDPListener, err error) {
	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(bindAddr, strconv.Itoa(bindPort)))
	if err != nil {
		return l, err
	}
	readConn, err := net.ListenUDP("udp", udpAddr)

	l = &UDPListener{
		addr:      udpAddr,
		acceptCh:  make(chan net.Conn),
		writeCh:   make(chan *UDPPacket, 1000),
		fakeConns: make(map[string]*FakeUDPConn),
	}

	// for reading
	go func() {
		for {
			buf := pool.GetBuf(1450)
			n, remoteAddr, err := readConn.ReadFromUDP(buf)
			if err != nil {
				close(l.acceptCh)
				close(l.writeCh)
				return
			}

			fakeConn, exist := l.fakeConns[remoteAddr.String()]
			if !exist || fakeConn.IsClosed() {
				fakeConn = NewFakeUDPConn(l, l.Addr(), remoteAddr)
				l.fakeConns[remoteAddr.String()] = fakeConn
			}
			fakeConn.putPacket(buf[:n])

			l.acceptCh <- fakeConn
		}
	}()

	// for writing
	go func() {
		for {
			packet, ok := <-l.writeCh
			if !ok {
				return
			}

			if addr, ok := packet.RemoteAddr.(*net.UDPAddr); ok {
				_, _ = readConn.WriteToUDP(packet.Buf, addr)
			}
		}
	}()

	return
}

func (l *UDPListener) writeUDPPacket(packet *UDPPacket) (err error) {
	defer func() {
		if errRet := recover(); errRet != nil {
			err = fmt.Errorf("udp write closed listener")
		}
	}()
	l.writeCh <- packet
	return
}

func (l *UDPListener) WriteMsg(buf []byte, remoteAddr *net.UDPAddr) (err error) {
	// only set remote addr here
	packet := &UDPPacket{
		Buf:        buf,
		RemoteAddr: remoteAddr,
	}
	err = l.writeUDPPacket(packet)
	return
}

func (l *UDPListener) Accept() (net.Conn, error) {
	conn, ok := <-l.acceptCh
	if !ok {
		return conn, fmt.Errorf("channel for udp listener closed")
	}
	return conn, nil
}

func (l *UDPListener) Close() error {
	if !l.closeFlag {
		l.closeFlag = true
		if l.readConn != nil {
			l.readConn.Close()
		}
	}
	return nil
}

func (l *UDPListener) Addr() net.Addr {
	return l.addr
}

// ConnectedUDPConn is a wrapper for net.UDPConn which converts WriteTo syscalls
// to Write syscalls that are 4 times faster on some OS'es. This should only be
// used for connections that were produced by a net.Dial* call.
type ConnectedUDPConn struct{ *net.UDPConn }

// WriteTo redirects all writes to the Write syscall, which is 4 times faster.
func (c *ConnectedUDPConn) WriteTo(b []byte, _ net.Addr) (int, error) { return c.Write(b) }

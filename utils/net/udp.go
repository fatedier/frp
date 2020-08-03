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
	"sync"
	"time"

	"github.com/fatedier/golib/pool"
)

type UdpPacket struct {
	Buf        []byte
	LocalAddr  net.Addr
	RemoteAddr net.Addr
}

type FakeUdpConn struct {
	l *UdpListener

	localAddr  net.Addr
	remoteAddr net.Addr
	packets    chan []byte
	closeFlag  bool

	lastActive time.Time
	mu         sync.RWMutex
}

func NewFakeUdpConn(l *UdpListener, laddr, raddr net.Addr) *FakeUdpConn {
	fc := &FakeUdpConn{
		l:          l,
		localAddr:  laddr,
		remoteAddr: raddr,
		packets:    make(chan []byte, 20),
	}

	go func() {
		for {
			time.Sleep(5 * time.Second)
			fc.mu.RLock()
			if time.Now().Sub(fc.lastActive) > 10*time.Second {
				fc.mu.RUnlock()
				fc.Close()
				break
			}
			fc.mu.RUnlock()
		}
	}()
	return fc
}

func (c *FakeUdpConn) putPacket(content []byte) {
	defer func() {
		if err := recover(); err != nil {
		}
	}()

	select {
	case c.packets <- content:
	default:
	}
}

func (c *FakeUdpConn) Read(b []byte) (n int, err error) {
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

func (c *FakeUdpConn) Write(b []byte) (n int, err error) {
	c.mu.RLock()
	if c.closeFlag {
		c.mu.RUnlock()
		return 0, io.ErrClosedPipe
	}
	c.mu.RUnlock()

	packet := &UdpPacket{
		Buf:        b,
		LocalAddr:  c.localAddr,
		RemoteAddr: c.remoteAddr,
	}
	c.l.writeUdpPacket(packet)

	c.mu.Lock()
	c.lastActive = time.Now()
	c.mu.Unlock()
	return len(b), nil
}

func (c *FakeUdpConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closeFlag {
		c.closeFlag = true
		close(c.packets)
	}
	return nil
}

func (c *FakeUdpConn) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closeFlag
}

func (c *FakeUdpConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *FakeUdpConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *FakeUdpConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *FakeUdpConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *FakeUdpConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type UdpListener struct {
	addr      net.Addr
	acceptCh  chan net.Conn
	writeCh   chan *UdpPacket
	readConn  net.Conn
	closeFlag bool

	fakeConns map[string]*FakeUdpConn
}

func ListenUDP(bindAddr string, bindPort int) (l *UdpListener, err error) {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", bindAddr, bindPort))
	if err != nil {
		return l, err
	}
	readConn, err := net.ListenUDP("udp", udpAddr)

	l = &UdpListener{
		addr:      udpAddr,
		acceptCh:  make(chan net.Conn),
		writeCh:   make(chan *UdpPacket, 1000),
		fakeConns: make(map[string]*FakeUdpConn),
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
				fakeConn = NewFakeUdpConn(l, l.Addr(), remoteAddr)
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
				readConn.WriteToUDP(packet.Buf, addr)
			}
		}
	}()

	return
}

func (l *UdpListener) writeUdpPacket(packet *UdpPacket) (err error) {
	defer func() {
		if errRet := recover(); errRet != nil {
			err = fmt.Errorf("udp write closed listener")
		}
	}()
	l.writeCh <- packet
	return
}

func (l *UdpListener) WriteMsg(buf []byte, remoteAddr *net.UDPAddr) (err error) {
	// only set remote addr here
	packet := &UdpPacket{
		Buf:        buf,
		RemoteAddr: remoteAddr,
	}
	err = l.writeUdpPacket(packet)
	return
}

func (l *UdpListener) Accept() (net.Conn, error) {
	conn, ok := <-l.acceptCh
	if !ok {
		return conn, fmt.Errorf("channel for udp listener closed")
	}
	return conn, nil
}

func (l *UdpListener) Close() error {
	if !l.closeFlag {
		l.closeFlag = true
		l.readConn.Close()
	}
	return nil
}

func (l *UdpListener) Addr() net.Addr {
	return l.addr
}

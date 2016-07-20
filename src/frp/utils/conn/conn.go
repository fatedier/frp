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

package conn

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

type Listener struct {
	addr      net.Addr
	l         *net.TCPListener
	accept    chan *Conn
	closeFlag bool
}

func Listen(bindAddr string, bindPort int64) (l *Listener, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", bindAddr, bindPort))
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return l, err
	}

	l = &Listener{
		addr:      listener.Addr(),
		l:         listener,
		accept:    make(chan *Conn),
		closeFlag: false,
	}

	go func() {
		for {
			conn, err := l.l.AcceptTCP()
			if err != nil {
				if l.closeFlag {
					return
				}
				continue
			}

			c := &Conn{
				TcpConn:   conn,
				closeFlag: false,
			}
			c.Reader = bufio.NewReader(c.TcpConn)
			l.accept <- c
		}
	}()
	return l, err
}

// wait util get one new connection or listener is closed
// if listener is closed, err returned
func (l *Listener) Accept() (*Conn, error) {
	conn, ok := <-l.accept
	if !ok {
		return conn, fmt.Errorf("channel close")
	}
	return conn, nil
}

func (l *Listener) Close() error {
	if l.l != nil && l.closeFlag == false {
		l.closeFlag = true
		l.l.Close()
		close(l.accept)
	}
	return nil
}

// wrap for TCPConn
type Conn struct {
	TcpConn   net.Conn
	Reader    *bufio.Reader
	closeFlag bool
	mutex     sync.RWMutex
}

func NewConn(conn net.Conn) (c *Conn) {
	c = &Conn{}
	c.TcpConn = conn
	c.Reader = bufio.NewReader(c.TcpConn)
	c.closeFlag = false
	return c
}

func ConnectServer(host string, port int64) (c *Conn, err error) {
	c = &Conn{}
	servertAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return
	}
	conn, err := net.DialTCP("tcp", nil, servertAddr)
	if err != nil {
		return
	}
	c.TcpConn = conn
	c.Reader = bufio.NewReader(c.TcpConn)
	c.closeFlag = false
	return c, nil
}

func (c *Conn) GetRemoteAddr() (addr string) {
	return c.TcpConn.RemoteAddr().String()
}

func (c *Conn) GetLocalAddr() (addr string) {
	return c.TcpConn.LocalAddr().String()
}

func (c *Conn) Read(p []byte) (n int, err error) {
	n, err = c.Reader.Read(p)
	return
}

func (c *Conn) ReadLine() (buff string, err error) {
	buff, err = c.Reader.ReadString('\n')
	if err != nil {
		// wsarecv error in windows means connection closed
		if err == io.EOF || strings.Contains(err.Error(), "wsarecv: An existing connection was forcibly closed") {
			c.mutex.Lock()
			c.closeFlag = true
			c.mutex.Unlock()
		}
	}
	return buff, err
}

func (c *Conn) WriteBytes(content []byte) (n int, err error) {
	n, err = c.TcpConn.Write(content)
	return
}

func (c *Conn) Write(content string) (err error) {
	_, err = c.TcpConn.Write([]byte(content))
	return err
}

func (c *Conn) SetDeadline(t time.Time) error {
	err := c.TcpConn.SetDeadline(t)
	return err
}

func (c *Conn) Close() {
	c.mutex.Lock()
	if c.TcpConn != nil && c.closeFlag == false {
		c.closeFlag = true
		c.TcpConn.Close()
	}
	c.mutex.Unlock()
}

func (c *Conn) IsClosed() (closeFlag bool) {
	c.mutex.RLock()
	closeFlag = c.closeFlag
	c.mutex.RUnlock()
	return
}

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
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/fatedier/frp/src/utils/pool"
)

type Listener struct {
	addr      net.Addr
	l         *net.TCPListener
	accept    chan *Conn
	closeFlag bool
}

func Listen(bindAddr string, bindPort int64) (l *Listener, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", bindAddr, bindPort))
	if err != nil {
		return l, err
	}
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

			c := NewConn(conn)
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
	buffer    *bytes.Buffer
	closeFlag bool

	mutex sync.RWMutex
}

func NewConn(conn net.Conn) (c *Conn) {
	c = &Conn{
		TcpConn:   conn,
		buffer:    nil,
		closeFlag: false,
	}
	c.Reader = bufio.NewReader(c.TcpConn)
	return
}

func ConnectServer(addr string) (c *Conn, err error) {
	servertAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}
	conn, err := net.DialTCP("tcp", nil, servertAddr)
	if err != nil {
		return
	}
	c = NewConn(conn)
	return c, nil
}

func ConnectServerByHttpProxy(httpProxy string, serverAddr string) (c *Conn, err error) {
	var proxyUrl *url.URL
	if proxyUrl, err = url.Parse(httpProxy); err != nil {
		return
	}

	var proxyAuth string
	if proxyUrl.User != nil {
		proxyAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(proxyUrl.User.String()))
	}

	if proxyUrl.Scheme != "http" {
		err = fmt.Errorf("Proxy URL scheme must be http, not [%s]", proxyUrl.Scheme)
		return
	}

	if c, err = ConnectServer(proxyUrl.Host); err != nil {
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
	req.Write(c.TcpConn)

	resp, err := http.ReadResponse(bufio.NewReader(c), req)
	if err != nil {
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("ConnectServer using proxy error, StatusCode [%d]", resp.StatusCode)
		return
	}

	return
}

// if the tcpConn is different with c.TcpConn
// you should call c.Close() first
func (c *Conn) SetTcpConn(tcpConn net.Conn) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.TcpConn = tcpConn
	c.closeFlag = false
	c.Reader = bufio.NewReader(c.TcpConn)
}

func (c *Conn) GetRemoteAddr() (addr string) {
	return c.TcpConn.RemoteAddr().String()
}

func (c *Conn) GetLocalAddr() (addr string) {
	return c.TcpConn.LocalAddr().String()
}

func (c *Conn) Read(p []byte) (n int, err error) {
	c.mutex.RLock()
	if c.buffer == nil {
		c.mutex.RUnlock()
		return c.Reader.Read(p)
	}
	c.mutex.RUnlock()

	n, err = c.buffer.Read(p)
	if err == io.EOF {
		c.mutex.Lock()
		c.buffer = nil
		c.mutex.Unlock()
		var n2 int
		n2, err = c.Reader.Read(p[n:])

		n += n2
	}
	return
}

func (c *Conn) ReadLine() (buff string, err error) {
	buff, err = c.Reader.ReadString('\n')
	if err != nil {
		// wsarecv error in windows means connection closed?
		if err == io.EOF || strings.Contains(err.Error(), "wsarecv") {
			c.mutex.Lock()
			c.closeFlag = true
			c.mutex.Unlock()
		}
	}
	return buff, err
}

func (c *Conn) Write(content []byte) (n int, err error) {
	n, err = c.TcpConn.Write(content)
	return
}

func (c *Conn) WriteString(content string) (err error) {
	_, err = c.TcpConn.Write([]byte(content))
	return err
}

func (c *Conn) AppendReaderBuffer(content []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.buffer == nil {
		c.buffer = bytes.NewBuffer(make([]byte, 0, 2048))
	}
	c.buffer.Write(content)
}

func (c *Conn) SetDeadline(t time.Time) error {
	return c.TcpConn.SetDeadline(t)
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.TcpConn.SetReadDeadline(t)
}

func (c *Conn) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.TcpConn != nil && c.closeFlag == false {
		c.closeFlag = true
		c.TcpConn.Close()
	}
	return nil
}

func (c *Conn) IsClosed() (closeFlag bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	closeFlag = c.closeFlag
	return
}

// when you call this function, you should make sure that
// no bytes were read before
func (c *Conn) CheckClosed() bool {
	c.mutex.RLock()
	if c.closeFlag {
		c.mutex.RUnlock()
		return true
	}
	c.mutex.RUnlock()

	tmp := pool.GetBuf(2048)
	defer pool.PutBuf(tmp)
	err := c.TcpConn.SetReadDeadline(time.Now().Add(time.Millisecond))
	if err != nil {
		c.Close()
		return true
	}

	n, err := c.TcpConn.Read(tmp)
	if err == io.EOF {
		return true
	}

	var tmp2 []byte = make([]byte, 1)
	err = c.TcpConn.SetReadDeadline(time.Now().Add(time.Millisecond))
	if err != nil {
		c.Close()
		return true
	}

	n2, err := c.TcpConn.Read(tmp2)
	if err == io.EOF {
		return true
	}

	err = c.TcpConn.SetReadDeadline(time.Time{})
	if err != nil {
		c.Close()
		return true
	}

	if n > 0 {
		c.AppendReaderBuffer(tmp[:n])
	}
	if n2 > 0 {
		c.AppendReaderBuffer(tmp2[:n2])
	}
	return false
}

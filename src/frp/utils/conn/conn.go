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
	"sync"
	"time"

	"frp/utils/log"
	"frp/utils/pcrypto"
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

func (c *Conn) ReadLine() (buff string, err error) {
	buff, err = c.Reader.ReadString('\n')
	if err == io.EOF {
		c.closeFlag = true
	}
	return buff, err
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
	if c.TcpConn != nil && c.closeFlag == false {
		c.closeFlag = true
		c.TcpConn.Close()
	}
}

func (c *Conn) IsClosed() bool {
	return c.closeFlag
}

// will block until connection close
func Join(c1 *Conn, c2 *Conn) {
	var wait sync.WaitGroup
	pipe := func(to *Conn, from *Conn) {
		defer to.Close()
		defer from.Close()
		defer wait.Done()

		var err error
		_, err = io.Copy(to.TcpConn, from.TcpConn)
		if err != nil {
			log.Warn("join connections error, %v", err)
		}
	}

	wait.Add(2)
	go pipe(c1, c2)
	go pipe(c2, c1)
	wait.Wait()
	return
}

// messages from c1 to c2 will be encrypted
// and from c2 to c1 will be decrypted
func JoinMore(c1 *Conn, c2 *Conn, cryptKey string) {
	var wait sync.WaitGroup
	encryptPipe := func(from *Conn, to *Conn, key string) {
		defer from.Close()
		defer to.Close()
		defer wait.Done()

		// we don't care about errors here
		PipeEncrypt(from.TcpConn, to.TcpConn, key)
	}

	decryptPipe := func(to *Conn, from *Conn, key string) {
		defer from.Close()
		defer to.Close()
		defer wait.Done()

		// we don't care about errors here
		PipeDecrypt(to.TcpConn, from.TcpConn, key)
	}

	wait.Add(2)
	go encryptPipe(c1, c2, cryptKey)
	go decryptPipe(c2, c1, cryptKey)
	wait.Wait()
	log.Debug("One tunnel stopped")
	return
}

// decrypt msg from reader, then write into writer
func PipeDecrypt(r net.Conn, w net.Conn, key string) error {
	laes := new(pcrypto.Pcrypto)
	if err := laes.Init([]byte(key)); err != nil {
		log.Error("Pcrypto Init error: %v", err)
		return fmt.Errorf("Pcrypto Init error: %v", err)
	}

	nreader := bufio.NewReader(r)
	for {
		buf, err := nreader.ReadBytes('\n')
		if err != nil {
			return err
		}

		res, err := laes.Decrypt(buf)
		if err != nil {
			log.Error("Decrypt [%s] error, %v", string(buf), err)
			return fmt.Errorf("Decrypt [%s] error: %v", string(buf), err)
		}

		_, err = w.Write(res)
		if err != nil {
			return err
		}
	}
	return nil
}

// recvive msg from reader, then encrypt msg into write
func PipeEncrypt(r net.Conn, w net.Conn, key string) error {
	laes := new(pcrypto.Pcrypto)
	if err := laes.Init([]byte(key)); err != nil {
		log.Error("Pcrypto Init error: %v", err)
		return fmt.Errorf("Pcrypto Init error: %v", err)
	}

	nreader := bufio.NewReader(r)
	buf := make([]byte, 10*1024)

	for {
		n, err := nreader.Read(buf)
		if err != nil {
			return err
		}
		res, err := laes.Encrypt(buf[:n])
		if err != nil {
			log.Error("Encrypt error: %v", err)
			return fmt.Errorf("Encrypt error: %v", err)
		}

		res = append(res, '\n')
		_, err = w.Write(res)
		if err != nil {
			return err
		}
	}
	return nil
}

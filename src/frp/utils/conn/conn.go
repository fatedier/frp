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

	"frp/utils/log"
	"frp/utils/pcrypto"
)

type Listener struct {
	addr      net.Addr
	l         *net.TCPListener
	conns     chan *Conn
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
		conns:     make(chan *Conn),
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
			l.conns <- c
		}
	}()
	return l, err
}

// wait util get one new connection or listener is closed
// if listener is closed, err returned
func (l *Listener) GetConn() (conn *Conn, err error) {
	var ok bool
	conn, ok = <-l.conns
	if !ok {
		return conn, fmt.Errorf("channel close")
	}
	return conn, nil
}

func (l *Listener) Close() {
	if l.l != nil && l.closeFlag == false {
		l.closeFlag = true
		l.l.Close()
		close(l.conns)
	}
}

// wrap for TCPConn
type Conn struct {
	TcpConn   *net.TCPConn
	Reader    *bufio.Reader
	closeFlag bool
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
			log.Warn("join conns error, %v", err)
		}
	}

	wait.Add(2)
	go pipe(c1, c2)
	go pipe(c2, c1)
	wait.Wait()
	return
}

// decrypto msg from reader, then write into writer
func PipeDecryptoReader(r net.Conn, w net.Conn, key string) {
	defer r.Close()
	defer w.Close()

	laes := new(pcrypto.Pcrypto)
	if err := laes.Init([]byte(key)); err != nil {
		log.Error("Pcrypto Init error, [%v]", err)
		return
	}

	log.Debug("PipeDecryptoReader")

	buf := make([]byte, 10*1024)

	for {
		n, err := r.Read(buf)
		if err != nil {
			log.Error("Conn ReadLine error, [%v]", err)
			return
		}

		res, err := laes.Decrypto(buf[:n])
		if err != nil {
			log.Error("Decrypto error, [%s] [%s]", err, string(buf[:n]))
			return
		}

		_, err = w.Write(res)
		if err != nil {
			log.Error("net.Conn Write error, [%v]", err)
			return
		}
	}
}

// recvive msg from reader, then encrypto msg into write
func PipeEncryptoWriter(r net.Conn, w net.Conn, key string) {
	defer r.Close()
	defer w.Close()

	laes := new(pcrypto.Pcrypto)
	if err := laes.Init([]byte(key)); err != nil {
		log.Error("Pcrypto Init error, [%v]", err)
		return
	}

	log.Debug("PipeEncryptoWriter")

	buf := make([]byte, 10*1024)

	for {
		n, err := r.Read(buf)
		if err != nil {
			log.Error("Conn ReadLine error, [%v]", err)
			return
		}
		res, err := laes.Encrypto(buf[:n])
		if err != nil {
			log.Error("Encrypto error, [%v]", err)
			return
		}

		_, err = w.Write(res)
		if err != nil {
			log.Error("net.Conn Write error, [%v]", err)
			return
		}
	}
}

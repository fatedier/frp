package conn

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"

	"frp/pkg/utils/log"
)

type Listener struct {
	Addr  net.Addr
	Conns chan *Conn
}

// wait util get one
func (l *Listener) GetConn() (conn *Conn) {
	conn = <-l.Conns
	return conn
}

type Conn struct {
	TcpConn *net.TCPConn
	Reader  *bufio.Reader
}

func (c *Conn) ConnectServer(host string, port int64) (err error) {
	servertAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, servertAddr)
	if err != nil {
		return err
	}
	c.TcpConn = conn
	c.Reader = bufio.NewReader(c.TcpConn)
	return nil
}

func (c *Conn) GetRemoteAddr() (addr string) {
	return c.TcpConn.RemoteAddr().String()
}

func (c *Conn) GetLocalAddr() (addr string) {
	return c.TcpConn.LocalAddr().String()
}

func (c *Conn) ReadLine() (buff string, err error) {
	buff, err = c.Reader.ReadString('\n')
	return buff, err
}

func (c *Conn) Write(content string) (err error) {
	_, err = c.TcpConn.Write([]byte(content))
	return err
}

func (c *Conn) Close() {
	if c.TcpConn != nil { // ZWF:我觉得应该加一个非空保护
		c.TcpConn.Close()
	}
}

func Listen(bindAddr string, bindPort int64) (l *Listener, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", bindAddr, bindPort))
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return l, err
	}

	l = &Listener{
		Addr:  listener.Addr(),
		Conns: make(chan *Conn),
	}

	go func() {
		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				log.Error("Accept new tcp connection error, %v", err)
				continue
			}

			c := &Conn{
				TcpConn: conn,
			}
			c.Reader = bufio.NewReader(c.TcpConn)
			l.Conns <- c
		}
	}()
	return l, err
}

// will block until conn close
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

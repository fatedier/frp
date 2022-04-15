package streamserver

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"

	libnet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/test/e2e/pkg/rpc"
)

type Type string

const (
	TCP  Type = "tcp"
	UDP  Type = "udp"
	Unix Type = "unix"
)

type Server struct {
	netType     Type
	bindAddr    string
	bindPort    int
	respContent []byte

	handler func(net.Conn)

	l net.Listener
}

type Option func(*Server) *Server

func New(netType Type, options ...Option) *Server {
	s := &Server{
		netType:  netType,
		bindAddr: "127.0.0.1",
	}
	s.handler = s.handle

	for _, option := range options {
		s = option(s)
	}
	return s
}

func WithBindAddr(addr string) Option {
	return func(s *Server) *Server {
		s.bindAddr = addr
		return s
	}
}

func WithBindPort(port int) Option {
	return func(s *Server) *Server {
		s.bindPort = port
		return s
	}
}

func WithRespContent(content []byte) Option {
	return func(s *Server) *Server {
		s.respContent = content
		return s
	}
}

func WithCustomHandler(handler func(net.Conn)) Option {
	return func(s *Server) *Server {
		s.handler = handler
		return s
	}
}

func (s *Server) Run() error {
	if err := s.initListener(); err != nil {
		return err
	}

	go func() {
		for {
			c, err := s.l.Accept()
			if err != nil {
				return
			}
			go s.handler(c)
		}
	}()
	return nil
}

func (s *Server) Close() error {
	if s.l != nil {
		return s.l.Close()
	}
	return nil
}

func (s *Server) initListener() (err error) {
	switch s.netType {
	case TCP:
		s.l, err = net.Listen("tcp", net.JoinHostPort(s.bindAddr, strconv.Itoa(s.bindPort)))
	case UDP:
		s.l, err = libnet.ListenUDP(s.bindAddr, s.bindPort)
	case Unix:
		s.l, err = net.Listen("unix", s.bindAddr)
	default:
		return fmt.Errorf("unknown server type: %s", s.netType)
	}
	return err
}

func (s *Server) handle(c net.Conn) {
	defer c.Close()

	var reader io.Reader = c
	if s.netType == UDP {
		reader = bufio.NewReader(c)
	}
	for {
		buf, err := rpc.ReadBytes(reader)
		if err != nil {
			return
		}

		if len(s.respContent) > 0 {
			buf = s.respContent
		}
		rpc.WriteBytes(c, buf)
	}
}

func (s *Server) BindAddr() string {
	return s.bindAddr
}

func (s *Server) BindPort() int {
	return s.bindPort
}

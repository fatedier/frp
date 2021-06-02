package server

import (
	"fmt"
	"net"

	libnet "github.com/fatedier/frp/pkg/util/net"
)

type ServerType string

const (
	TCP  ServerType = "tcp"
	UDP  ServerType = "udp"
	Unix ServerType = "unix"
)

type Server struct {
	netType     ServerType
	bindAddr    string
	bindPort    int
	respContent []byte
	bufSize     int64

	echoMode bool

	l net.Listener
}

type Option func(*Server) *Server

func New(netType ServerType, options ...Option) *Server {
	s := &Server{
		netType:  netType,
		bindAddr: "127.0.0.1",
		bufSize:  2048,
	}

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

func WithBufSize(bufSize int64) Option {
	return func(s *Server) *Server {
		s.bufSize = bufSize
		return s
	}
}

func WithEchoMode(echoMode bool) Option {
	return func(s *Server) *Server {
		s.echoMode = echoMode
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
			go s.handle(c)
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
		s.l, err = net.Listen("tcp", fmt.Sprintf("%s:%d", s.bindAddr, s.bindPort))
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

	buf := make([]byte, s.bufSize)
	for {
		n, err := c.Read(buf)
		if err != nil {
			return
		}

		if s.echoMode {
			c.Write(buf[:n])
		} else {
			c.Write(s.respContent)
		}
	}
}

func (s *Server) BindAddr() string {
	return s.bindAddr
}

func (s *Server) BindPort() int {
	return s.bindPort
}

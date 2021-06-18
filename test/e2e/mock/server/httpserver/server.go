package httpserver

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
)

type Server struct {
	bindAddr string
	bindPort int
	hanlder  http.Handler

	l  net.Listener
	hs *http.Server
}

type Option func(*Server) *Server

func New(options ...Option) *Server {
	s := &Server{
		bindAddr: "127.0.0.1",
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

func WithHandler(h http.Handler) Option {
	return func(s *Server) *Server {
		s.hanlder = h
		return s
	}
}

func (s *Server) Run() error {
	if err := s.initListener(); err != nil {
		return err
	}

	addr := net.JoinHostPort(s.bindAddr, strconv.Itoa(s.bindPort))
	hs := &http.Server{
		Addr:    addr,
		Handler: s.hanlder,
	}
	s.hs = hs
	go hs.Serve(s.l)
	return nil
}

func (s *Server) Close() error {
	if s.hs != nil {
		return s.hs.Close()
	}
	return nil
}

func (s *Server) initListener() (err error) {
	s.l, err = net.Listen("tcp", fmt.Sprintf("%s:%d", s.bindAddr, s.bindPort))
	return
}

func (s *Server) BindAddr() string {
	return s.bindAddr
}

func (s *Server) BindPort() int {
	return s.bindPort
}

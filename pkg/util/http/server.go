// Copyright 2023 The frp Authors
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

package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/fatedier/frp/assets"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

var (
	defaultReadTimeout  = 60 * time.Second
	defaultWriteTimeout = 60 * time.Second
)

type Server struct {
	addr   string
	ln     net.Listener
	tlsCfg *tls.Config

	router *mux.Router
	hs     *http.Server

	authMiddleware mux.MiddlewareFunc
}

func NewServer(cfg v1.WebServerConfig) (*Server, error) {
	assets.Load(cfg.AssetsDir)

	s, err := newServer(cfg.Addr, cfg.Port, cfg.User, cfg.Password, cfg.TLS)
	if err != nil {
		return nil, err
	}
	if cfg.PprofEnable {
		s.registerPprofHandlers()
	}
	return s, nil
}

func newServer(bindAddr string, port int, user, password string, tlsConfig *v1.TLSConfig) (*Server, error) {
	addr := net.JoinHostPort(bindAddr, strconv.Itoa(port))
	if addr == ":" {
		addr = ":http"
	}

	router := mux.NewRouter()
	hs := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
	}
	s := &Server{
		addr:   addr,
		hs:     hs,
		router: router,
	}
	if tlsConfig != nil {
		cert, err := tls.LoadX509KeyPair(tlsConfig.CertFile, tlsConfig.KeyFile)
		if err != nil {
			return nil, err
		}
		s.tlsCfg = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}
	s.authMiddleware = netpkg.NewHTTPAuthMiddleware(user, password).SetAuthFailDelay(200 * time.Millisecond).Middleware

	// Bind only after all fallible preparation so an error cannot leak the listener.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s.ln = ln
	return s, nil
}

func (s *Server) Address() string {
	return s.addr
}

func (s *Server) Run() error {
	ln := s.ln
	if s.tlsCfg != nil {
		ln = tls.NewListener(ln, s.tlsCfg)
	}
	return s.hs.Serve(ln)
}

func (s *Server) Close() error {
	err := s.hs.Close()
	if s.ln != nil {
		_ = s.ln.Close()
	}
	return err
}

type RouterRegisterHelper struct {
	Router         *mux.Router
	AssetsFS       http.FileSystem
	AuthMiddleware mux.MiddlewareFunc
}

func (s *Server) RouteRegister(register func(helper *RouterRegisterHelper)) {
	register(&RouterRegisterHelper{
		Router:         s.router,
		AssetsFS:       assets.FileSystem,
		AuthMiddleware: s.authMiddleware,
	})
}

func (s *Server) registerPprofHandlers() {
	s.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	s.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	s.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	s.router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	s.router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
}

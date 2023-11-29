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

	addr := net.JoinHostPort(cfg.Addr, strconv.Itoa(cfg.Port))
	if addr == ":" {
		addr = ":http"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
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
		ln:     ln,
		hs:     hs,
		router: router,
	}
	if cfg.PprofEnable {
		s.registerPprofHandlers()
	}
	if cfg.TLS != nil {
		cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if err != nil {
			return nil, err
		}
		s.tlsCfg = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}
	s.authMiddleware = netpkg.NewHTTPAuthMiddleware(cfg.User, cfg.Password).SetAuthFailDelay(200 * time.Millisecond).Middleware
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
	return s.hs.Close()
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

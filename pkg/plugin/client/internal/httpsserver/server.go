// Copyright 2026 The frp Authors
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

//go:build !frps

package httpsserver

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/transport"
	httppkg "github.com/fatedier/frp/pkg/util/http"
)

// snapshot is immutable after publication. Errors retain the last usable pair.
type snapshot struct {
	cert            *tls.Certificate
	certPEM, keyPEM []byte
	err             error
}

type Server struct {
	*http.Server

	crtPath, keyPath string
	reloadMu         sync.Mutex
	state            atomic.Pointer[snapshot]
}

func New(handler http.Handler, crtPath, keyPath string, enableHTTP2 *bool) (*Server, error) {
	server := &Server{crtPath: crtPath, keyPath: keyPath}
	var tlsConfig *tls.Config
	var err error
	if crtPath == "" || keyPath == "" {
		tlsConfig, err = transport.NewServerTLSConfig(crtPath, keyPath, "")
	} else {
		err = server.Reload()
		tlsConfig = &tls.Config{GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return server.state.Load().cert, nil
		}}
	}
	if err != nil {
		return nil, fmt.Errorf("gen TLS config error: %v", err)
	}

	server.Server = &http.Server{
		Handler:           withMisdirectedRequestCheck(handler),
		ReadHeaderTimeout: 60 * time.Second,
		TLSConfig:         tlsConfig,
	}
	if !lo.FromPtr(enableHTTP2) {
		server.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}
	return server, nil
}

func withMisdirectedRequestCheck(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil {
			tlsServerName, _ := httppkg.CanonicalHost(r.TLS.ServerName)
			host, _ := httppkg.CanonicalHost(r.Host)
			if tlsServerName != "" && tlsServerName != host {
				w.WriteHeader(http.StatusMisdirectedRequest)
				return
			}
		}
		handler.ServeHTTP(w, r)
	})
}

func (s *Server) Reload() (err error) {
	if s.crtPath == "" || s.keyPath == "" {
		return nil
	}
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()
	old := s.state.Load()
	// Publish the error before releasing reloadMu; status never waits for I/O.
	defer func() {
		if err != nil {
			next := snapshot{err: err}
			if old != nil {
				next.cert, next.certPEM, next.keyPEM = old.cert, old.certPEM, old.keyPEM
			}
			s.state.Store(&next)
		}
	}()
	certPEM, err := os.ReadFile(s.crtPath)
	if err != nil {
		return fmt.Errorf("read certificate: %w", err)
	}
	keyPEM, err := os.ReadFile(s.keyPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	if old != nil && old.cert != nil && bytes.Equal(certPEM, old.certPEM) && bytes.Equal(keyPEM, old.keyPEM) {
		if old.err != nil {
			next := *old
			next.err = nil
			s.state.Store(&next)
		}
		return nil
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("parse certificate pair: %w", err)
	}
	s.state.Store(&snapshot{cert: &cert, certPEM: certPEM, keyPEM: keyPEM})
	return nil
}

func (s *Server) ReloadError() error {
	if v := s.state.Load(); v != nil {
		return v.err
	}
	return nil
}

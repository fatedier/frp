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
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/transport"
	httppkg "github.com/fatedier/frp/pkg/util/http"
)

func New(handler http.Handler, crtPath, keyPath string, enableHTTP2 *bool) (*http.Server, error) {
	tlsConfig, err := transport.NewServerTLSConfig(crtPath, keyPath, "")
	if err != nil {
		return nil, fmt.Errorf("gen TLS config error: %v", err)
	}

	server := &http.Server{
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

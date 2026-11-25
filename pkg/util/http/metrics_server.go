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

package http

import (
	"net/http"
	"net/http/pprof"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// NewMetricsServer creates a listener with only health, optional metrics, and
// opt-in profiling endpoints. A zero port disables it and returns a nil server.
func NewMetricsServer(cfg v1.MetricsServerConfig, metricsHandler http.Handler) (*Server, error) {
	if cfg.Port == 0 {
		return nil, nil
	}
	s, err := newServer(cfg.Addr, cfg.Port, cfg.User, cfg.Password, cfg.TLS)
	if err != nil {
		return nil, err
	}
	s.router.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	authed := s.router.NewRoute().Subrouter()
	authed.Use(s.authMiddleware)
	if metricsHandler != nil {
		authed.Handle("/metrics", metricsHandler)
	}
	if cfg.PprofEnable {
		// Use exact routes rather than a prefix so cmdline and unknown endpoints
		// cannot reach pprof.Index's fallback dispatch.
		authed.HandleFunc("/debug/pprof/", pprof.Index)
		authed.HandleFunc("/debug/pprof/profile", pprof.Profile)
		authed.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		authed.HandleFunc("/debug/pprof/trace", pprof.Trace)
		for _, name := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
			authed.Handle("/debug/pprof/"+name, pprof.Handler(name))
		}
	}
	return s, nil
}

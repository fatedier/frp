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
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/assets"
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func newTestMetricsServer(t *testing.T, pprofEnabled bool, handler http.Handler) *Server {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	s, err := NewMetricsServer(v1.MetricsServerConfig{
		Addr: "127.0.0.1", Port: port, User: "monitor", Password: "secret", PprofEnable: pprofEnabled,
	}, handler)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return s
}

func TestMetricsServerRoutes(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "pprof disabled"
		if enabled {
			name = "pprof enabled"
		}
		t.Run(name, func(t *testing.T) {
			calls := 0
			s := newTestMetricsServer(t, enabled, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				_, _ = w.Write([]byte("business metrics"))
			}))
			for _, authed := range []bool{false, true} {
				for _, path := range []string{
					"/healthz", "/metrics", "/debug/pprof/", "/debug/pprof/goroutine",
					"/debug/pprof/cmdline", "/debug/pprof/unknown", "/", "/static/app.js", "/api/serverinfo", "/api/config",
				} {
					req := httptest.NewRequest(http.MethodGet, path, nil)
					if authed {
						req.SetBasicAuth("monitor", "secret")
					}
					resp := httptest.NewRecorder()
					s.router.ServeHTTP(resp, req)
					expected := http.StatusNotFound
					switch path {
					case "/healthz":
						expected = http.StatusOK
						require.Empty(t, resp.Body.String())
					case "/metrics":
						expected = http.StatusUnauthorized
						if authed {
							expected = http.StatusOK
							require.Equal(t, "business metrics", resp.Body.String())
						}
					case "/debug/pprof/", "/debug/pprof/goroutine":
						if enabled {
							expected = http.StatusUnauthorized
							if authed {
								expected = http.StatusOK
							}
						}
					}
					require.Equal(t, expected, resp.Code, "path=%s authenticated=%v", path, authed)
				}
			}
			require.Equal(t, 1, calls, "unauthenticated requests must not reach the metrics handler")
		})
	}
}

func TestMetricsServerWithoutMetricsPreservesAssets(t *testing.T) {
	assets.Load(t.TempDir())
	before := assets.FileSystem
	t.Cleanup(func() { assets.Load("") })
	s := newTestMetricsServer(t, false, nil)
	// assets.Load uses http.Dir for local assets, a value rather than a pointer.
	require.Equal(t, before, assets.FileSystem)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.SetBasicAuth("monitor", "secret")
	resp := httptest.NewRecorder()
	s.router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestMetricsServerDisabled(t *testing.T) {
	s, err := NewMetricsServer(v1.MetricsServerConfig{TLS: &v1.TLSConfig{CertFile: "nonexistent"}}, nil)
	require.NoError(t, err)
	require.Nil(t, s)
}

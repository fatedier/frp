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

package client

import (
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestNewServiceReleasesWebServerOnMetricsServerFailure(t *testing.T) {
	for _, mode := range []string{"port conflict", "invalid TLS certificate"} {
		t.Run(mode, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer ln.Close()
			cfg := &v1.ClientCommonConfig{
				WebServer:     v1.WebServerConfig{Port: getFreeTCPPort(t)},
				MetricsServer: v1.MetricsServerConfig{Port: ln.Addr().(*net.TCPAddr).Port},
			}
			if mode == "invalid TLS certificate" {
				cfg.MetricsServer.TLS = &v1.TLSConfig{
					CertFile: filepath.Join(t.TempDir(), "missing.crt"), KeyFile: "missing.key",
				}
				require.NoError(t, ln.Close())
			}
			svr, err := NewService(ServiceOptions{
				Common: cfg, ConfigSourceAggregator: source.NewAggregator(source.NewConfigSource()),
			})
			if svr != nil {
				svr.Close()
			}
			require.Error(t, err)
			require.Nil(t, svr)
			assertLifecycleAdminPort(t, net.JoinHostPort(cfg.WebServer.Addr, strconv.Itoa(cfg.WebServer.Port)), true)
			if mode == "invalid TLS certificate" {
				assertLifecycleAdminPort(t, ln.Addr().String(), true)
			}
		})
	}
}

func TestServiceMetricsServerLifecycle(t *testing.T) {
	svr, err := NewService(ServiceOptions{
		Common: &v1.ClientCommonConfig{
			LoginFailExit: new(false), MetricsServer: v1.MetricsServerConfig{Port: getFreeTCPPort(t)},
		},
		ConfigSourceAggregator: source.NewAggregator(source.NewConfigSource()),
		ConnectorCreator: func(context.Context, *v1.ClientCommonConfig) Connector {
			return &failingConnector{err: net.ErrClosed}
		},
	})
	require.NoError(t, err)
	require.Nil(t, svr.webServer)
	address := svr.metricsServer.Address()
	done := make(chan struct{})
	go func() { _ = svr.Run(context.Background()); close(done) }()
	defer func() {
		svr.Close()
		waitLifecycleSignal(t, done)
		assertLifecycleAdminPort(t, address, true)
	}()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + address + "/healthz")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, resp.Body.Close())

	// frpc serves Go runtime metrics only, and no credentials are configured
	// here, so this endpoint is reachable without authentication.
	resp, err = client.Get("http://" + address + "/metrics")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Contains(t, string(body), "go_goroutines")
	require.NotContains(t, string(body), "frp_")
}

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

package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestServiceMetricsServerWithoutWebServer(t *testing.T) {
	cfg := &v1.ServerConfig{
		BindAddr: "127.0.0.1", ProxyBindAddr: "127.0.0.1",
		MetricsServer: v1.MetricsServerConfig{Port: freeLifecycleTCPPort(t)},
	}
	require.NoError(t, cfg.Complete())
	cfg.BindPort = 0
	svr, err := NewService(cfg)
	require.NoError(t, err)
	require.Nil(t, svr.webServer)
	address := svr.metricsServer.Address()
	done := make(chan struct{})
	go func() { svr.Run(context.Background()); close(done) }()
	defer func() {
		require.NoError(t, svr.Close())
		waitForSignal(t, done, "metrics service shutdown")
		assertLifecyclePorts(t, []string{address}, true)
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + address + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Empty(t, body)
}

func TestNewServiceReleasesResourcesOnMetricsServerFailure(t *testing.T) {
	for _, mode := range []string{"port conflict", "invalid TLS certificate"} {
		t.Run(mode, func(t *testing.T) {
			cfg := &v1.ServerConfig{BindAddr: "127.0.0.1", ProxyBindAddr: "127.0.0.1"}
			require.NoError(t, cfg.Complete())
			// Reserve distinct ports together, including the second HTTP listener.
			ports := []*int{&cfg.BindPort, &cfg.VhostHTTPPort, &cfg.WebServer.Port, &cfg.MetricsServer.Port}
			reservations := make([]net.Listener, 0, len(ports))
			addresses := make([]string, 0, len(ports))
			for _, port := range ports {
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				t.Cleanup(func() { _ = ln.Close() })
				*port = ln.Addr().(*net.TCPAddr).Port
				reservations = append(reservations, ln)
				addresses = append(addresses, ln.Addr().String())
			}
			for _, ln := range reservations[:len(reservations)-1] {
				require.NoError(t, ln.Close())
			}
			if mode == "invalid TLS certificate" {
				require.NoError(t, reservations[len(reservations)-1].Close())
				cfg.MetricsServer.TLS = &v1.TLSConfig{
					CertFile: filepath.Join(t.TempDir(), "missing.crt"), KeyFile: "missing.key",
				}
			} else {
				// This port belongs to the test, not to the failed service.
				addresses = addresses[:len(addresses)-1]
			}
			workersBefore := countPortManagerWorkers(t)
			for range 5 {
				svr, err := NewService(cfg)
				if svr != nil {
					_ = svr.Close()
				}
				require.Error(t, err)
				require.Nil(t, svr)
				if mode == "port conflict" {
					require.ErrorContains(t, err, fmt.Sprint(cfg.MetricsServer.Port))
				} else {
					require.ErrorContains(t, err, "missing.crt")
				}
				assertLifecyclePorts(t, addresses, true)
			}
			require.Equal(t, workersBefore, countPortManagerWorkers(t), "failed construction leaked port-manager workers")
		})
	}
}

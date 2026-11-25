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

package validation

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestValidateMetricsServerConfig(t *testing.T) {
	for _, tc := range []struct {
		name    string
		metrics v1.MetricsServerConfig
		web     v1.WebServerConfig
		wantErr string
	}{
		{name: "disabled"},
		{name: "negative port", metrics: v1.MetricsServerConfig{Port: -1}, wantErr: "metricsServer.port"},
		{name: "overflow port", metrics: v1.MetricsServerConfig{Port: 65536}, wantErr: "metricsServer.port"},
		{name: "maximum port", metrics: v1.MetricsServerConfig{Port: 65535}},
		{name: "empty TLS", metrics: v1.MetricsServerConfig{TLS: &v1.TLSConfig{}}, wantErr: "metricsServer.tls.certFile"},
		{name: "only certificate", metrics: v1.MetricsServerConfig{TLS: &v1.TLSConfig{CertFile: "cert"}}, wantErr: "metricsServer.tls.keyFile"},
		{name: "only key", metrics: v1.MetricsServerConfig{TLS: &v1.TLSConfig{KeyFile: "key"}}, wantErr: "metricsServer.tls.certFile"},
		{name: "TLS pair", metrics: v1.MetricsServerConfig{TLS: &v1.TLSConfig{CertFile: "cert", KeyFile: "key"}}},
		{name: "system CA", metrics: v1.MetricsServerConfig{TLS: &v1.TLSConfig{UseSystemCa: true}}, wantErr: "metricsServer.tls.useSystemCa"},
		{
			name: "same address", metrics: v1.MetricsServerConfig{Addr: "127.0.0.1", Port: 7501},
			web: v1.WebServerConfig{Addr: "127.0.0.1", Port: 7501}, wantErr: "conflicts with webServer",
		},
		{
			name: "different addresses", metrics: v1.MetricsServerConfig{Addr: "127.0.0.1", Port: 7501},
			web: v1.WebServerConfig{Addr: "127.0.0.2", Port: 7501},
		},
		{
			name: "different ports", metrics: v1.MetricsServerConfig{Port: 7501},
			web: v1.WebServerConfig{Port: 7500},
		},
		{
			name: "web disabled", metrics: v1.MetricsServerConfig{Port: 7501},
		},
		{
			name: "metrics disabled", web: v1.WebServerConfig{Port: 7501},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkMetricsConfigBoth(t, tc.metrics, tc.web, tc.wantErr)
		})
	}
	// Two different addresses on the same port may or may not coexist depending
	// on the address family and the platform, so only an exact duplicate is a
	// configuration error here.
	for _, wildcard := range []string{"", "0.0.0.0", "::"} {
		t.Run("wildcard "+wildcard+" with loopback", func(t *testing.T) {
			checkMetricsConfigBoth(t, v1.MetricsServerConfig{Addr: wildcard, Port: 7501},
				v1.WebServerConfig{Addr: "127.0.0.1", Port: 7501}, "")
			checkMetricsConfigBoth(t, v1.MetricsServerConfig{Addr: "127.0.0.1", Port: 7501},
				v1.WebServerConfig{Addr: wildcard, Port: 7501}, "")
		})
		t.Run("same wildcard "+wildcard, func(t *testing.T) {
			checkMetricsConfigBoth(t, v1.MetricsServerConfig{Addr: wildcard, Port: 7501},
				v1.WebServerConfig{Addr: wildcard, Port: 7501}, "conflicts with webServer")
		})
	}
	t.Run("different wildcards on the same port", func(t *testing.T) {
		checkMetricsConfigBoth(t, v1.MetricsServerConfig{Addr: "0.0.0.0", Port: 7501},
			v1.WebServerConfig{Addr: "::", Port: 7501}, "")
	})
}

func checkMetricsConfigBoth(t *testing.T, metrics v1.MetricsServerConfig, web v1.WebServerConfig, wantErr string) {
	t.Helper()
	server := &v1.ServerConfig{}
	client := &v1.ClientCommonConfig{}
	require.NoError(t, server.Complete())
	require.NoError(t, client.Complete())
	// Assign after completion to also check raw empty (wildcard) addresses.
	server.MetricsServer, server.WebServer = metrics, web
	client.MetricsServer, client.WebServer = metrics, web
	validator := NewConfigValidator(nil)
	_, serverErr := validator.ValidateServerConfig(server)
	_, clientErr := validator.ValidateClientCommonConfig(client)
	for _, err := range []error{serverErr, clientErr} {
		if wantErr == "" {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, wantErr)
			require.ErrorContains(t, err, "metricsServer")
		}
	}
}

func TestMetricsServerDefaults(t *testing.T) {
	for _, port := range []int{0, 7501} {
		server := &v1.ServerConfig{MetricsServer: v1.MetricsServerConfig{Port: port}}
		client := &v1.ClientCommonConfig{MetricsServer: v1.MetricsServerConfig{Port: port}}
		require.NoError(t, server.Complete())
		require.NoError(t, client.Complete())
		for _, cfg := range []v1.MetricsServerConfig{server.MetricsServer, client.MetricsServer} {
			require.Equal(t, "127.0.0.1", cfg.Addr)
			require.Equal(t, port, cfg.Port)
			require.False(t, cfg.PprofEnable)
		}
	}
}

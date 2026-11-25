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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestClientTLSCATrustSources(t *testing.T) {
	for _, test := range []struct {
		name        string
		tlsJSON     string
		useSystemCa bool
		caFile      string
	}{
		{name: "default", tlsJSON: `{}`},
		{name: "file only", tlsJSON: `{"trustedCaFile":"ca.pem"}`, caFile: "ca.pem"},
		{name: "system only", tlsJSON: `{"useSystemCa":true}`, useSystemCa: true},
		{name: "combined", tlsJSON: `{"useSystemCa":true,"trustedCaFile":"ca.pem"}`, useSystemCa: true, caFile: "ca.pem"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &v1.ClientCommonConfig{}
			require.NoError(t, json.Unmarshal([]byte(`{"transport":{"tls":`+test.tlsJSON+`}}`), cfg))
			require.NoError(t, cfg.Complete())
			require.Equal(t, test.useSystemCa, cfg.Transport.TLS.UseSystemCa)
			require.Equal(t, test.caFile, cfg.Transport.TLS.TrustedCaFile)
			warnings, err := NewConfigValidator(nil).ValidateClientCommonConfig(cfg)
			require.NoError(t, warnings)
			require.NoError(t, err)
		})
	}
}

func TestClientSystemCARequiresTLS(t *testing.T) {
	for _, protocol := range []string{"tcp", "kcp", "quic", "websocket", "wss"} {
		for _, state := range []struct {
			name   string
			enable *bool
		}{
			{name: "default"},
			{name: "enabled", enable: new(true)},
			{name: "disabled", enable: new(false)},
		} {
			t.Run(protocol+"/"+state.name, func(t *testing.T) {
				cfg := &v1.ClientCommonConfig{}
				cfg.Transport.Protocol = protocol
				cfg.Transport.TLS.Enable = state.enable
				cfg.Transport.TLS.UseSystemCa = true
				require.NoError(t, cfg.Complete())
				warnings, err := NewConfigValidator(nil).ValidateClientCommonConfig(cfg)
				require.NoError(t, warnings, "useSystemCa must produce an error, not a warning")
				if state.name == "disabled" && protocol != "wss" {
					require.ErrorContains(t, err, "transport.tls.useSystemCa")
					require.ErrorContains(t, err, "transport.tls.enable")
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

func TestDisabledTLSFileWarnings(t *testing.T) {
	for _, protocol := range []string{"tcp", "kcp", "quic", "websocket", "wss"} {
		t.Run(protocol, func(t *testing.T) {
			cfg := &v1.ClientCommonConfig{}
			cfg.Transport.Protocol = protocol
			cfg.Transport.TLS.Enable = new(false)
			cfg.Transport.TLS.CertFile = "client.pem"
			cfg.Transport.TLS.KeyFile = "client.key"
			cfg.Transport.TLS.TrustedCaFile = "ca.pem"
			require.NoError(t, cfg.Complete())
			warnings, err := NewConfigValidator(nil).ValidateClientCommonConfig(cfg)
			require.NoError(t, err)
			for _, field := range []string{"certFile", "keyFile", "trustedCaFile"} {
				require.ErrorContains(t, warnings, "transport.tls."+field+" is invalid when transport.tls.enable is false")
			}
		})
	}
}

func TestSystemCAUnsupportedLocations(t *testing.T) {
	for _, test := range []struct {
		name       string
		server     bool
		configJSON string
		fieldPath  string
	}{
		{
			name: "frps transport", server: true,
			configJSON: `{"transport":{"tls":{"useSystemCa":%t}}}`, fieldPath: "transport.tls.useSystemCa",
		},
		{
			name: "frpc webServer", configJSON: `{"webServer":{"tls":{"certFile":"server.pem","keyFile":"server.key","useSystemCa":%t}}}`,
			fieldPath: "webServer.tls.useSystemCa",
		},
		{
			name: "frps webServer", server: true,
			configJSON: `{"webServer":{"tls":{"certFile":"server.pem","keyFile":"server.key","useSystemCa":%t}}}`,
			fieldPath:  "webServer.tls.useSystemCa",
		},
	} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", test.name, enabled), func(t *testing.T) {
				data := fmt.Appendf(nil, test.configJSON, enabled)
				var warnings Warning
				var err error
				if test.server {
					cfg := &v1.ServerConfig{}
					require.NoError(t, json.Unmarshal(data, cfg))
					require.NoError(t, cfg.Complete())
					warnings, err = NewConfigValidator(nil).ValidateServerConfig(cfg)
				} else {
					cfg := &v1.ClientCommonConfig{}
					require.NoError(t, json.Unmarshal(data, cfg))
					require.NoError(t, cfg.Complete())
					warnings, err = NewConfigValidator(nil).ValidateClientCommonConfig(cfg)
				}
				require.NoError(t, warnings)
				if enabled {
					require.EqualError(t, err, test.fieldPath+" is only supported in frpc transport.tls")
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

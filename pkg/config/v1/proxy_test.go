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

package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalTypedProxyConfig(t *testing.T) {
	require := require.New(t)
	proxyConfigs := struct {
		Proxies []TypedProxyConfig `json:"proxies,omitempty"`
	}{}

	strs := `{
		"proxies": [
			{
				"type": "tcp",
				"localPort": 22,
				"remotePort": 6000
			},
			{
				"type": "http",
				"localPort": 80,
				"customDomains": ["www.example.com"]
			}
		]
	}`
	err := json.Unmarshal([]byte(strs), &proxyConfigs)
	require.NoError(err)

	require.IsType(&TCPProxyConfig{}, proxyConfigs.Proxies[0].ProxyConfigurer)
	require.IsType(&HTTPProxyConfig{}, proxyConfigs.Proxies[1].ProxyConfigurer)
}

func TestUnmarshalTypedProxyConfigWithClientLevelFields(t *testing.T) {
	// Enable strict mode to test DisallowUnknownFields behavior
	DisallowUnknownFieldsMu.Lock()
	oldValue := DisallowUnknownFields
	DisallowUnknownFields = true
	DisallowUnknownFieldsMu.Unlock()
	defer func() {
		DisallowUnknownFieldsMu.Lock()
		DisallowUnknownFields = oldValue
		DisallowUnknownFieldsMu.Unlock()
	}()

	tests := []struct {
		name          string
		config        string
		expectedError string
	}{
		{
			name: "featureGates in proxy config",
			config: `{
				"proxies": [
					{
						"type": "tcp",
						"localPort": 22,
						"remotePort": 6000,
						"featureGates": {"VirtualNet": true}
					}
				]
			}`,
			expectedError: "featureGates",
		},
		{
			name: "virtualNet in proxy config",
			config: `{
				"proxies": [
					{
						"type": "tcp",
						"localPort": 22,
						"remotePort": 6000,
						"virtualNet": {"address": "100.86.0.2/24"}
					}
				]
			}`,
			expectedError: "virtualNet",
		},
		{
			name: "auth in proxy config",
			config: `{
				"proxies": [
					{
						"type": "tcp",
						"localPort": 22,
						"remotePort": 6000,
						"auth": {"token": "12345"}
					}
				]
			}`,
			expectedError: "auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			proxyConfigs := struct {
				Proxies []TypedProxyConfig `json:"proxies,omitempty"`
			}{}

			err := json.Unmarshal([]byte(tt.config), &proxyConfigs)
			require.Error(err)
			require.Contains(err.Error(), tt.expectedError)
			require.Contains(err.Error(), "client-level configuration field")
			require.Contains(err.Error(), "root level")
			require.Contains(err.Error(), "not within a [[proxies]] section")
		})
	}
}

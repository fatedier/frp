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
	"sigs.k8s.io/yaml"
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

func TestMarshalTypedProxyConfig(t *testing.T) {
	require := require.New(t)

	clientConfig := ClientConfig{
		ClientCommonConfig: ClientCommonConfig{
			Auth: AuthClientConfig{
				Method: AuthMethodToken,
				Token:  "update-me",
			},
			ServerAddr: "frp.example.org",
			ServerPort: 8080,
			Log: LogConfig{
				Level: "info",
			},
		},
		Proxies: []TypedProxyConfig{
			{
				Type: string(ProxyTypeTCP),
				ProxyConfigurer: &TCPProxyConfig{
					ProxyBaseConfig: ProxyBaseConfig{
						Name: "proxy1",
						Type: string(ProxyTypeTCP),
						ProxyBackend: ProxyBackend{
							LocalIP:   "192.168.0.101",
							LocalPort: 8889,
						},
					},
					RemotePort: 30001,
				},
			},
			{
				Type: string(ProxyTypeTCP),
				ProxyConfigurer: &TCPProxyConfig{
					ProxyBaseConfig: ProxyBaseConfig{
						Name: "proxy2",
						Type: string(ProxyTypeTCP),
						ProxyBackend: ProxyBackend{
							LocalIP:   "192.168.0.102",
							LocalPort: 8889,
						},
					},
					RemotePort: 30002,
				},
			},
		},
	}

	yamlConfig, err := yaml.Marshal(&clientConfig)
	require.NoError(err)
	clientConfigUnmarshalled := ClientConfig{}
	err = yaml.Unmarshal(yamlConfig, &clientConfigUnmarshalled)
	require.NoError(err)
	require.Equal(clientConfig, clientConfigUnmarshalled)
}

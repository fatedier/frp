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

package config

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

const tomlServerContent = `
bindAddr = "127.0.0.1"
kcpBindPort = 7000
quicBindPort = 7001
tcpmuxHTTPConnectPort = 7005
custom404Page = "/abc.html"
transport.tcpKeepalive = 10
`

const yamlServerContent = `
bindAddr: 127.0.0.1
kcpBindPort: 7000
quicBindPort: 7001
tcpmuxHTTPConnectPort: 7005
custom404Page: /abc.html
transport:
  tcpKeepalive: 10
`

const jsonServerContent = `
{
  "bindAddr": "127.0.0.1",
  "kcpBindPort": 7000,
  "quicBindPort": 7001,
  "tcpmuxHTTPConnectPort": 7005,
  "custom404Page": "/abc.html",
  "transport": {
    "tcpKeepalive": 10
  }
}
`

func TestLoadServerConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"toml", tomlServerContent},
		{"yaml", yamlServerContent},
		{"json", jsonServerContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			svrCfg := v1.ServerConfig{}
			err := LoadConfigure([]byte(test.content), &svrCfg, true)
			require.NoError(err)
			require.EqualValues("127.0.0.1", svrCfg.BindAddr)
			require.EqualValues(7000, svrCfg.KCPBindPort)
			require.EqualValues(7001, svrCfg.QUICBindPort)
			require.EqualValues(7005, svrCfg.TCPMuxHTTPConnectPort)
			require.EqualValues("/abc.html", svrCfg.Custom404Page)
			require.EqualValues(10, svrCfg.Transport.TCPKeepAlive)
		})
	}
}

// Test that loading in strict mode fails when the config is invalid.
func TestLoadServerConfigStrictMode(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"toml", tomlServerContent},
		{"yaml", yamlServerContent},
		{"json", jsonServerContent},
	}

	for _, strict := range []bool{false, true} {
		for _, test := range tests {
			t.Run(fmt.Sprintf("%s-strict-%t", test.name, strict), func(t *testing.T) {
				require := require.New(t)
				// Break the content with an innocent typo
				brokenContent := strings.Replace(test.content, "bindAddr", "bindAdur", 1)
				svrCfg := v1.ServerConfig{}
				err := LoadConfigure([]byte(brokenContent), &svrCfg, strict)
				if strict {
					require.ErrorContains(err, "bindAdur")
				} else {
					require.NoError(err)
					// BindAddr didn't get parsed because of the typo.
					require.EqualValues("", svrCfg.BindAddr)
				}
			})
		}
	}
}

func TestRenderWithTemplate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"toml", tomlServerContent, tomlServerContent},
		{"yaml", yamlServerContent, yamlServerContent},
		{"json", jsonServerContent, jsonServerContent},
		{"template numeric", `key = {{ 123 }}`, "key = 123"},
		{"template string", `key = {{ "xyz" }}`, "key = xyz"},
		{"template quote", `key = {{ printf "%q" "with space" }}`, `key = "with space"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			got, err := RenderWithTemplate([]byte(test.content), nil)
			require.NoError(err)
			require.EqualValues(test.want, string(got))
		})
	}
}

func TestCustomStructStrictMode(t *testing.T) {
	require := require.New(t)

	proxyStr := `
serverPort = 7000

[[proxies]]
name = "test"
type = "tcp"
remotePort = 6000
`
	clientCfg := v1.ClientConfig{}
	err := LoadConfigure([]byte(proxyStr), &clientCfg, true)
	require.NoError(err)

	proxyStr += `unknown = "unknown"`
	err = LoadConfigure([]byte(proxyStr), &clientCfg, true)
	require.Error(err)

	visitorStr := `
serverPort = 7000

[[visitors]]
name = "test"
type = "stcp"
bindPort = 6000
serverName = "server"
`
	err = LoadConfigure([]byte(visitorStr), &clientCfg, true)
	require.NoError(err)

	visitorStr += `unknown = "unknown"`
	err = LoadConfigure([]byte(visitorStr), &clientCfg, true)
	require.Error(err)

	pluginStr := `
serverPort = 7000

[[proxies]]
name = "test"
type = "tcp"
remotePort = 6000
[proxies.plugin]
type = "unix_domain_socket"
unixPath = "/tmp/uds.sock"
`
	err = LoadConfigure([]byte(pluginStr), &clientCfg, true)
	require.NoError(err)
	pluginStr += `unknown = "unknown"`
	err = LoadConfigure([]byte(pluginStr), &clientCfg, true)
	require.Error(err)
}

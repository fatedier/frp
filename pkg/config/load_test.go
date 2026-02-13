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
	"encoding/json"
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

// TestYAMLMergeInStrictMode tests that YAML merge functionality works
// even in strict mode by properly handling dot-prefixed fields
func TestYAMLMergeInStrictMode(t *testing.T) {
	require := require.New(t)

	yamlContent := `
serverAddr: "127.0.0.1"
serverPort: 7000

.common: &common
  type: stcp
  secretKey: "test-secret"
  localIP: 127.0.0.1
  transport:
    useEncryption: true
    useCompression: true

proxies:
- name: ssh
  localPort: 22
  <<: *common
- name: web
  localPort: 80
  <<: *common
`

	clientCfg := v1.ClientConfig{}
	// This should work in strict mode
	err := LoadConfigure([]byte(yamlContent), &clientCfg, true)
	require.NoError(err)

	// Verify the merge worked correctly
	require.Equal("127.0.0.1", clientCfg.ServerAddr)
	require.Equal(7000, clientCfg.ServerPort)
	require.Len(clientCfg.Proxies, 2)

	// Check first proxy
	sshProxy := clientCfg.Proxies[0].ProxyConfigurer
	require.Equal("ssh", sshProxy.GetBaseConfig().Name)
	require.Equal("stcp", sshProxy.GetBaseConfig().Type)

	// Check second proxy
	webProxy := clientCfg.Proxies[1].ProxyConfigurer
	require.Equal("web", webProxy.GetBaseConfig().Name)
	require.Equal("stcp", webProxy.GetBaseConfig().Type)
}

// TestOptimizedYAMLProcessing tests the optimization logic for YAML processing
func TestOptimizedYAMLProcessing(t *testing.T) {
	require := require.New(t)

	yamlWithDotFields := []byte(`
serverAddr: "127.0.0.1"
.common: &common
  type: stcp
proxies:
- name: test
  <<: *common
`)

	yamlWithoutDotFields := []byte(`
serverAddr: "127.0.0.1"
proxies:
- name: test
  type: tcp
  localPort: 22
`)

	// Test that YAML without dot fields works in strict mode
	clientCfg := v1.ClientConfig{}
	err := LoadConfigure(yamlWithoutDotFields, &clientCfg, true)
	require.NoError(err)
	require.Equal("127.0.0.1", clientCfg.ServerAddr)
	require.Len(clientCfg.Proxies, 1)
	require.Equal("test", clientCfg.Proxies[0].ProxyConfigurer.GetBaseConfig().Name)

	// Test that YAML with dot fields still works in strict mode
	err = LoadConfigure(yamlWithDotFields, &clientCfg, true)
	require.NoError(err)
	require.Equal("127.0.0.1", clientCfg.ServerAddr)
	require.Len(clientCfg.Proxies, 1)
	require.Equal("test", clientCfg.Proxies[0].ProxyConfigurer.GetBaseConfig().Name)
	require.Equal("stcp", clientCfg.Proxies[0].ProxyConfigurer.GetBaseConfig().Type)
}

func TestFilterClientConfigurers_PreserveRawNamesAndNoMutation(t *testing.T) {
	require := require.New(t)

	enabled := true
	proxyCfg := &v1.TCPProxyConfig{}
	proxyCfg.Name = "proxy-raw"
	proxyCfg.Type = "tcp"
	proxyCfg.LocalPort = 10080
	proxyCfg.Enabled = &enabled

	visitorCfg := &v1.XTCPVisitorConfig{}
	visitorCfg.Name = "visitor-raw"
	visitorCfg.Type = "xtcp"
	visitorCfg.ServerName = "server-raw"
	visitorCfg.FallbackTo = "fallback-raw"
	visitorCfg.SecretKey = "secret"
	visitorCfg.BindPort = 10081
	visitorCfg.Enabled = &enabled

	common := &v1.ClientCommonConfig{
		User: "alice",
	}

	proxies, visitors := FilterClientConfigurers(common, []v1.ProxyConfigurer{proxyCfg}, []v1.VisitorConfigurer{visitorCfg})
	require.Len(proxies, 1)
	require.Len(visitors, 1)

	p := proxies[0].GetBaseConfig()
	require.Equal("proxy-raw", p.Name)
	require.Empty(p.LocalIP)

	v := visitors[0].GetBaseConfig()
	require.Equal("visitor-raw", v.Name)
	require.Equal("server-raw", v.ServerName)
	require.Empty(v.BindAddr)

	xtcp := visitors[0].(*v1.XTCPVisitorConfig)
	require.Equal("fallback-raw", xtcp.FallbackTo)
	require.Empty(xtcp.Protocol)
}

func TestCompleteProxyConfigurers_PreserveRawNames(t *testing.T) {
	require := require.New(t)

	enabled := true
	proxyCfg := &v1.TCPProxyConfig{}
	proxyCfg.Name = "proxy-raw"
	proxyCfg.Type = "tcp"
	proxyCfg.LocalPort = 10080
	proxyCfg.Enabled = &enabled

	proxies := CompleteProxyConfigurers([]v1.ProxyConfigurer{proxyCfg})
	require.Len(proxies, 1)

	p := proxies[0].GetBaseConfig()
	require.Equal("proxy-raw", p.Name)
	require.Equal("127.0.0.1", p.LocalIP)
}

func TestCompleteVisitorConfigurers_PreserveRawNames(t *testing.T) {
	require := require.New(t)

	enabled := true
	visitorCfg := &v1.XTCPVisitorConfig{}
	visitorCfg.Name = "visitor-raw"
	visitorCfg.Type = "xtcp"
	visitorCfg.ServerName = "server-raw"
	visitorCfg.FallbackTo = "fallback-raw"
	visitorCfg.SecretKey = "secret"
	visitorCfg.BindPort = 10081
	visitorCfg.Enabled = &enabled

	visitors := CompleteVisitorConfigurers([]v1.VisitorConfigurer{visitorCfg})
	require.Len(visitors, 1)

	v := visitors[0].GetBaseConfig()
	require.Equal("visitor-raw", v.Name)
	require.Equal("server-raw", v.ServerName)
	require.Equal("127.0.0.1", v.BindAddr)

	xtcp := visitors[0].(*v1.XTCPVisitorConfig)
	require.Equal("fallback-raw", xtcp.FallbackTo)
	require.Equal("quic", xtcp.Protocol)
}

func TestCompleteProxyConfigurers_Idempotent(t *testing.T) {
	require := require.New(t)

	proxyCfg := &v1.TCPProxyConfig{}
	proxyCfg.Name = "proxy"
	proxyCfg.Type = "tcp"
	proxyCfg.LocalPort = 10080

	proxies := CompleteProxyConfigurers([]v1.ProxyConfigurer{proxyCfg})
	firstProxyJSON, err := json.Marshal(proxies[0])
	require.NoError(err)

	proxies = CompleteProxyConfigurers(proxies)
	secondProxyJSON, err := json.Marshal(proxies[0])
	require.NoError(err)

	require.Equal(string(firstProxyJSON), string(secondProxyJSON))
}

func TestCompleteVisitorConfigurers_Idempotent(t *testing.T) {
	require := require.New(t)

	visitorCfg := &v1.XTCPVisitorConfig{}
	visitorCfg.Name = "visitor"
	visitorCfg.Type = "xtcp"
	visitorCfg.ServerName = "server"
	visitorCfg.SecretKey = "secret"
	visitorCfg.BindPort = 10081

	visitors := CompleteVisitorConfigurers([]v1.VisitorConfigurer{visitorCfg})
	firstVisitorJSON, err := json.Marshal(visitors[0])
	require.NoError(err)

	visitors = CompleteVisitorConfigurers(visitors)
	secondVisitorJSON, err := json.Marshal(visitors[0])
	require.NoError(err)

	require.Equal(string(firstVisitorJSON), string(secondVisitorJSON))
}

func TestFilterClientConfigurers_FilterByStartAndEnabled(t *testing.T) {
	require := require.New(t)

	enabled := true
	disabled := false

	proxyKeep := &v1.TCPProxyConfig{}
	proxyKeep.Name = "keep"
	proxyKeep.Type = "tcp"
	proxyKeep.LocalPort = 10080
	proxyKeep.Enabled = &enabled

	proxyDropByStart := &v1.TCPProxyConfig{}
	proxyDropByStart.Name = "drop-by-start"
	proxyDropByStart.Type = "tcp"
	proxyDropByStart.LocalPort = 10081
	proxyDropByStart.Enabled = &enabled

	proxyDropByEnabled := &v1.TCPProxyConfig{}
	proxyDropByEnabled.Name = "drop-by-enabled"
	proxyDropByEnabled.Type = "tcp"
	proxyDropByEnabled.LocalPort = 10082
	proxyDropByEnabled.Enabled = &disabled

	common := &v1.ClientCommonConfig{
		Start: []string{"keep"},
	}

	proxies, visitors := FilterClientConfigurers(common, []v1.ProxyConfigurer{
		proxyKeep,
		proxyDropByStart,
		proxyDropByEnabled,
	}, nil)
	require.Len(visitors, 0)
	require.Len(proxies, 1)
	require.Equal("keep", proxies[0].GetBaseConfig().Name)
}

// TestYAMLEdgeCases tests edge cases for YAML parsing, including non-map types
func TestYAMLEdgeCases(t *testing.T) {
	require := require.New(t)

	// Test array at root (should fail for frp config)
	arrayYAML := []byte(`
- item1
- item2
`)
	clientCfg := v1.ClientConfig{}
	err := LoadConfigure(arrayYAML, &clientCfg, true)
	require.Error(err) // Should fail because ClientConfig expects an object

	// Test scalar at root (should fail for frp config)
	scalarYAML := []byte(`"just a string"`)
	err = LoadConfigure(scalarYAML, &clientCfg, true)
	require.Error(err) // Should fail because ClientConfig expects an object

	// Test empty object (should work)
	emptyYAML := []byte(`{}`)
	err = LoadConfigure(emptyYAML, &clientCfg, true)
	require.NoError(err)

	// Test nested structure without dots (should work)
	nestedYAML := []byte(`
serverAddr: "127.0.0.1"
serverPort: 7000
`)
	err = LoadConfigure(nestedYAML, &clientCfg, true)
	require.NoError(err)
	require.Equal("127.0.0.1", clientCfg.ServerAddr)
	require.Equal(7000, clientCfg.ServerPort)
}

// Copyright 2020 The frp Authors
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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fatedier/frp/pkg/auth"
	plugin "github.com/fatedier/frp/pkg/plugin/server"
)

func Test_LoadServerCommonConf(t *testing.T) {
	assert := assert.New(t)

	testcases := []struct {
		source   []byte
		expected ServerCommonConf
	}{
		{
			source: []byte(`
				# [common] is integral section
				[common]
				bind_addr = 0.0.0.9
				bind_port = 7009
				bind_udp_port = 7008
				kcp_bind_port = 7007
				proxy_bind_addr = 127.0.0.9
				vhost_http_port = 89
				vhost_https_port = 449
				vhost_http_timeout = 69
				tcpmux_httpconnect_port = 1339
				dashboard_addr = 0.0.0.9
				dashboard_port = 7509
				dashboard_user = admin9
				dashboard_pwd = admin9
				enable_prometheus
				assets_dir = ./static9
				log_file = ./frps.log9
				log_way = file
				log_level = info9
				log_max_days = 39
				disable_log_color = false
				detailed_errors_to_client
				authentication_method = token
				authenticate_heartbeats = false
				authenticate_new_work_conns = false
				token = 123456789
				oidc_issuer = test9
				oidc_audience = test9
				oidc_skip_expiry_check
				oidc_skip_issuer_check
				heartbeat_timeout = 99
				user_conn_timeout = 9
				allow_ports = 10-12,99
				max_pool_count = 59
				max_ports_per_client = 9
				tls_only = false
				tls_cert_file = server.crt
				tls_key_file = server.key
				tls_trusted_ca_file = ca.crt
				subdomain_host = frps.com
				tcp_mux
				udp_packet_size = 1509
				[plugin.user-manager]
				addr = 127.0.0.1:9009
				path = /handler
				ops = Login
				[plugin.port-manager]
				addr = 127.0.0.1:9009
				path = /handler
				ops = NewProxy
				tls_verify
			`),
			expected: ServerCommonConf{
				ServerConfig: auth.ServerConfig{
					BaseConfig: auth.BaseConfig{
						AuthenticationMethod:     "token",
						AuthenticateHeartBeats:   false,
						AuthenticateNewWorkConns: false,
					},
					TokenConfig: auth.TokenConfig{
						Token: "123456789",
					},
					OidcServerConfig: auth.OidcServerConfig{
						OidcIssuer:          "test9",
						OidcAudience:        "test9",
						OidcSkipExpiryCheck: true,
						OidcSkipIssuerCheck: true,
					},
				},
				BindAddr:               "0.0.0.9",
				BindPort:               7009,
				BindUDPPort:            7008,
				KCPBindPort:            7007,
				ProxyBindAddr:          "127.0.0.9",
				VhostHTTPPort:          89,
				VhostHTTPSPort:         449,
				VhostHTTPTimeout:       69,
				TCPMuxHTTPConnectPort:  1339,
				DashboardAddr:          "0.0.0.9",
				DashboardPort:          7509,
				DashboardUser:          "admin9",
				DashboardPwd:           "admin9",
				EnablePrometheus:       true,
				AssetsDir:              "./static9",
				LogFile:                "./frps.log9",
				LogWay:                 "file",
				LogLevel:               "info9",
				LogMaxDays:             39,
				DisableLogColor:        false,
				DetailedErrorsToClient: true,
				HeartbeatTimeout:       99,
				UserConnTimeout:        9,
				AllowPorts: map[int]struct{}{
					10: {},
					11: {},
					12: {},
					99: {},
				},
				MaxPoolCount:            59,
				MaxPortsPerClient:       9,
				TLSOnly:                 true,
				TLSCertFile:             "server.crt",
				TLSKeyFile:              "server.key",
				TLSTrustedCaFile:        "ca.crt",
				SubDomainHost:           "frps.com",
				TCPMux:                  true,
				TCPMuxKeepaliveInterval: 60,
				TCPKeepAlive:            7200,
				UDPPacketSize:           1509,

				HTTPPlugins: map[string]plugin.HTTPPluginOptions{
					"user-manager": {
						Name: "user-manager",
						Addr: "127.0.0.1:9009",
						Path: "/handler",
						Ops:  []string{"Login"},
					},
					"port-manager": {
						Name:      "port-manager",
						Addr:      "127.0.0.1:9009",
						Path:      "/handler",
						Ops:       []string{"NewProxy"},
						TLSVerify: true,
					},
				},
			},
		},
		{
			source: []byte(`
				# [common] is integral section
				[common]
				bind_addr = 0.0.0.9
				bind_port = 7009
				bind_udp_port = 7008
			`),
			expected: ServerCommonConf{
				ServerConfig: auth.ServerConfig{
					BaseConfig: auth.BaseConfig{
						AuthenticationMethod:     "token",
						AuthenticateHeartBeats:   false,
						AuthenticateNewWorkConns: false,
					},
				},
				BindAddr:                "0.0.0.9",
				BindPort:                7009,
				BindUDPPort:             7008,
				ProxyBindAddr:           "0.0.0.9",
				VhostHTTPTimeout:        60,
				DashboardAddr:           "0.0.0.0",
				DashboardUser:           "",
				DashboardPwd:            "",
				EnablePrometheus:        false,
				LogFile:                 "console",
				LogWay:                  "console",
				LogLevel:                "info",
				LogMaxDays:              3,
				DetailedErrorsToClient:  true,
				TCPMux:                  true,
				TCPMuxKeepaliveInterval: 60,
				TCPKeepAlive:            7200,
				AllowPorts:              make(map[int]struct{}),
				MaxPoolCount:            5,
				HeartbeatTimeout:        90,
				UserConnTimeout:         10,
				HTTPPlugins:             make(map[string]plugin.HTTPPluginOptions),
				UDPPacketSize:           1500,
			},
		},
	}

	for _, c := range testcases {
		actual, err := UnmarshalServerConfFromIni(c.source)
		assert.NoError(err)
		actual.Complete()
		assert.Equal(c.expected, actual)
	}
}

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

	"github.com/fatedier/frp/pkg/auth"
	"github.com/fatedier/frp/pkg/consts"

	"github.com/stretchr/testify/assert"
)

const (
	testUser = "test"
)

var (
	testClientBytesWithFull = []byte(`
		# [common] is integral section
		[common]
		server_addr = 0.0.0.9
		server_port = 7009
		http_proxy = http://user:passwd@192.168.1.128:8080
		log_file = ./frpc.log9
		log_way = file
		log_level = info9
		log_max_days = 39
		disable_log_color = false
		authenticate_heartbeats = false
		authenticate_new_work_conns = false
		token = 12345678
		oidc_client_id = client-id
		oidc_client_secret = client-secret
		oidc_audience = audience
		oidc_token_endpoint_url = endpoint_url
		admin_addr = 127.0.0.9
		admin_port = 7409
		admin_user = admin9
		admin_pwd = admin9
		assets_dir = ./static9
		pool_count = 59
		tcp_mux
		user = your_name
		login_fail_exit
		protocol = tcp
		tls_enable = true
		tls_cert_file = client.crt
		tls_key_file = client.key
		tls_trusted_ca_file = ca.crt
		tls_server_name = example.com
		dns_server = 8.8.8.9
		start = ssh,dns
		heartbeat_interval = 39
		heartbeat_timeout = 99
		meta_var1 = 123
		meta_var2 = 234
		udp_packet_size = 1509
		
		# all proxy
		[ssh]
		type = tcp
		local_ip = 127.0.0.9
		local_port = 29
		bandwidth_limit = 19MB
		use_encryption
		use_compression
		remote_port = 6009
		group = test_group
		group_key = 123456
		health_check_type = tcp
		health_check_timeout_s = 3
		health_check_max_failed = 3
		health_check_interval_s = 19
		meta_var1 = 123
		meta_var2 = 234
		
		[ssh_random]
		type = tcp
		local_ip = 127.0.0.9
		local_port = 29
		remote_port = 9
		
		[range:tcp_port]
		type = tcp
		local_ip = 127.0.0.9
		local_port = 6010-6011,6019
		remote_port = 6010-6011,6019
		use_encryption = false
		use_compression = false
		
		[dns]
		type = udp
		local_ip = 114.114.114.114
		local_port = 59
		remote_port = 6009
		use_encryption
		use_compression
		
		[range:udp_port]
		type = udp
		local_ip = 114.114.114.114
		local_port = 6000,6010-6011
		remote_port = 6000,6010-6011
		use_encryption
		use_compression
		
		[web01]
		type = http
		local_ip = 127.0.0.9
		local_port = 89
		use_encryption
		use_compression
		http_user = admin
		http_pwd = admin
		subdomain = web01
		custom_domains = web02.yourdomain.com
		locations = /,/pic
		host_header_rewrite = example.com
		header_X-From-Where = frp
		health_check_type = http
		health_check_url = /status
		health_check_interval_s = 19
		health_check_max_failed = 3
		health_check_timeout_s = 3
		
		[web02]
		type = https
		local_ip = 127.0.0.9
		local_port = 8009
		use_encryption
		use_compression
		subdomain = web01
		custom_domains = web02.yourdomain.com
		proxy_protocol_version = v2
		
		[secret_tcp]
		type = stcp
		sk = abcdefg
		local_ip = 127.0.0.1
		local_port = 22
		use_encryption = false
		use_compression = false
		
		[p2p_tcp]
		type = xtcp
		sk = abcdefg
		local_ip = 127.0.0.1
		local_port = 22
		use_encryption = false
		use_compression = false
		
		[tcpmuxhttpconnect]
		type = tcpmux
		multiplexer = httpconnect
		local_ip = 127.0.0.1
		local_port = 10701
		custom_domains = tunnel1
		
		[plugin_unix_domain_socket]
		type = tcp
		remote_port = 6003
		plugin = unix_domain_socket
		plugin_unix_path = /var/run/docker.sock
		
		[plugin_http_proxy]
		type = tcp
		remote_port = 6004
		plugin = http_proxy
		plugin_http_user = abc
		plugin_http_passwd = abc
		
		[plugin_socks5]
		type = tcp
		remote_port = 6005
		plugin = socks5
		plugin_user = abc
		plugin_passwd = abc
		
		[plugin_static_file]
		type = tcp
		remote_port = 6006
		plugin = static_file
		plugin_local_path = /var/www/blog
		plugin_strip_prefix = static
		plugin_http_user = abc
		plugin_http_passwd = abc
		
		[plugin_https2http]
		type = https
		custom_domains = test.yourdomain.com
		plugin = https2http
		plugin_local_addr = 127.0.0.1:80
		plugin_crt_path = ./server.crt
		plugin_key_path = ./server.key
		plugin_host_header_rewrite = 127.0.0.1
		plugin_header_X-From-Where = frp
		
		[plugin_http2https]
		type = http
		custom_domains = test.yourdomain.com
		plugin = http2https
		plugin_local_addr = 127.0.0.1:443
		plugin_host_header_rewrite = 127.0.0.1
		plugin_header_X-From-Where = frp
		
		# visitor
		[secret_tcp_visitor]
		role = visitor
		type = stcp
		server_name = secret_tcp
		sk = abcdefg
		bind_addr = 127.0.0.1
		bind_port = 9000
		use_encryption = false
		use_compression = false
		
		[p2p_tcp_visitor]
		role = visitor
		type = xtcp
		server_name = p2p_tcp
		sk = abcdefg
		bind_addr = 127.0.0.1
		bind_port = 9001
		use_encryption = false
		use_compression = false
	`)
)

func Test_LoadClientCommonConf(t *testing.T) {
	assert := assert.New(t)

	expected := ClientCommonConf{
		ClientConfig: auth.ClientConfig{
			BaseConfig: auth.BaseConfig{
				AuthenticationMethod:     "token",
				AuthenticateHeartBeats:   false,
				AuthenticateNewWorkConns: false,
			},
			TokenConfig: auth.TokenConfig{
				Token: "12345678",
			},
			OidcClientConfig: auth.OidcClientConfig{
				OidcClientID:         "client-id",
				OidcClientSecret:     "client-secret",
				OidcAudience:         "audience",
				OidcTokenEndpointURL: "endpoint_url",
			},
		},
		ServerAddr:        "0.0.0.9",
		ServerPort:        7009,
		HTTPProxy:         "http://user:passwd@192.168.1.128:8080",
		LogFile:           "./frpc.log9",
		LogWay:            "file",
		LogLevel:          "info9",
		LogMaxDays:        39,
		DisableLogColor:   false,
		AdminAddr:         "127.0.0.9",
		AdminPort:         7409,
		AdminUser:         "admin9",
		AdminPwd:          "admin9",
		AssetsDir:         "./static9",
		PoolCount:         59,
		TCPMux:            true,
		User:              "your_name",
		LoginFailExit:     true,
		Protocol:          "tcp",
		TLSEnable:         true,
		TLSCertFile:       "client.crt",
		TLSKeyFile:        "client.key",
		TLSTrustedCaFile:  "ca.crt",
		TLSServerName:     "example.com",
		DNSServer:         "8.8.8.9",
		Start:             []string{"ssh", "dns"},
		HeartbeatInterval: 39,
		HeartbeatTimeout:  99,
		Metas: map[string]string{
			"var1": "123",
			"var2": "234",
		},
		UDPPacketSize: 1509,
	}

	common, err := UnmarshalClientConfFromIni(testClientBytesWithFull)
	assert.NoError(err)
	assert.Equal(expected, common)
}

func Test_LoadClientBasicConf(t *testing.T) {
	assert := assert.New(t)

	proxyExpected := map[string]ProxyConf{
		testUser + ".ssh": &TCPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName:      testUser + ".ssh",
				ProxyType:      consts.TCPProxy,
				UseCompression: true,
				UseEncryption:  true,
				Group:          "test_group",
				GroupKey:       "123456",
				BandwidthLimit: MustBandwidthQuantity("19MB"),
				Metas: map[string]string{
					"var1": "123",
					"var2": "234",
				},
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "127.0.0.9",
					LocalPort: 29,
				},
				HealthCheckConf: HealthCheckConf{
					HealthCheckType:      consts.TCPProxy,
					HealthCheckTimeoutS:  3,
					HealthCheckMaxFailed: 3,
					HealthCheckIntervalS: 19,
					HealthCheckAddr:      "127.0.0.9:29",
				},
			},
			RemotePort: 6009,
		},
		testUser + ".ssh_random": &TCPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".ssh_random",
				ProxyType: consts.TCPProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "127.0.0.9",
					LocalPort: 29,
				},
			},
			RemotePort: 9,
		},
		testUser + ".tcp_port_0": &TCPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".tcp_port_0",
				ProxyType: consts.TCPProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "127.0.0.9",
					LocalPort: 6010,
				},
			},
			RemotePort: 6010,
		},
		testUser + ".tcp_port_1": &TCPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".tcp_port_1",
				ProxyType: consts.TCPProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "127.0.0.9",
					LocalPort: 6011,
				},
			},
			RemotePort: 6011,
		},
		testUser + ".tcp_port_2": &TCPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".tcp_port_2",
				ProxyType: consts.TCPProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "127.0.0.9",
					LocalPort: 6019,
				},
			},
			RemotePort: 6019,
		},
		testUser + ".dns": &UDPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName:      testUser + ".dns",
				ProxyType:      consts.UDPProxy,
				UseEncryption:  true,
				UseCompression: true,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "114.114.114.114",
					LocalPort: 59,
				},
			},
			RemotePort: 6009,
		},
		testUser + ".udp_port_0": &UDPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName:      testUser + ".udp_port_0",
				ProxyType:      consts.UDPProxy,
				UseEncryption:  true,
				UseCompression: true,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "114.114.114.114",
					LocalPort: 6000,
				},
			},
			RemotePort: 6000,
		},
		testUser + ".udp_port_1": &UDPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName:      testUser + ".udp_port_1",
				ProxyType:      consts.UDPProxy,
				UseEncryption:  true,
				UseCompression: true,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "114.114.114.114",
					LocalPort: 6010,
				},
			},
			RemotePort: 6010,
		},
		testUser + ".udp_port_2": &UDPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName:      testUser + ".udp_port_2",
				ProxyType:      consts.UDPProxy,
				UseEncryption:  true,
				UseCompression: true,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "114.114.114.114",
					LocalPort: 6011,
				},
			},
			RemotePort: 6011,
		},
		testUser + ".web01": &HTTPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName:      testUser + ".web01",
				ProxyType:      consts.HTTPProxy,
				UseCompression: true,
				UseEncryption:  true,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "127.0.0.9",
					LocalPort: 89,
				},
				HealthCheckConf: HealthCheckConf{
					HealthCheckType:      consts.HTTPProxy,
					HealthCheckTimeoutS:  3,
					HealthCheckMaxFailed: 3,
					HealthCheckIntervalS: 19,
					HealthCheckURL:       "http://127.0.0.9:89/status",
				},
			},
			DomainConf: DomainConf{
				CustomDomains: []string{"web02.yourdomain.com"},
				SubDomain:     "web01",
			},
			Locations:         []string{"/", "/pic"},
			HTTPUser:          "admin",
			HTTPPwd:           "admin",
			HostHeaderRewrite: "example.com",
			Headers: map[string]string{
				"X-From-Where": "frp",
			},
		},
		testUser + ".web02": &HTTPSProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName:      testUser + ".web02",
				ProxyType:      consts.HTTPSProxy,
				UseCompression: true,
				UseEncryption:  true,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "127.0.0.9",
					LocalPort: 8009,
				},
				ProxyProtocolVersion: "v2",
			},
			DomainConf: DomainConf{
				CustomDomains: []string{"web02.yourdomain.com"},
				SubDomain:     "web01",
			},
		},
		testUser + ".secret_tcp": &STCPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".secret_tcp",
				ProxyType: consts.STCPProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "127.0.0.1",
					LocalPort: 22,
				},
			},
			Role: "server",
			Sk:   "abcdefg",
		},
		testUser + ".p2p_tcp": &XTCPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".p2p_tcp",
				ProxyType: consts.XTCPProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "127.0.0.1",
					LocalPort: 22,
				},
			},
			Role: "server",
			Sk:   "abcdefg",
		},
		testUser + ".tcpmuxhttpconnect": &TCPMuxProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".tcpmuxhttpconnect",
				ProxyType: consts.TCPMuxProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP:   "127.0.0.1",
					LocalPort: 10701,
				},
			},
			DomainConf: DomainConf{
				CustomDomains: []string{"tunnel1"},
				SubDomain:     "",
			},
			Multiplexer: "httpconnect",
		},
		testUser + ".plugin_unix_domain_socket": &TCPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".plugin_unix_domain_socket",
				ProxyType: consts.TCPProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP: "127.0.0.1",
					Plugin:  "unix_domain_socket",
					PluginParams: map[string]string{
						"plugin_unix_path": "/var/run/docker.sock",
					},
				},
			},
			RemotePort: 6003,
		},
		testUser + ".plugin_http_proxy": &TCPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".plugin_http_proxy",
				ProxyType: consts.TCPProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP: "127.0.0.1",
					Plugin:  "http_proxy",
					PluginParams: map[string]string{
						"plugin_http_user":   "abc",
						"plugin_http_passwd": "abc",
					},
				},
			},
			RemotePort: 6004,
		},
		testUser + ".plugin_socks5": &TCPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".plugin_socks5",
				ProxyType: consts.TCPProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP: "127.0.0.1",
					Plugin:  "socks5",
					PluginParams: map[string]string{
						"plugin_user":   "abc",
						"plugin_passwd": "abc",
					},
				},
			},
			RemotePort: 6005,
		},
		testUser + ".plugin_static_file": &TCPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".plugin_static_file",
				ProxyType: consts.TCPProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP: "127.0.0.1",
					Plugin:  "static_file",
					PluginParams: map[string]string{
						"plugin_local_path":   "/var/www/blog",
						"plugin_strip_prefix": "static",
						"plugin_http_user":    "abc",
						"plugin_http_passwd":  "abc",
					},
				},
			},
			RemotePort: 6006,
		},
		testUser + ".plugin_https2http": &HTTPSProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".plugin_https2http",
				ProxyType: consts.HTTPSProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP: "127.0.0.1",
					Plugin:  "https2http",
					PluginParams: map[string]string{
						"plugin_local_addr":          "127.0.0.1:80",
						"plugin_crt_path":            "./server.crt",
						"plugin_key_path":            "./server.key",
						"plugin_host_header_rewrite": "127.0.0.1",
						"plugin_header_X-From-Where": "frp",
					},
				},
			},
			DomainConf: DomainConf{
				CustomDomains: []string{"test.yourdomain.com"},
			},
		},
		testUser + ".plugin_http2https": &HTTPProxyConf{
			BaseProxyConf: BaseProxyConf{
				ProxyName: testUser + ".plugin_http2https",
				ProxyType: consts.HTTPProxy,
				LocalSvrConf: LocalSvrConf{
					LocalIP: "127.0.0.1",
					Plugin:  "http2https",
					PluginParams: map[string]string{
						"plugin_local_addr":          "127.0.0.1:443",
						"plugin_host_header_rewrite": "127.0.0.1",
						"plugin_header_X-From-Where": "frp",
					},
				},
			},
			DomainConf: DomainConf{
				CustomDomains: []string{"test.yourdomain.com"},
			},
		},
	}

	visitorExpected := map[string]VisitorConf{
		testUser + ".secret_tcp_visitor": &STCPVisitorConf{
			BaseVisitorConf: BaseVisitorConf{
				ProxyName:  testUser + ".secret_tcp_visitor",
				ProxyType:  consts.STCPProxy,
				Role:       "visitor",
				Sk:         "abcdefg",
				ServerName: testVisitorPrefix + "secret_tcp",
				BindAddr:   "127.0.0.1",
				BindPort:   9000,
			},
		},
		testUser + ".p2p_tcp_visitor": &XTCPVisitorConf{
			BaseVisitorConf: BaseVisitorConf{
				ProxyName:  testUser + ".p2p_tcp_visitor",
				ProxyType:  consts.XTCPProxy,
				Role:       "visitor",
				Sk:         "abcdefg",
				ServerName: testProxyPrefix + "p2p_tcp",
				BindAddr:   "127.0.0.1",
				BindPort:   9001,
			},
		},
	}

	proxyActual, visitorActual, err := LoadAllProxyConfsFromIni(testUser, testClientBytesWithFull, nil)
	assert.NoError(err)
	assert.Equal(proxyExpected, proxyActual)
	assert.Equal(visitorExpected, visitorActual)

}

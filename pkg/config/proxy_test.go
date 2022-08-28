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
	"gopkg.in/ini.v1"

	"github.com/fatedier/frp/pkg/consts"
)

var (
	testLoadOptions = ini.LoadOptions{
		Insensitive:         false,
		InsensitiveSections: false,
		InsensitiveKeys:     false,
		IgnoreInlineComment: true,
		AllowBooleanKeys:    true,
	}

	testProxyPrefix = "test."
)

func Test_Proxy_Interface(t *testing.T) {
	for name := range proxyConfTypeMap {
		NewConfByType(name)
	}
}

func Test_Proxy_UnmarshalFromIni(t *testing.T) {
	assert := assert.New(t)

	testcases := []struct {
		sname    string
		source   []byte
		expected ProxyConf
	}{
		{
			sname: "ssh",
			source: []byte(`
				[ssh]
				# tcp | udp | http | https | stcp | xtcp, default is tcp
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
				meta_var2 = 234`),
			expected: &TCPProxyConf{
				BaseProxyConf: BaseProxyConf{
					ProxyName:      testProxyPrefix + "ssh",
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
		},
		{
			sname: "ssh_random",
			source: []byte(`
				[ssh_random]
				type = tcp
				local_ip = 127.0.0.9
				local_port = 29
				remote_port = 9
			`),
			expected: &TCPProxyConf{
				BaseProxyConf: BaseProxyConf{
					ProxyName: testProxyPrefix + "ssh_random",
					ProxyType: consts.TCPProxy,
					LocalSvrConf: LocalSvrConf{
						LocalIP:   "127.0.0.9",
						LocalPort: 29,
					},
				},
				RemotePort: 9,
			},
		},
		{
			sname: "dns",
			source: []byte(`
				[dns]
				type = udp
				local_ip = 114.114.114.114
				local_port = 59
				remote_port = 6009
				use_encryption
				use_compression
			`),
			expected: &UDPProxyConf{
				BaseProxyConf: BaseProxyConf{
					ProxyName:      testProxyPrefix + "dns",
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
		},
		{
			sname: "web01",
			source: []byte(`
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
			`),
			expected: &HTTPProxyConf{
				BaseProxyConf: BaseProxyConf{
					ProxyName:      testProxyPrefix + "web01",
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
		},
		{
			sname: "web02",
			source: []byte(`
				[web02]
				type = https
				local_ip = 127.0.0.9
				local_port = 8009
				use_encryption
				use_compression
				subdomain = web01
				custom_domains = web02.yourdomain.com
				proxy_protocol_version = v2
			`),
			expected: &HTTPSProxyConf{
				BaseProxyConf: BaseProxyConf{
					ProxyName:      testProxyPrefix + "web02",
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
		},
		{
			sname: "secret_tcp",
			source: []byte(`
				[secret_tcp]
				type = stcp
				sk = abcdefg
				local_ip = 127.0.0.1
				local_port = 22
				use_encryption = false
				use_compression = false
			`),
			expected: &STCPProxyConf{
				BaseProxyConf: BaseProxyConf{
					ProxyName: testProxyPrefix + "secret_tcp",
					ProxyType: consts.STCPProxy,
					LocalSvrConf: LocalSvrConf{
						LocalIP:   "127.0.0.1",
						LocalPort: 22,
					},
				},
				Role: "server",
				Sk:   "abcdefg",
			},
		},
		{
			sname: "p2p_tcp",
			source: []byte(`
				[p2p_tcp]
				type = xtcp
				sk = abcdefg
				local_ip = 127.0.0.1
				local_port = 22
				use_encryption = false
				use_compression = false
			`),
			expected: &XTCPProxyConf{
				BaseProxyConf: BaseProxyConf{
					ProxyName: testProxyPrefix + "p2p_tcp",
					ProxyType: consts.XTCPProxy,
					LocalSvrConf: LocalSvrConf{
						LocalIP:   "127.0.0.1",
						LocalPort: 22,
					},
				},
				Role: "server",
				Sk:   "abcdefg",
			},
		},
		{
			sname: "tcpmuxhttpconnect",
			source: []byte(`
				[tcpmuxhttpconnect]
				type = tcpmux
				multiplexer = httpconnect
				local_ip = 127.0.0.1
				local_port = 10701
				custom_domains = tunnel1
			`),
			expected: &TCPMuxProxyConf{
				BaseProxyConf: BaseProxyConf{
					ProxyName: testProxyPrefix + "tcpmuxhttpconnect",
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
		},
	}

	for _, c := range testcases {
		f, err := ini.LoadSources(testLoadOptions, c.source)
		assert.NoError(err)

		proxyType := f.Section(c.sname).Key("type").String()
		assert.NotEmpty(proxyType)

		actual := DefaultProxyConf(proxyType)
		assert.NotNil(actual)

		err = actual.UnmarshalFromIni(testProxyPrefix, c.sname, f.Section(c.sname))
		assert.NoError(err)
		assert.Equal(c.expected, actual)
	}
}

func Test_RangeProxy_UnmarshalFromIni(t *testing.T) {
	assert := assert.New(t)

	testcases := []struct {
		sname    string
		source   []byte
		expected map[string]ProxyConf
	}{
		{
			sname: "range:tcp_port",
			source: []byte(`
				[range:tcp_port]
				type = tcp
				local_ip = 127.0.0.9
				local_port = 6010-6011,6019
				remote_port = 6010-6011,6019
				use_encryption = false
				use_compression = false
			`),
			expected: map[string]ProxyConf{
				"tcp_port_0": &TCPProxyConf{
					BaseProxyConf: BaseProxyConf{
						ProxyName: testProxyPrefix + "tcp_port_0",
						ProxyType: consts.TCPProxy,
						LocalSvrConf: LocalSvrConf{
							LocalIP:   "127.0.0.9",
							LocalPort: 6010,
						},
					},
					RemotePort: 6010,
				},
				"tcp_port_1": &TCPProxyConf{
					BaseProxyConf: BaseProxyConf{
						ProxyName: testProxyPrefix + "tcp_port_1",
						ProxyType: consts.TCPProxy,
						LocalSvrConf: LocalSvrConf{
							LocalIP:   "127.0.0.9",
							LocalPort: 6011,
						},
					},
					RemotePort: 6011,
				},
				"tcp_port_2": &TCPProxyConf{
					BaseProxyConf: BaseProxyConf{
						ProxyName: testProxyPrefix + "tcp_port_2",
						ProxyType: consts.TCPProxy,
						LocalSvrConf: LocalSvrConf{
							LocalIP:   "127.0.0.9",
							LocalPort: 6019,
						},
					},
					RemotePort: 6019,
				},
			},
		},
		{
			sname: "range:udp_port",
			source: []byte(`
				[range:udp_port]
				type = udp
				local_ip = 114.114.114.114
				local_port = 6000,6010-6011
				remote_port = 6000,6010-6011
				use_encryption
				use_compression
			`),
			expected: map[string]ProxyConf{
				"udp_port_0": &UDPProxyConf{
					BaseProxyConf: BaseProxyConf{
						ProxyName:      testProxyPrefix + "udp_port_0",
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
				"udp_port_1": &UDPProxyConf{
					BaseProxyConf: BaseProxyConf{
						ProxyName:      testProxyPrefix + "udp_port_1",
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
				"udp_port_2": &UDPProxyConf{
					BaseProxyConf: BaseProxyConf{
						ProxyName:      testProxyPrefix + "udp_port_2",
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
			},
		},
	}

	for _, c := range testcases {

		f, err := ini.LoadSources(testLoadOptions, c.source)
		assert.NoError(err)

		actual := make(map[string]ProxyConf)
		s := f.Section(c.sname)

		err = renderRangeProxyTemplates(f, s)
		assert.NoError(err)

		f.DeleteSection(ini.DefaultSection)
		f.DeleteSection(c.sname)

		for _, section := range f.Sections() {
			proxyType := section.Key("type").String()
			newsname := section.Name()

			tmp := DefaultProxyConf(proxyType)
			err = tmp.UnmarshalFromIni(testProxyPrefix, newsname, section)
			assert.NoError(err)

			actual[newsname] = tmp
		}

		assert.Equal(c.expected, actual)
	}
}

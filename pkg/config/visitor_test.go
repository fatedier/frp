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

	"github.com/fatedier/frp/pkg/consts"

	"github.com/stretchr/testify/assert"
	"gopkg.in/ini.v1"
)

const testVisitorPrefix = "test."

func Test_Visitor_Interface(t *testing.T) {
	for name := range visitorConfTypeMap {
		DefaultVisitorConf(name)
	}
}

func Test_Visitor_UnmarshalFromIni(t *testing.T) {
	assert := assert.New(t)

	testcases := []struct {
		sname    string
		source   []byte
		expected VisitorConf
	}{
		{
			sname: "secret_tcp_visitor",
			source: []byte(`
				[secret_tcp_visitor]
				role = visitor
				type = stcp
				server_name = secret_tcp
				sk = abcdefg
				bind_addr = 127.0.0.1
				bind_port = 9000
				use_encryption = false
				use_compression = false
			`),
			expected: &STCPVisitorConf{
				BaseVisitorConf: BaseVisitorConf{
					ProxyName:  testVisitorPrefix + "secret_tcp_visitor",
					ProxyType:  consts.STCPProxy,
					Role:       "visitor",
					Sk:         "abcdefg",
					ServerName: testVisitorPrefix + "secret_tcp",
					BindAddr:   "127.0.0.1",
					BindPort:   9000,
				},
			},
		},
		{
			sname: "p2p_tcp_visitor",
			source: []byte(`
				[p2p_tcp_visitor]
				role = visitor
				type = xtcp
				server_name = p2p_tcp
				sk = abcdefg
				bind_addr = 127.0.0.1
				bind_port = 9001
				use_encryption = false
				use_compression = false
			`),
			expected: &XTCPVisitorConf{
				BaseVisitorConf: BaseVisitorConf{
					ProxyName:  testVisitorPrefix + "p2p_tcp_visitor",
					ProxyType:  consts.XTCPProxy,
					Role:       "visitor",
					Sk:         "abcdefg",
					ServerName: testProxyPrefix + "p2p_tcp",
					BindAddr:   "127.0.0.1",
					BindPort:   9001,
				},
			},
		},
	}

	for _, c := range testcases {
		f, err := ini.LoadSources(testLoadOptions, c.source)
		assert.NoError(err)

		visitorType := f.Section(c.sname).Key("type").String()
		assert.NotEmpty(visitorType)

		actual := DefaultVisitorConf(visitorType)
		assert.NotNil(actual)

		err = actual.UnmarshalFromIni(testVisitorPrefix, c.sname, f.Section(c.sname))
		assert.NoError(err)
		assert.Equal(c.expected, actual)
	}
}

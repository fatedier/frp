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
	for _, content := range []string{tomlServerContent, yamlServerContent, jsonServerContent} {
		svrCfg := v1.ServerConfig{}
		err := LoadConfigure([]byte(content), &svrCfg)
		require := require.New(t)
		require.NoError(err)
		require.EqualValues("127.0.0.1", svrCfg.BindAddr)
		require.EqualValues(7000, svrCfg.KCPBindPort)
		require.EqualValues(7001, svrCfg.QUICBindPort)
		require.EqualValues(7005, svrCfg.TCPMuxHTTPConnectPort)
		require.EqualValues("/abc.html", svrCfg.Custom404Page)
		require.EqualValues(10, svrCfg.Transport.TCPKeepAlive)
	}
}

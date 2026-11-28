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

package legacy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertServerCommonConfPprofEnable(t *testing.T) {
	tests := []struct {
		name string
		ini  string
	}{
		{
			name: "dashboard without tls",
			ini: `[common]
dashboard_port = 7500
pprof_enable = true
`,
		},
		{
			name: "dashboard with tls",
			ini: `[common]
dashboard_port = 7500
dashboard_tls_mode = true
dashboard_tls_cert_file = server.crt
dashboard_tls_key_file = server.key
pprof_enable = true
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			legacyCfg, err := UnmarshalServerConfFromIni([]byte(tt.ini))
			require.NoError(err)
			require.True(legacyCfg.PprofEnable)

			cfg := Convert_ServerCommonConf_To_v1(&legacyCfg)
			require.True(cfg.WebServer.PprofEnable)
		})
	}

	t.Run("disabled by default", func(t *testing.T) {
		require := require.New(t)

		legacyCfg, err := UnmarshalServerConfFromIni([]byte("[common]\ndashboard_port = 7500\n"))
		require.NoError(err)

		cfg := Convert_ServerCommonConf_To_v1(&legacyCfg)
		require.False(cfg.WebServer.PprofEnable)
	})
}

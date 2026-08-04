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

package sub

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/policy/security"
)

func TestVerifyClientConfigFeatureGates(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "VirtualNet enabled",
			content: `featureGates = { VirtualNet = true }
virtualNet.address = "100.86.0.4/24"
`,
		},
		{
			name: "VirtualNet disabled",
			content: `featureGates = { VirtualNet = false }
virtualNet.address = "100.86.0.4/24"
`,
			wantErr: "VirtualNet feature is not enabled",
		},
		{
			name:    "unknown feature gate",
			content: `featureGates = { UnknownFeature = true }`,
			wantErr: "unrecognized feature gate: UnknownFeature",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "frpc.toml")
			require.NoError(t, os.WriteFile(configFile, []byte(tc.content), 0o600))

			warning, err := verifyClientConfig(configFile, true, security.NewUnsafeFeatures(nil))
			require.NoError(t, warning)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

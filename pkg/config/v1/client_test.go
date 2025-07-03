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
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestClientConfigComplete(t *testing.T) {
	require := require.New(t)
	c := &ClientConfig{}
	err := c.Complete()
	require.NoError(err)

	require.EqualValues("token", c.Auth.Method)
	require.Equal(true, lo.FromPtr(c.Transport.TCPMux))
	require.Equal(true, lo.FromPtr(c.LoginFailExit))
	require.Equal(true, lo.FromPtr(c.Transport.TLS.Enable))
	require.Equal(true, lo.FromPtr(c.Transport.TLS.DisableCustomTLSFirstByte))
	require.NotEmpty(c.NatHoleSTUNServer)
}

func TestAuthClientConfig_Complete(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_token")
	testContent := "client-token-value"
	err := os.WriteFile(testFile, []byte(testContent), 0o600)
	require.NoError(t, err)

	tests := []struct {
		name        string
		config      AuthClientConfig
		expectToken string
		expectPanic bool
	}{
		{
			name: "tokenSource resolved to token",
			config: AuthClientConfig{
				Method: AuthMethodToken,
				TokenSource: &ValueSource{
					Type: "file",
					File: &FileSource{
						Path: testFile,
					},
				},
			},
			expectToken: testContent,
			expectPanic: false,
		},
		{
			name: "direct token unchanged",
			config: AuthClientConfig{
				Method: AuthMethodToken,
				Token:  "direct-token",
			},
			expectToken: "direct-token",
			expectPanic: false,
		},
		{
			name: "invalid tokenSource should panic",
			config: AuthClientConfig{
				Method: AuthMethodToken,
				TokenSource: &ValueSource{
					Type: "file",
					File: &FileSource{
						Path: "/non/existent/file",
					},
				},
			},
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				err := tt.config.Complete()
				require.Error(t, err)
			} else {
				err := tt.config.Complete()
				require.NoError(t, err)
				require.Equal(t, tt.expectToken, tt.config.Token)
				require.Nil(t, tt.config.TokenSource, "TokenSource should be cleared after resolution")
			}
		})
	}
}

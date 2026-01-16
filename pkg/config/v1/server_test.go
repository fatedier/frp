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
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestServerConfigComplete(t *testing.T) {
	require := require.New(t)
	c := &ServerConfig{}
	err := c.Complete()
	require.NoError(err)

	require.EqualValues("token", c.Auth.Method)
	require.Equal(true, lo.FromPtr(c.Transport.TCPMux))
	require.Equal(true, lo.FromPtr(c.DetailedErrorsToClient))
}

func TestServerConfigComplete_TLSForceWithTrustedCaFile(t *testing.T) {
	// Test: When TrustedCaFile is set and Force is nil, Force should default to true
	t.Run("default Force to true when trustedCaFile set and Force is nil", func(t *testing.T) {
		require := require.New(t)
		c := &ServerConfig{}
		c.Transport.TLS.TrustedCaFile = "/path/to/ca.crt"
		err := c.Complete()
		require.NoError(err)
		require.Equal(true, lo.FromPtr(c.Transport.TLS.Force))
	})

	// Test: When TrustedCaFile is set but Force is explicitly set to false, preserve false
	t.Run("preserve explicit Force=false with trustedCaFile", func(t *testing.T) {
		require := require.New(t)
		c := &ServerConfig{}
		c.Transport.TLS.TrustedCaFile = "/path/to/ca.crt"
		c.Transport.TLS.Force = lo.ToPtr(false) // Explicitly set to false
		err := c.Complete()
		require.NoError(err)
		require.Equal(false, lo.FromPtr(c.Transport.TLS.Force))
	})

	// Test: When TrustedCaFile is set and Force is explicitly set to true, preserve true
	t.Run("preserve explicit Force=true with trustedCaFile", func(t *testing.T) {
		require := require.New(t)
		c := &ServerConfig{}
		c.Transport.TLS.TrustedCaFile = "/path/to/ca.crt"
		c.Transport.TLS.Force = lo.ToPtr(true) // Explicitly set to true
		err := c.Complete()
		require.NoError(err)
		require.Equal(true, lo.FromPtr(c.Transport.TLS.Force))
	})

	// Test: When TrustedCaFile is NOT set, Force should remain nil (default)
	t.Run("Force remains nil when no trustedCaFile", func(t *testing.T) {
		require := require.New(t)
		c := &ServerConfig{}
		err := c.Complete()
		require.NoError(err)
		require.Nil(c.Transport.TLS.Force)
	})
}

func TestAuthServerConfig_Complete(t *testing.T) {
	require := require.New(t)
	cfg := &AuthServerConfig{}
	err := cfg.Complete()
	require.NoError(err)
	require.EqualValues("token", cfg.Method)
}


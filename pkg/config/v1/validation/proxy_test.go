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

package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestValidateHTTPProxyRequiresVhostHTTPPort(t *testing.T) {
	require := require.New(t)

	cfg := &v1.HTTPProxyConfig{}
	err := ValidateProxyConfigurerForServer(cfg, &v1.ServerConfig{})
	require.Error(err)
	require.Contains(err.Error(), "type [http] not supported when vhost http port is not set")
	require.Contains(err.Error(), "vhostHTTPPort")

	err = ValidateProxyConfigurerForServer(cfg, &v1.ServerConfig{VhostHTTPPort: 80})
	// Domain validation may still fail, but not because the vhost port is missing.
	if err != nil {
		require.False(strings.Contains(err.Error(), "vhost http port is not set"))
	}
}

func TestValidateHTTPSProxyRequiresVhostHTTPSPort(t *testing.T) {
	require := require.New(t)

	cfg := &v1.HTTPSProxyConfig{}
	err := ValidateProxyConfigurerForServer(cfg, &v1.ServerConfig{})
	require.Error(err)
	require.Contains(err.Error(), "type [https] not supported when vhost https port is not set")
	require.Contains(err.Error(), "vhostHTTPSPort")
}

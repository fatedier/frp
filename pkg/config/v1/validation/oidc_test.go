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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestValidateOIDCClientCredentialsConfig(t *testing.T) {
	tokenServer := httptest.NewServer(http.NotFoundHandler())
	defer tokenServer.Close()

	t.Run("valid", func(t *testing.T) {
		require.NoError(t, ValidateOIDCClientCredentialsConfig(&v1.AuthOIDCClientConfig{
			ClientID:         "test-client",
			TokenEndpointURL: tokenServer.URL,
			AdditionalEndpointParams: map[string]string{
				"resource": "api",
			},
		}))
	})

	t.Run("invalid token endpoint url", func(t *testing.T) {
		err := ValidateOIDCClientCredentialsConfig(&v1.AuthOIDCClientConfig{
			ClientID:         "test-client",
			TokenEndpointURL: "://bad",
		})
		require.ErrorContains(t, err, "auth.oidc.tokenEndpointURL")
	})

	t.Run("missing client id", func(t *testing.T) {
		err := ValidateOIDCClientCredentialsConfig(&v1.AuthOIDCClientConfig{
			TokenEndpointURL: tokenServer.URL,
		})
		require.ErrorContains(t, err, "auth.oidc.clientID is required")
	})

	t.Run("scope endpoint param is not allowed", func(t *testing.T) {
		err := ValidateOIDCClientCredentialsConfig(&v1.AuthOIDCClientConfig{
			ClientID:         "test-client",
			TokenEndpointURL: tokenServer.URL,
			AdditionalEndpointParams: map[string]string{
				"scope": "email",
			},
		})
		require.ErrorContains(t, err, "auth.oidc.additionalEndpointParams.scope is not allowed; use auth.oidc.scope instead")
	})

	t.Run("audience conflict", func(t *testing.T) {
		err := ValidateOIDCClientCredentialsConfig(&v1.AuthOIDCClientConfig{
			ClientID:         "test-client",
			TokenEndpointURL: tokenServer.URL,
			Audience:         "api",
			AdditionalEndpointParams: map[string]string{
				"audience": "override",
			},
		})
		require.ErrorContains(t, err, "cannot specify both auth.oidc.audience and auth.oidc.additionalEndpointParams.audience")
	})
}

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
	"errors"
	"net/url"
	"strings"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func ValidateOIDCClientCredentialsConfig(c *v1.AuthOIDCClientConfig) error {
	var errs []string

	if c.ClientID == "" {
		errs = append(errs, "auth.oidc.clientID is required")
	}

	if c.TokenEndpointURL == "" {
		errs = append(errs, "auth.oidc.tokenEndpointURL is required")
	} else {
		tokenURL, err := url.Parse(c.TokenEndpointURL)
		if err != nil || !tokenURL.IsAbs() || tokenURL.Host == "" {
			errs = append(errs, "auth.oidc.tokenEndpointURL must be an absolute http or https URL")
		} else if tokenURL.Scheme != "http" && tokenURL.Scheme != "https" {
			errs = append(errs, "auth.oidc.tokenEndpointURL must use http or https")
		}
	}

	if _, ok := c.AdditionalEndpointParams["scope"]; ok {
		errs = append(errs, "auth.oidc.additionalEndpointParams.scope is not allowed; use auth.oidc.scope instead")
	}

	if c.Audience != "" {
		if _, ok := c.AdditionalEndpointParams["audience"]; ok {
			errs = append(errs, "cannot specify both auth.oidc.audience and auth.oidc.additionalEndpointParams.audience")
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

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

package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/samber/lo"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/policy/featuregate"
	"github.com/fatedier/frp/pkg/policy/security"
)

func (v *ConfigValidator) ValidateClientCommonConfig(c *v1.ClientCommonConfig) (Warning, error) {
	var (
		warnings Warning
		errs     error
	)

	validators := []func() (Warning, error){
		func() (Warning, error) { return validateFeatureGates(c) },
		func() (Warning, error) { return v.validateAuthConfig(&c.Auth) },
		func() (Warning, error) { return nil, validateLogConfig(&c.Log) },
		func() (Warning, error) { return nil, validateWebServerConfig(&c.WebServer) },
		func() (Warning, error) { return validateTransportConfig(&c.Transport) },
		func() (Warning, error) { return validateIncludeFiles(c.IncludeConfigFiles) },
	}

	for _, validator := range validators {
		w, err := validator()
		warnings = AppendError(warnings, w)
		errs = AppendError(errs, err)
	}
	return warnings, errs
}

func validateFeatureGates(c *v1.ClientCommonConfig) (Warning, error) {
	if c.VirtualNet.Address != "" {
		if !featuregate.Enabled(featuregate.VirtualNet) {
			return nil, fmt.Errorf("VirtualNet feature is not enabled; enable it by setting the appropriate feature gate flag")
		}
	}
	return nil, nil
}

func (v *ConfigValidator) validateAuthConfig(c *v1.AuthClientConfig) (Warning, error) {
	var errs error
	if !slices.Contains(SupportedAuthMethods, c.Method) {
		errs = AppendError(errs, fmt.Errorf("invalid auth method, optional values are %v", SupportedAuthMethods))
	}
	if !lo.Every(SupportedAuthAdditionalScopes, c.AdditionalScopes) {
		errs = AppendError(errs, fmt.Errorf("invalid auth additional scopes, optional values are %v", SupportedAuthAdditionalScopes))
	}

	// Validate token/tokenSource mutual exclusivity
	if c.Token != "" && c.TokenSource != nil {
		errs = AppendError(errs, fmt.Errorf("cannot specify both auth.token and auth.tokenSource"))
	}

	// Validate tokenSource if specified
	if c.TokenSource != nil {
		if c.TokenSource.Type == "exec" {
			if err := v.ValidateUnsafeFeature(security.TokenSourceExec); err != nil {
				errs = AppendError(errs, err)
			}
		}
		if err := c.TokenSource.Validate(); err != nil {
			errs = AppendError(errs, fmt.Errorf("invalid auth.tokenSource: %v", err))
		}
	}

	if err := v.validateOIDCConfig(&c.OIDC); err != nil {
		errs = AppendError(errs, err)
	}
	return nil, errs
}

func (v *ConfigValidator) validateOIDCConfig(c *v1.AuthOIDCClientConfig) error {
	if c.TokenSource == nil {
		return nil
	}
	var errs error
	// Validate oidc.tokenSource mutual exclusivity with other fields of oidc
	if c.ClientID != "" || c.ClientSecret != "" || c.Audience != "" ||
		c.Scope != "" || c.TokenEndpointURL != "" || len(c.AdditionalEndpointParams) > 0 ||
		c.TrustedCaFile != "" || c.InsecureSkipVerify || c.ProxyURL != "" {
		errs = AppendError(errs, fmt.Errorf("cannot specify both auth.oidc.tokenSource and any other field of auth.oidc"))
	}
	if c.TokenSource.Type == "exec" {
		if err := v.ValidateUnsafeFeature(security.TokenSourceExec); err != nil {
			errs = AppendError(errs, err)
		}
	}
	if err := c.TokenSource.Validate(); err != nil {
		errs = AppendError(errs, fmt.Errorf("invalid auth.oidc.tokenSource: %v", err))
	}
	return errs
}

func validateTransportConfig(c *v1.ClientTransportConfig) (Warning, error) {
	var (
		warnings Warning
		errs     error
	)

	if c.HeartbeatTimeout > 0 && c.HeartbeatInterval > 0 {
		if c.HeartbeatTimeout < c.HeartbeatInterval {
			errs = AppendError(errs, fmt.Errorf("invalid transport.heartbeatTimeout, heartbeat timeout should not less than heartbeat interval"))
		}
	}

	if !lo.FromPtr(c.TLS.Enable) {
		checkTLSConfig := func(name string, value string) Warning {
			if value != "" {
				return fmt.Errorf("%s is invalid when transport.tls.enable is false", name)
			}
			return nil
		}

		warnings = AppendError(warnings, checkTLSConfig("transport.tls.certFile", c.TLS.CertFile))
		warnings = AppendError(warnings, checkTLSConfig("transport.tls.keyFile", c.TLS.KeyFile))
		warnings = AppendError(warnings, checkTLSConfig("transport.tls.trustedCaFile", c.TLS.TrustedCaFile))
	}

	if !slices.Contains(SupportedTransportProtocols, c.Protocol) {
		errs = AppendError(errs, fmt.Errorf("invalid transport.protocol, optional values are %v", SupportedTransportProtocols))
	}
	return warnings, errs
}

func validateIncludeFiles(files []string) (Warning, error) {
	var errs error
	for _, f := range files {
		absDir, err := filepath.Abs(filepath.Dir(f))
		if err != nil {
			errs = AppendError(errs, fmt.Errorf("include: parse directory of %s failed: %v", f, err))
			continue
		}
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			errs = AppendError(errs, fmt.Errorf("include: directory of %s not exist", f))
		}
	}
	return nil, errs
}

func ValidateAllClientConfig(
	c *v1.ClientCommonConfig,
	proxyCfgs []v1.ProxyConfigurer,
	visitorCfgs []v1.VisitorConfigurer,
	unsafeFeatures *security.UnsafeFeatures,
) (Warning, error) {
	validator := NewConfigValidator(unsafeFeatures)
	var warnings Warning
	if c != nil {
		warning, err := validator.ValidateClientCommonConfig(c)
		warnings = AppendError(warnings, warning)
		if err != nil {
			return warnings, err
		}
	}

	for _, c := range proxyCfgs {
		if err := ValidateProxyConfigurerForClient(c); err != nil {
			return warnings, fmt.Errorf("proxy %s: %v", c.GetBaseConfig().Name, err)
		}
	}

	for _, c := range visitorCfgs {
		if err := ValidateVisitorConfigurer(c); err != nil {
			return warnings, fmt.Errorf("visitor %s: %v", c.GetBaseConfig().Name, err)
		}
	}
	return warnings, nil
}

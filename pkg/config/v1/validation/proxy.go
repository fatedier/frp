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
	"errors"
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func validateProxyBaseConfigForClient(c *v1.ProxyBaseConfig) error {
	if c.Name == "" {
		return errors.New("name should not be empty")
	}

	if err := ValidateAnnotations(c.Annotations); err != nil {
		return err
	}
	if !slices.Contains([]string{"", "v1", "v2"}, c.Transport.ProxyProtocolVersion) {
		return fmt.Errorf("not support proxy protocol version: %s", c.Transport.ProxyProtocolVersion)
	}
	if !slices.Contains([]string{"client", "server"}, c.Transport.BandwidthLimitMode) {
		return fmt.Errorf("bandwidth limit mode should be client or server")
	}

	if c.Plugin.Type == "" {
		if err := ValidatePort(c.LocalPort, "localPort"); err != nil {
			return fmt.Errorf("localPort: %v", err)
		}
	}

	if !slices.Contains([]string{"", "tcp", "http"}, c.HealthCheck.Type) {
		return fmt.Errorf("not support health check type: %s", c.HealthCheck.Type)
	}
	if c.HealthCheck.Type != "" {
		if c.HealthCheck.Type == "http" &&
			c.HealthCheck.Path == "" {
			return fmt.Errorf("health check path should not be empty")
		}
	}

	if c.Plugin.Type != "" {
		if err := ValidateClientPluginOptions(c.Plugin.ClientPluginOptions); err != nil {
			return fmt.Errorf("plugin %s: %v", c.Plugin.Type, err)
		}
	}
	return nil
}

func validateProxyBaseConfigForServer(c *v1.ProxyBaseConfig) error {
	if err := ValidateAnnotations(c.Annotations); err != nil {
		return err
	}
	return nil
}

func validateDomainConfigForClient(c *v1.DomainConfig) error {
	if c.SubDomain == "" && len(c.CustomDomains) == 0 {
		return errors.New("subdomain and custom domains should not be both empty")
	}
	return nil
}

func validateDomainConfigForServer(c *v1.DomainConfig, s *v1.ServerConfig) error {
	for _, domain := range c.CustomDomains {
		if s.SubDomainHost != "" && len(strings.Split(s.SubDomainHost, ".")) < len(strings.Split(domain, ".")) {
			if strings.Contains(domain, s.SubDomainHost) {
				return fmt.Errorf("custom domain [%s] should not belong to subdomain host [%s]", domain, s.SubDomainHost)
			}
		}
	}

	if c.SubDomain != "" {
		if s.SubDomainHost == "" {
			return errors.New("subdomain is not supported because this feature is not enabled in server")
		}

		if strings.Contains(c.SubDomain, ".") || strings.Contains(c.SubDomain, "*") {
			return errors.New("'.' and '*' are not supported in subdomain")
		}
	}
	return nil
}

func ValidateProxyConfigurerForClient(c v1.ProxyConfigurer) error {
	base := c.GetBaseConfig()
	if err := validateProxyBaseConfigForClient(base); err != nil {
		return err
	}

	switch v := c.(type) {
	case *v1.TCPProxyConfig:
		return validateTCPProxyConfigForClient(v)
	case *v1.UDPProxyConfig:
		return validateUDPProxyConfigForClient(v)
	case *v1.TCPMuxProxyConfig:
		return validateTCPMuxProxyConfigForClient(v)
	case *v1.HTTPProxyConfig:
		return validateHTTPProxyConfigForClient(v)
	case *v1.HTTPSProxyConfig:
		return validateHTTPSProxyConfigForClient(v)
	case *v1.STCPProxyConfig:
		return validateSTCPProxyConfigForClient(v)
	case *v1.XTCPProxyConfig:
		return validateXTCPProxyConfigForClient(v)
	case *v1.SUDPProxyConfig:
		return validateSUDPProxyConfigForClient(v)
	}
	return errors.New("unknown proxy config type")
}

func validateTCPProxyConfigForClient(c *v1.TCPProxyConfig) error {
	return nil
}

func validateUDPProxyConfigForClient(c *v1.UDPProxyConfig) error {
	return nil
}

func validateTCPMuxProxyConfigForClient(c *v1.TCPMuxProxyConfig) error {
	if err := validateDomainConfigForClient(&c.DomainConfig); err != nil {
		return err
	}

	if !slices.Contains([]string{string(v1.TCPMultiplexerHTTPConnect)}, c.Multiplexer) {
		return fmt.Errorf("not support multiplexer: %s", c.Multiplexer)
	}
	return nil
}

func validateHTTPProxyConfigForClient(c *v1.HTTPProxyConfig) error {
	return validateDomainConfigForClient(&c.DomainConfig)
}

func validateHTTPSProxyConfigForClient(c *v1.HTTPSProxyConfig) error {
	return validateDomainConfigForClient(&c.DomainConfig)
}

func validateSTCPProxyConfigForClient(c *v1.STCPProxyConfig) error {
	return nil
}

func validateXTCPProxyConfigForClient(c *v1.XTCPProxyConfig) error {
	return nil
}

func validateSUDPProxyConfigForClient(c *v1.SUDPProxyConfig) error {
	return nil
}

func ValidateProxyConfigurerForServer(c v1.ProxyConfigurer, s *v1.ServerConfig) error {
	base := c.GetBaseConfig()
	if err := validateProxyBaseConfigForServer(base); err != nil {
		return err
	}

	switch v := c.(type) {
	case *v1.TCPProxyConfig:
		return validateTCPProxyConfigForServer(v, s)
	case *v1.UDPProxyConfig:
		return validateUDPProxyConfigForServer(v, s)
	case *v1.TCPMuxProxyConfig:
		return validateTCPMuxProxyConfigForServer(v, s)
	case *v1.HTTPProxyConfig:
		return validateHTTPProxyConfigForServer(v, s)
	case *v1.HTTPSProxyConfig:
		return validateHTTPSProxyConfigForServer(v, s)
	case *v1.STCPProxyConfig:
		return validateSTCPProxyConfigForServer(v, s)
	case *v1.XTCPProxyConfig:
		return validateXTCPProxyConfigForServer(v, s)
	case *v1.SUDPProxyConfig:
		return validateSUDPProxyConfigForServer(v, s)
	default:
		return errors.New("unknown proxy config type")
	}
}

func validateTCPProxyConfigForServer(c *v1.TCPProxyConfig, s *v1.ServerConfig) error {
	return nil
}

func validateUDPProxyConfigForServer(c *v1.UDPProxyConfig, s *v1.ServerConfig) error {
	return nil
}

func validateTCPMuxProxyConfigForServer(c *v1.TCPMuxProxyConfig, s *v1.ServerConfig) error {
	if c.Multiplexer == string(v1.TCPMultiplexerHTTPConnect) &&
		s.TCPMuxHTTPConnectPort == 0 {
		return fmt.Errorf("tcpmux with multiplexer httpconnect not supported because this feature is not enabled in server")
	}

	return validateDomainConfigForServer(&c.DomainConfig, s)
}

func validateHTTPProxyConfigForServer(c *v1.HTTPProxyConfig, s *v1.ServerConfig) error {
	if s.VhostHTTPPort == 0 {
		return fmt.Errorf("type [http] not supported when vhost http port is not set")
	}

	return validateDomainConfigForServer(&c.DomainConfig, s)
}

func validateHTTPSProxyConfigForServer(c *v1.HTTPSProxyConfig, s *v1.ServerConfig) error {
	if s.VhostHTTPSPort == 0 {
		return fmt.Errorf("type [https] not supported when vhost https port is not set")
	}

	return validateDomainConfigForServer(&c.DomainConfig, s)
}

func validateSTCPProxyConfigForServer(c *v1.STCPProxyConfig, s *v1.ServerConfig) error {
	return nil
}

func validateXTCPProxyConfigForServer(c *v1.XTCPProxyConfig, s *v1.ServerConfig) error {
	return nil
}

func validateSUDPProxyConfigForServer(c *v1.SUDPProxyConfig, s *v1.ServerConfig) error {
	return nil
}

// ValidateAnnotations validates that a set of annotations are correctly defined.
func ValidateAnnotations(annotations map[string]string) error {
	if len(annotations) == 0 {
		return nil
	}

	var errs error
	for k := range annotations {
		for _, msg := range validation.IsQualifiedName(strings.ToLower(k)) {
			errs = AppendError(errs, fmt.Errorf("annotation key %s is invalid: %s", k, msg))
		}
	}
	if err := ValidateAnnotationsSize(annotations); err != nil {
		errs = AppendError(errs, err)
	}
	return errs
}

const TotalAnnotationSizeLimitB int = 256 * (1 << 10) // 256 kB

func ValidateAnnotationsSize(annotations map[string]string) error {
	var totalSize int64
	for k, v := range annotations {
		totalSize += (int64)(len(k)) + (int64)(len(v))
	}
	if totalSize > (int64)(TotalAnnotationSizeLimitB) {
		return fmt.Errorf("annotations size %d is larger than limit %d", totalSize, TotalAnnotationSizeLimitB)
	}
	return nil
}

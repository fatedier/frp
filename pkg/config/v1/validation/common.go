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
	"slices"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func validateWebServerConfig(c *v1.WebServerConfig) error {
	if c.TLS != nil {
		if c.TLS.UseSystemCa {
			return fmt.Errorf("webServer.tls.useSystemCa is only supported in frpc transport.tls")
		}
		if c.TLS.CertFile == "" {
			return fmt.Errorf("tls.certFile must be specified when tls is enabled")
		}
		if c.TLS.KeyFile == "" {
			return fmt.Errorf("tls.keyFile must be specified when tls is enabled")
		}
	}

	return ValidatePort(c.Port, "webServer.port")
}

func validateMetricsServerConfig(c *v1.MetricsServerConfig, web *v1.WebServerConfig) error {
	if err := ValidatePort(c.Port, "metricsServer.port"); err != nil {
		return err
	}
	if c.TLS != nil {
		if c.TLS.UseSystemCa {
			return fmt.Errorf("metricsServer.tls.useSystemCa is only supported in frpc transport.tls")
		}
		if c.TLS.CertFile == "" || c.TLS.KeyFile == "" {
			return fmt.Errorf("metricsServer.tls.certFile and metricsServer.tls.keyFile must both be specified when tls is enabled")
		}
	}
	// Only reject an exact duplicate. Whether two different addresses can share
	// a port depends on the address family and the platform, so leave it to
	// net.Listen to decide.
	if c.Port > 0 && web.Port > 0 && c.Port == web.Port && c.Addr == web.Addr {
		return fmt.Errorf("metricsServer.addr:port conflicts with webServer.addr:port")
	}
	return nil
}

// ValidatePort checks that the network port is in range
func ValidatePort(port int, fieldPath string) error {
	if 0 <= port && port <= 65535 {
		return nil
	}
	return fmt.Errorf("%s: port number %d must be in the range 0..65535", fieldPath, port)
}

func validateLogConfig(c *v1.LogConfig) error {
	if !slices.Contains(SupportedLogLevels, c.Level) {
		return fmt.Errorf("invalid log level, optional values are %v", SupportedLogLevels)
	}
	return nil
}

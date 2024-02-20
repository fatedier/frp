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
		if c.TLS.CertFile == "" {
			return fmt.Errorf("tls.certFile must be specified when tls is enabled")
		}
		if c.TLS.KeyFile == "" {
			return fmt.Errorf("tls.keyFile must be specified when tls is enabled")
		}
	}

	return ValidatePort(c.Port, "webServer.port")
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

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
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func ValidateServerConfig(c *v1.ServerConfig) (Warning, error) {
	var (
		warnings Warning
		errs     error
	)
	if err := validateWebServerConfig(&c.WebServer); err != nil {
		errs = AppendError(errs, err)
	}

	errs = AppendError(errs, ValidatePort(c.BindPort))
	errs = AppendError(errs, ValidatePort(c.KCPBindPort))
	errs = AppendError(errs, ValidatePort(c.QUICBindPort))
	errs = AppendError(errs, ValidatePort(c.VhostHTTPPort))
	errs = AppendError(errs, ValidatePort(c.VhostHTTPSPort))
	errs = AppendError(errs, ValidatePort(c.TCPMuxHTTPConnectPort))
	return warnings, errs
}

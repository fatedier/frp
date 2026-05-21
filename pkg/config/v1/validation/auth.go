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
	"fmt"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/policy/security"
)

func (v *ConfigValidator) validateAuthTokenSource(token string, tokenSource *v1.ValueSource) error {
	var errs error
	// Preserve the previous client/server validation order for joined errors.
	if token != "" && tokenSource != nil {
		errs = AppendError(errs, fmt.Errorf("cannot specify both auth.token and auth.tokenSource"))
	}
	if tokenSource == nil {
		return errs
	}

	if tokenSource.Type == "exec" {
		if err := v.ValidateUnsafeFeature(security.TokenSourceExec); err != nil {
			errs = AppendError(errs, err)
		}
	}
	if err := tokenSource.Validate(); err != nil {
		errs = AppendError(errs, fmt.Errorf("invalid auth.tokenSource: %v", err))
	}
	return errs
}

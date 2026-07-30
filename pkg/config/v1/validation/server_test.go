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
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestValidateServerConfigMaxPoolCount(t *testing.T) {
	for _, tc := range []struct {
		name         string
		maxPoolCount int64
		wantErr      bool
	}{
		{name: "negative", maxPoolCount: -1, wantErr: true},
		{name: "zero", maxPoolCount: 0},
		{name: "positive", maxPoolCount: 5},
		{name: "maximum int64", maxPoolCount: math.MaxInt64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validServerConfigWithAuth(v1.AuthServerConfig{Method: v1.AuthMethodToken})
			cfg.Transport.MaxPoolCount = tc.maxPoolCount
			require.NoError(t, cfg.Complete())

			_, err := NewConfigValidator(nil).ValidateServerConfig(cfg)
			if tc.wantErr {
				require.ErrorContains(t, err, "invalid transport.maxPoolCount")
				require.ErrorContains(t, err, "must be non-negative")
				return
			}
			require.NoError(t, err)
		})
	}
}

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
)

func TestValidateIdentifiers(t *testing.T) {
	tests := []struct {
		name      string
		validate  func(string) error
		value     string
		wantError string
	}{
		{name: "run id accepts printable values", validate: ValidateRunID, value: "run-%1000s-中文"},
		{name: "run id rejects empty", validate: ValidateRunID, wantError: "cannot be empty"},
		{name: "run id rejects control character", validate: ValidateRunID, value: "run\nforged", wantError: "non-printable"},
		{name: "run id rejects invalid utf8", validate: ValidateRunID, value: string([]byte{0xff}), wantError: "valid UTF-8"},
		{name: "run id rejects excessive length", validate: ValidateRunID, value: strings.Repeat("a", MaxRunIDLength+1), wantError: "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validate(tt.value)
			if tt.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantError)
		})
	}
}

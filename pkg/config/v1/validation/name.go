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
	"unicode"
	"unicode/utf8"
)

const (
	// MaxRunIDLength is the maximum number of bytes accepted for a control run ID.
	MaxRunIDLength = 64
)

func validateIdentifier(value, kind string, maxLength int) error {
	if value == "" {
		return fmt.Errorf("%s cannot be empty", kind)
	}
	if len(value) > maxLength {
		return fmt.Errorf("%s is too long: length %d exceeds maximum %d", kind, len(value), maxLength)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", kind)
	}
	for _, r := range value {
		if !unicode.IsPrint(r) {
			return fmt.Errorf("%s contains non-printable character", kind)
		}
	}
	return nil
}

func ValidateRunID(runID string) error {
	return validateIdentifier(runID, "run id", MaxRunIDLength)
}

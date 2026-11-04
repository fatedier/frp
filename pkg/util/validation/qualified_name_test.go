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

func TestIsQualifiedNameValidity(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "single letter", value: "a", valid: true},
		{name: "single digit", value: "0", valid: true},
		{name: "uppercase name", value: "MyName", valid: true},
		{name: "allowed punctuation", value: "a-._b", valid: true},
		{name: "name at max length", value: strings.Repeat("a", 63), valid: true},
		{name: "name over max length", value: strings.Repeat("a", 64), valid: false},
		{name: "prefix and name", value: "example.com/MyName", valid: true},
		{name: "prefix at max length", value: strings.Repeat("a", 253) + "/key", valid: true},
		{name: "prefix over max length", value: strings.Repeat("a", 254) + "/key", valid: false},
		{name: "name at max length with prefix", value: "example.com/" + strings.Repeat("a", 63), valid: true},
		{name: "name over max length with prefix", value: "example.com/" + strings.Repeat("a", 64), valid: false},
		{name: "empty", value: "", valid: false},
		{name: "leading hyphen", value: "-a", valid: false},
		{name: "trailing hyphen", value: "a-", valid: false},
		{name: "leading underscore", value: "_a", valid: false},
		{name: "trailing dot", value: "a.", valid: false},
		{name: "only dot", value: ".", valid: false},
		{name: "double dot in name", value: "a..b", valid: true},
		{name: "space", value: "a b", valid: false},
		{name: "tab", value: "a\tb", valid: false},
		{name: "newline", value: "a\nb", valid: false},
		{name: "nul byte", value: "a\x00b", valid: false},
		{name: "non ascii name", value: "é", valid: false},
		{name: "non ascii prefix", value: "é/key", valid: false},
		{name: "empty prefix", value: "/key", valid: false},
		{name: "empty name", value: "example.com/", valid: false},
		{name: "two slashes", value: "a//b", valid: false},
		{name: "three segments", value: "a/b/c", valid: false},
		{name: "uppercase prefix", value: "Example.com/key", valid: false},
		{name: "prefix with empty label", value: "example..com/key", valid: false},
		{name: "prefix with leading hyphen", value: "-example.com/key", valid: false},
		{name: "prefix with trailing dot", value: "example.com./key", valid: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := IsQualifiedName(tc.value)
			if tc.valid {
				require.Empty(t, errs)
			} else {
				require.NotEmpty(t, errs)
			}
		})
	}
}

func TestIsQualifiedNameMessages(t *testing.T) {
	t.Run("name too long", func(t *testing.T) {
		errs := IsQualifiedName(strings.Repeat("a", 64))
		require.Contains(t, errs, "name part must be no more than 63 characters")
	})
	t.Run("prefix too long", func(t *testing.T) {
		errs := IsQualifiedName(strings.Repeat("a", 254) + "/key")
		require.Contains(t, errs, "prefix part must be no more than 253 characters")
	})
	t.Run("too many segments", func(t *testing.T) {
		errs := IsQualifiedName("a/b/c")
		require.Len(t, errs, 1)
		require.Contains(t, errs[0], "a qualified name must consist of alphanumeric characters")
		require.Contains(t, errs[0], "with an optional DNS subdomain prefix and '/'")
	})
	t.Run("invalid character", func(t *testing.T) {
		errs := IsQualifiedName("a b")
		require.Contains(t, errs[0], "name part must consist of alphanumeric characters")
	})
	t.Run("empty prefix", func(t *testing.T) {
		errs := IsQualifiedName("/key")
		require.Equal(t, []string{"prefix part must be non-empty"}, errs)
	})
	t.Run("empty name", func(t *testing.T) {
		errs := IsQualifiedName("example.com/")
		require.Contains(t, errs[0], "name part must be non-empty")
	})
}

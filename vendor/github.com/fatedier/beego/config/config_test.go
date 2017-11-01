// Copyright 2016 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"os"
	"testing"
)

func TestExpandValueEnv(t *testing.T) {

	testCases := []struct {
		item string
		want string
	}{
		{"", ""},
		{"$", "$"},
		{"{", "{"},
		{"{}", "{}"},
		{"${}", ""},
		{"${|}", ""},
		{"${}", ""},
		{"${{}}", ""},
		{"${{||}}", "}"},
		{"${pwd||}", ""},
		{"${pwd||}", ""},
		{"${pwd||}", ""},
		{"${pwd||}}", "}"},
		{"${pwd||{{||}}}", "{{||}}"},
		{"${GOPATH}", os.Getenv("GOPATH")},
		{"${GOPATH||}", os.Getenv("GOPATH")},
		{"${GOPATH||root}", os.Getenv("GOPATH")},
		{"${GOPATH_NOT||root}", "root"},
		{"${GOPATH_NOT||||root}", "||root"},
	}

	for _, c := range testCases {
		if got := ExpandValueEnv(c.item); got != c.want {
			t.Errorf("expand value error, item %q want %q, got %q", c.item, c.want, got)
		}
	}

}

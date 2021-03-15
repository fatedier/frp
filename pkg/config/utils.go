// Copyright 2020 The frp Authors
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

package config

import (
	"strings"
)

func GetMapWithoutPrefix(set map[string]string, prefix string) map[string]string {
	m := make(map[string]string)

	for key, value := range set {
		if strings.HasPrefix(key, prefix) {
			m[strings.TrimPrefix(key, prefix)] = value
		}
	}

	if len(m) == 0 {
		return nil
	}

	return m
}

func GetMapByPrefix(set map[string]string, prefix string) map[string]string {
	m := make(map[string]string)

	for key, value := range set {
		if strings.HasPrefix(key, prefix) {
			m[key] = value
		}
	}

	if len(m) == 0 {
		return nil
	}

	return m
}

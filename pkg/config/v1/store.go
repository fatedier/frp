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

package v1

// StoreConfig configures the built-in store source.
type StoreConfig struct {
	// Path is the store file path.
	Path string `json:"path,omitempty"`
}

// IsEnabled returns true if the store is configured with a valid path.
func (c *StoreConfig) IsEnabled() bool {
	return c.Path != ""
}

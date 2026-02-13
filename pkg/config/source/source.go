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

package source

import (
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// Source is the interface for configuration sources.
// A Source provides proxy and visitor configurations from various backends.
// Aggregator currently uses the built-in config source as base and an optional
// store source as higher-priority overlay.
type Source interface {
	// Load loads the proxy and visitor configurations from this source.
	// Returns the loaded configurations and any error encountered.
	// A disabled entry in one source is source-local filtering, not a cross-source
	// tombstone for entries from lower-priority sources.
	//
	// Error handling contract with Aggregator:
	//   - When err is nil, returned slices are consumed.
	//   - When err is non-nil, Aggregator aborts the merge and returns the error.
	//   - To publish best-effort or partial results, return those results with
	//     err set to nil.
	Load() (proxies []v1.ProxyConfigurer, visitors []v1.VisitorConfigurer, err error)
}

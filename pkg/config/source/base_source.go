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
	"sync"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// baseSource provides shared state and behavior for Source implementations.
// It manages proxy/visitor storage.
// Concrete types (ConfigSource, StoreSource) embed this struct.
type baseSource struct {
	mu sync.RWMutex

	proxies  map[string]v1.ProxyConfigurer
	visitors map[string]v1.VisitorConfigurer
}

func newBaseSource() baseSource {
	return baseSource{
		proxies:  make(map[string]v1.ProxyConfigurer),
		visitors: make(map[string]v1.VisitorConfigurer),
	}
}

// Load returns all enabled proxy and visitor configurations.
// Configurations with Enabled explicitly set to false are filtered out.
func (s *baseSource) Load() ([]v1.ProxyConfigurer, []v1.VisitorConfigurer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	proxies := make([]v1.ProxyConfigurer, 0, len(s.proxies))
	for _, p := range s.proxies {
		// Filter out disabled proxies (nil or true means enabled)
		if enabled := p.GetBaseConfig().Enabled; enabled != nil && !*enabled {
			continue
		}
		proxies = append(proxies, p)
	}

	visitors := make([]v1.VisitorConfigurer, 0, len(s.visitors))
	for _, v := range s.visitors {
		// Filter out disabled visitors (nil or true means enabled)
		if enabled := v.GetBaseConfig().Enabled; enabled != nil && !*enabled {
			continue
		}
		visitors = append(visitors, v)
	}

	return proxies, visitors, nil
}

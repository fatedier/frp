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
	"fmt"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// ConfigSource implements Source for in-memory configuration.
// All operations are thread-safe.
type ConfigSource struct {
	baseSource
}

func NewConfigSource() *ConfigSource {
	return &ConfigSource{
		baseSource: newBaseSource(),
	}
}

// ReplaceAll replaces all proxy and visitor configurations atomically.
func (s *ConfigSource) ReplaceAll(proxies []v1.ProxyConfigurer, visitors []v1.VisitorConfigurer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	nextProxies := make(map[string]v1.ProxyConfigurer, len(proxies))
	for _, p := range proxies {
		if p == nil {
			return fmt.Errorf("proxy cannot be nil")
		}
		name := p.GetBaseConfig().Name
		if name == "" {
			return fmt.Errorf("proxy name cannot be empty")
		}
		nextProxies[name] = p
	}
	nextVisitors := make(map[string]v1.VisitorConfigurer, len(visitors))
	for _, v := range visitors {
		if v == nil {
			return fmt.Errorf("visitor cannot be nil")
		}
		name := v.GetBaseConfig().Name
		if name == "" {
			return fmt.Errorf("visitor name cannot be empty")
		}
		nextVisitors[name] = v
	}
	s.proxies = nextProxies
	s.visitors = nextVisitors
	return nil
}

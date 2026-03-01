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

func cloneConfigurers(
	proxies []v1.ProxyConfigurer,
	visitors []v1.VisitorConfigurer,
) ([]v1.ProxyConfigurer, []v1.VisitorConfigurer, error) {
	clonedProxies := make([]v1.ProxyConfigurer, 0, len(proxies))
	clonedVisitors := make([]v1.VisitorConfigurer, 0, len(visitors))

	for _, cfg := range proxies {
		if cfg == nil {
			return nil, nil, fmt.Errorf("proxy cannot be nil")
		}
		clonedProxies = append(clonedProxies, cfg.Clone())
	}
	for _, cfg := range visitors {
		if cfg == nil {
			return nil, nil, fmt.Errorf("visitor cannot be nil")
		}
		clonedVisitors = append(clonedVisitors, cfg.Clone())
	}
	return clonedProxies, clonedVisitors, nil
}

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

func validateProxyName(proxy v1.ProxyConfigurer) (string, error) {
	if proxy == nil {
		return "", fmt.Errorf("proxy cannot be nil")
	}
	name := proxy.GetBaseConfig().Name
	if name == "" {
		return "", fmt.Errorf("proxy name cannot be empty")
	}
	return name, nil
}

func validateVisitorName(visitor v1.VisitorConfigurer) (string, error) {
	if visitor == nil {
		return "", fmt.Errorf("visitor cannot be nil")
	}
	name := visitor.GetBaseConfig().Name
	if name == "" {
		return "", fmt.Errorf("visitor name cannot be empty")
	}
	return name, nil
}

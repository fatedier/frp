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

package http

import (
	"encoding/json"
	"testing"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestGetConfFromConfigurerKeepsPluginFields(t *testing.T) {
	cfg := &v1.TCPProxyConfig{
		ProxyBaseConfig: v1.ProxyBaseConfig{
			Name: "test-proxy",
			Type: string(v1.ProxyTypeTCP),
			ProxyBackend: v1.ProxyBackend{
				Plugin: v1.TypedClientPluginOptions{
					Type: v1.PluginHTTPProxy,
					ClientPluginOptions: &v1.HTTPProxyPluginOptions{
						Type:         v1.PluginHTTPProxy,
						HTTPUser:     "user",
						HTTPPassword: "password",
					},
				},
			},
		},
		RemotePort: 6000,
	}

	content, err := json.Marshal(getConfFromConfigurer(cfg))
	if err != nil {
		t.Fatalf("marshal conf failed: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(content, &out); err != nil {
		t.Fatalf("unmarshal conf failed: %v", err)
	}

	pluginValue, ok := out["plugin"]
	if !ok {
		t.Fatalf("plugin field missing in output: %v", out)
	}
	plugin, ok := pluginValue.(map[string]any)
	if !ok {
		t.Fatalf("plugin field should be object, got: %#v", pluginValue)
	}

	if got := plugin["type"]; got != v1.PluginHTTPProxy {
		t.Fatalf("plugin type mismatch, want %q got %#v", v1.PluginHTTPProxy, got)
	}
	if got := plugin["httpUser"]; got != "user" {
		t.Fatalf("plugin httpUser mismatch, want %q got %#v", "user", got)
	}
	if got := plugin["httpPassword"]; got != "password" {
		t.Fatalf("plugin httpPassword mismatch, want %q got %#v", "password", got)
	}
}

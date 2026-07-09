// Copyright 2025 The frp Authors
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

import (
	"strings"
	"testing"
)

// tcp+udp and stcp+sudp are REAL merged proxy types (not sugar): each entry
// decodes to a single proxy of that type, keeping its own name. http+https was
// removed entirely, so it must now be rejected as an unknown type.
func TestMergedProxyTypesDecodeAsSingleRealProxies(t *testing.T) {
	jsonStr := `{
		"proxies": [
			{"name":"game","type":"tcp+udp","localPort":8000,"remotePort":9000,"localPortUDP":8001},
			{"name":"secret","type":"stcp+sudp","secretKey":"k","localPort":22},
			{"name":"plain","type":"tcp","localPort":22,"remotePort":6000}
		]
	}`
	cfg, err := DecodeClientConfigJSON([]byte(jsonStr), DecodeOptions{})
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	got := map[string]string{}
	for _, p := range cfg.Proxies {
		got[p.GetBaseConfig().Name] = p.GetBaseConfig().Type
	}
	want := map[string]string{
		"game":   "tcp+udp",
		"secret": "stcp+sudp",
		"plain":  "tcp",
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d proxies, got %d: %v", len(want), len(got), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("proxy %s: want type %s, got %q", k, v, got[k])
		}
	}

	// The merged tcp+udp proxy keeps its fields on a single real configurer.
	for _, p := range cfg.Proxies {
		if p.GetBaseConfig().Name == "game" {
			u, ok := p.ProxyConfigurer.(*TCPUDPProxyConfig)
			if !ok || u.LocalPort != 8000 || u.RemotePort != 9000 || u.LocalPortUDP != 8001 {
				t.Errorf("game did not decode as a TCPUDPProxyConfig with expected fields: %+v", p.ProxyConfigurer)
			}
		}
	}
}

func TestHTTPHTTPSComboTypeRemoved(t *testing.T) {
	jsonStr := `{"proxies":[{"name":"web","type":"http+https","customDomains":["a.com"]}]}`
	if _, err := DecodeClientConfigJSON([]byte(jsonStr), DecodeOptions{}); err == nil {
		t.Fatal("expected http+https to be rejected as unknown type, got nil error")
	} else if !strings.Contains(err.Error(), "unknown proxy type") {
		t.Errorf("expected unknown proxy type error, got: %v", err)
	}
}

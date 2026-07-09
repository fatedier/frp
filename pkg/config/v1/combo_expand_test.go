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

import "testing"

func TestExpandComboProxy(t *testing.T) {
	jsonStr := `{
		"proxies": [
			{"name":"web","type":"http+https","customDomains":["a.com"]},
			{"name":"game","type":"tcp+udp","localPort":8000,"remotePort":9000},
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
		"web-http": "http", "web-https": "https",
		"game-tcp": "tcp", "game-udp": "udp",
		"secret-stcp": "stcp", "secret-sudp": "sudp",
		"plain": "tcp",
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d proxies, got %d: %v", len(want), len(got), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("proxy %s: want type %s, got %q", k, v, got[k])
		}
	}

	// shared fields must survive the split
	for _, p := range cfg.Proxies {
		if p.GetBaseConfig().Name == "game-udp" {
			if u, ok := p.ProxyConfigurer.(*UDPProxyConfig); !ok || u.LocalPort != 8000 || u.RemotePort != 9000 {
				t.Errorf("game-udp did not inherit shared fields: %+v", p.ProxyConfigurer)
			}
		}
	}
}

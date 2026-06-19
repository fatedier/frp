// Copyright 2024 The frp Authors
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

package client

import "testing"

func TestProxyURLForServer(t *testing.T) {
	const proxy = "http://proxy.example.com:8080"

	tests := []struct {
		name     string
		proxyURL string
		server   string
		noProxy  string
		want     string
	}{
		{name: "no proxy configured", proxyURL: "", server: "1.2.3.4:7000", noProxy: "example.com", want: ""},
		{name: "proxied by default", proxyURL: proxy, server: "1.2.3.4:7000", noProxy: "", want: proxy},
		{name: "loopback always bypassed", proxyURL: proxy, server: "127.0.0.1:7000", noProxy: "", want: ""},
		{name: "localhost always bypassed", proxyURL: proxy, server: "localhost:7000", noProxy: "", want: ""},
		{name: "ip in no_proxy", proxyURL: proxy, server: "10.0.0.5:7000", noProxy: "10.0.0.5", want: ""},
		{name: "ip not in no_proxy", proxyURL: proxy, server: "10.0.0.5:7000", noProxy: "10.0.0.6", want: proxy},
		{name: "cidr in no_proxy", proxyURL: proxy, server: "10.0.0.5:7000", noProxy: "10.0.0.0/24", want: ""},
		{name: "domain in no_proxy", proxyURL: proxy, server: "frps.example.com:7000", noProxy: "example.com", want: ""},
		{name: "domain not in no_proxy", proxyURL: proxy, server: "frps.example.com:7000", noProxy: "other.com", want: proxy},
		{name: "wildcard no_proxy", proxyURL: proxy, server: "frps.example.com:7000", noProxy: "*", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := proxyURLForServer(tt.proxyURL, tt.server, tt.noProxy); got != tt.want {
				t.Fatalf("proxyURLForServer(%q, %q, %q) = %q, want %q",
					tt.proxyURL, tt.server, tt.noProxy, got, tt.want)
			}
		})
	}
}

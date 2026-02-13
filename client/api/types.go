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

package api

// StatusResp is the response for GET /api/status
type StatusResp map[string][]ProxyStatusResp

// ProxyStatusResp contains proxy status information
type ProxyStatusResp struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Err        string `json:"err"`
	LocalAddr  string `json:"local_addr"`
	Plugin     string `json:"plugin"`
	RemoteAddr string `json:"remote_addr"`
	Source     string `json:"source,omitempty"` // "store" or "config"
}

// ProxyConfig wraps proxy configuration for API requests/responses.
type ProxyConfig struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

// VisitorConfig wraps visitor configuration for API requests/responses.
type VisitorConfig struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

// ProxyListResp is the response for GET /api/store/proxies
type ProxyListResp struct {
	Proxies []ProxyConfig `json:"proxies"`
}

// VisitorListResp is the response for GET /api/store/visitors
type VisitorListResp struct {
	Visitors []VisitorConfig `json:"visitors"`
}

// ErrorResp represents an error response
type ErrorResp struct {
	Error string `json:"error"`
}

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

package v1

import (
	"github.com/fatedier/frp/pkg/config/types"
)

// EtcdConfig contains configuration for connecting to etcd for multi-tenant token management.
type EtcdConfig struct {
	// Endpoints is the list of etcd server addresses.
	Endpoints []string `json:"endpoints,omitempty"`
	// Region is the region identifier for this frps instance.
	// Only tokens with matching region will be accepted.
	Region string `json:"region,omitempty"`
	// Prefix is the key prefix for storing token configurations in etcd.
	// Default is "/frp/tokens/".
	Prefix string `json:"prefix,omitempty"`
	// DialTimeout is the timeout for establishing connection to etcd in seconds.
	// Default is 5 seconds.
	DialTimeout int64 `json:"dialTimeout,omitempty"`
	// Username for etcd authentication (optional).
	Username string `json:"username,omitempty"`
	// Password for etcd authentication (optional).
	Password string `json:"password,omitempty"`
	// TLS configuration for etcd connection (optional).
	TLS TLSConfig `json:"tls,omitempty"`
	// TrafficReportURL is the URL to report traffic usage.
	// POST request with JSON body will be sent to this URL.
	TrafficReportURL string `json:"trafficReportUrl,omitempty"`
}

// IsEnabled returns true if etcd configuration is set.
func (c *EtcdConfig) IsEnabled() bool {
	return len(c.Endpoints) > 0 && c.Region != ""
}

// TokenConfig represents the configuration for a single token stored in etcd.
// Key format in etcd: {prefix}{token}
// Example: /frp/tokens/abc123
type TokenConfig struct {
	// Token is the authentication token.
	Token string `json:"token"`
	// Region is the region this token is allowed to connect to.
	Region string `json:"region"`
	// AllowPorts specifies the ports this token can use.
	// Format: "8080,9000-9100"
	AllowPorts []types.PortsRange `json:"allowPorts,omitempty"`
	// BandwidthLimit specifies the bandwidth limit for this token.
	// Format: "10MB" or "1024KB"
	BandwidthLimit types.BandwidthQuantity `json:"bandwidthLimit,omitempty"`
	// MaxPortsPerClient specifies the maximum number of ports this token can use.
	// 0 means no limit.
	MaxPortsPerClient int64 `json:"maxPortsPerClient,omitempty"`
	// Enabled indicates whether this token is active.
	Enabled bool `json:"enabled"`
	// Description is an optional description for this token.
	Description string `json:"description,omitempty"`
	// TrafficReportInterval specifies how often to report traffic in MB.
	// For example, 50 means report every 50MB of traffic.
	// 0 means no traffic reporting for this token.
	TrafficReportIntervalMB int64 `json:"trafficReportIntervalMB,omitempty"`
}

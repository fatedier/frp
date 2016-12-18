// Copyright 2016 fatedier, fatedier@gmail.com
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

package config

type BaseConf struct {
	Name              string `json:"name"`
	AuthToken         string `json:"-"`
	Type              string `json:"type"`
	UseEncryption     bool   `json:"use_encryption"`
	UseGzip           bool   `json:"use_gzip"`
	PrivilegeMode     bool   `json:"privilege_mode"`
	PrivilegeToken    string `json:"-"`
	PoolCount         int64  `json:"pool_count"`
	HostHeaderRewrite string `json:"host_header_rewrite"`
}

type ProxyServerConf struct {
	BaseConf
	BindAddr      string   `json:"bind_addr"`
	ListenPort    int64    `json:"bind_port"`
	CustomDomains []string `json:"custom_domains"`
	Locations     []string `json:"custom_locations"`

	Status int64 `json:"status"`
}

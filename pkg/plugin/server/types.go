// Copyright 2019 fatedier, fatedier@gmail.com
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

package plugin

import (
	"github.com/fatedier/frp/pkg/msg"
)

type Request struct {
	Version string `json:"version"`
	Op      string `json:"op"`
	Content any    `json:"content"`
}

type Response struct {
	Reject       bool   `json:"reject"`
	RejectReason string `json:"reject_reason"`
	Unchange     bool   `json:"unchange"`
	Content      any    `json:"content"`
}

type LoginContent struct {
	msg.Login

	ClientAddress string `json:"client_address,omitempty"`
}

type UserInfo struct {
	User  string            `json:"user"`
	Metas map[string]string `json:"metas"`
	RunID string            `json:"run_id"`
}

type NewProxyContent struct {
	User UserInfo `json:"user"`
	msg.NewProxy
}

type CloseProxyContent struct {
	User UserInfo `json:"user"`
	msg.CloseProxy
}

type PingContent struct {
	User UserInfo `json:"user"`
	msg.Ping
}

type NewWorkConnContent struct {
	User UserInfo `json:"user"`
	msg.NewWorkConn
}

type NewUserConnContent struct {
	User       UserInfo `json:"user"`
	ProxyName  string   `json:"proxy_name"`
	ProxyType  string   `json:"proxy_type"`
	RemoteAddr string   `json:"remote_addr"`
}

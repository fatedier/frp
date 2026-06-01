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

package model

type V2PageResp[T any] struct {
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Items    []T `json:"items"`
}

type V2UserResp struct {
	User        string `json:"user"`
	ClientCount int    `json:"clientCount"`
	ProxyCount  int    `json:"proxyCount"`
}

type V2ProxyResp struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	User     string            `json:"user"`
	ClientID string            `json:"clientID"`
	Spec     any               `json:"spec"`
	Status   V2ProxyStatusResp `json:"status"`
}

type V2ProxyStatusResp struct {
	State           string `json:"phase"`
	TodayTrafficIn  int64  `json:"todayTrafficIn"`
	TodayTrafficOut int64  `json:"todayTrafficOut"`
	CurConns        int64  `json:"curConns"`
	LastStartTime   string `json:"lastStartTime"`
	LastCloseTime   string `json:"lastCloseTime"`
}

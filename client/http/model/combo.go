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

package model

import (
	"github.com/fatedier/frp/pkg/util/jsonx"
)

// comboProxySubTypes maps an admin-API "combo" proxy type to the concrete proxy
// types it expands into. This mirrors the config-file sugar in pkg/config/v1 but
// operates on the admin API's ProxyDefinition JSON shape (typed blocks).
//
// Note: "xtcp+xudp" is intentionally NOT here — it is a real single-hole proxy
// type, handled like any other type.
var comboProxySubTypes = map[string][]string{
	"tcp+udp":    {"tcp", "udp"},
	"http+https": {"http", "https"},
	"stcp+sudp":  {"stcp", "sudp"},
}

// IsComboProxyType reports whether typ is a combo sugar type that a single admin
// API create request expands into multiple concrete proxies.
func IsComboProxyType(typ string) bool {
	_, ok := comboProxySubTypes[typ]
	return ok
}

// ExpandComboProxyBody turns a combo ProxyDefinition request body into one body
// per concrete sub-proxy: the name is suffixed with "-<subtype>", the type is set
// to the sub-type, and the combo's field block (keyed by the combo type) is moved
// to the sub-type key. Non-combo bodies return (nil, false, nil) unchanged.
func ExpandComboProxyBody(body []byte) ([][]byte, bool, error) {
	var env struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := jsonx.Unmarshal(body, &env); err != nil {
		return nil, false, err
	}
	subs, ok := comboProxySubTypes[env.Type]
	if !ok {
		return nil, false, nil
	}

	var fields map[string]jsonx.RawMessage
	if err := jsonx.Unmarshal(body, &fields); err != nil {
		return nil, false, err
	}
	comboBlock := fields[env.Type]

	out := make([][]byte, 0, len(subs))
	for _, st := range subs {
		nameRaw, err := jsonx.Marshal(env.Name + "-" + st)
		if err != nil {
			return nil, false, err
		}
		typeRaw, err := jsonx.Marshal(st)
		if err != nil {
			return nil, false, err
		}
		m := map[string]jsonx.RawMessage{
			"name": nameRaw,
			"type": typeRaw,
		}
		if len(comboBlock) > 0 {
			m[st] = comboBlock
		}
		b, err := jsonx.Marshal(m)
		if err != nil {
			return nil, false, err
		}
		out = append(out, b)
	}
	return out, true, nil
}

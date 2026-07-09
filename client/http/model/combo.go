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
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/jsonx"
)

// Combo definitions live in one place — v1.ComboProxySubTypes — shared with the
// config-file loader. This file only adapts them to the admin API's
// ProxyDefinition JSON shape (typed blocks) instead of flat config fields.
// Note: "xtcp+xudp" is NOT a combo — it is a real single-hole proxy type.

// IsComboProxyType reports whether typ is a combo sugar type that a single admin
// API create request expands into multiple concrete proxies.
func IsComboProxyType(typ string) bool {
	_, ok := v1.ComboProxySubTypes(typ)
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
	subs, ok := v1.ComboProxySubTypes(env.Type)
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

// IsComboVisitorType reports whether typ is a combo visitor type (only stcp+sudp).
func IsComboVisitorType(typ string) bool {
	_, ok := v1.ComboVisitorSubTypes(typ)
	return ok
}

// ExpandComboVisitorBody turns a combo visitor request body into one body per
// concrete sub-visitor. Unlike proxies, a visitor's serverName lives inside the
// typed block, so we suffix both the top-level name and the block's serverName
// with "-<subtype>" so they line up with the expanded proxy combo names.
func ExpandComboVisitorBody(body []byte) ([][]byte, bool, error) {
	var env struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := jsonx.Unmarshal(body, &env); err != nil {
		return nil, false, err
	}
	subs, ok := v1.ComboVisitorSubTypes(env.Type)
	if !ok {
		return nil, false, nil
	}

	var fields map[string]jsonx.RawMessage
	if err := jsonx.Unmarshal(body, &fields); err != nil {
		return nil, false, err
	}
	block := map[string]jsonx.RawMessage{}
	if raw, ok := fields[env.Type]; ok && len(raw) > 0 {
		if err := jsonx.Unmarshal(raw, &block); err != nil {
			return nil, false, err
		}
	}
	var serverName string
	if snRaw, ok := block["serverName"]; ok {
		_ = jsonx.Unmarshal(snRaw, &serverName)
	}

	out := make([][]byte, 0, len(subs))
	for _, st := range subs {
		subBlock := make(map[string]jsonx.RawMessage, len(block))
		for k, v := range block {
			subBlock[k] = v
		}
		if serverName != "" {
			snRaw, err := jsonx.Marshal(serverName + "-" + st)
			if err != nil {
				return nil, false, err
			}
			subBlock["serverName"] = snRaw
		}
		subBlockRaw, err := jsonx.Marshal(subBlock)
		if err != nil {
			return nil, false, err
		}
		nameRaw, err := jsonx.Marshal(env.Name + "-" + st)
		if err != nil {
			return nil, false, err
		}
		typeRaw, err := jsonx.Marshal(st)
		if err != nil {
			return nil, false, err
		}
		mb, err := jsonx.Marshal(map[string]jsonx.RawMessage{
			"name": nameRaw,
			"type": typeRaw,
			st:     subBlockRaw,
		})
		if err != nil {
			return nil, false, err
		}
		out = append(out, mb)
	}
	return out, true, nil
}

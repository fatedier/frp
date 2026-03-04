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

package v1

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/fatedier/frp/pkg/util/jsonx"
)

type DecodeOptions struct {
	DisallowUnknownFields bool
}

func decodeJSONWithOptions(b []byte, out any, options DecodeOptions) error {
	return jsonx.UnmarshalWithOptions(b, out, jsonx.DecodeOptions{
		RejectUnknownMembers: options.DisallowUnknownFields,
	})
}

func isJSONNull(b []byte) bool {
	return len(b) == 0 || string(b) == "null"
}

type typedEnvelope struct {
	Type   string           `json:"type"`
	Plugin jsonx.RawMessage `json:"plugin,omitempty"`
}

func DecodeProxyConfigurerJSON(b []byte, options DecodeOptions) (ProxyConfigurer, error) {
	if isJSONNull(b) {
		return nil, errors.New("type is required")
	}

	var env typedEnvelope
	if err := jsonx.Unmarshal(b, &env); err != nil {
		return nil, err
	}

	configurer := NewProxyConfigurerByType(ProxyType(env.Type))
	if configurer == nil {
		return nil, fmt.Errorf("unknown proxy type: %s", env.Type)
	}
	if err := decodeJSONWithOptions(b, configurer, options); err != nil {
		return nil, fmt.Errorf("unmarshal ProxyConfig error: %v", err)
	}

	if len(env.Plugin) > 0 && !isJSONNull(env.Plugin) {
		plugin, err := DecodeClientPluginOptionsJSON(env.Plugin, options)
		if err != nil {
			return nil, fmt.Errorf("unmarshal proxy plugin error: %v", err)
		}
		configurer.GetBaseConfig().Plugin = plugin
	}
	return configurer, nil
}

func DecodeVisitorConfigurerJSON(b []byte, options DecodeOptions) (VisitorConfigurer, error) {
	if isJSONNull(b) {
		return nil, errors.New("type is required")
	}

	var env typedEnvelope
	if err := jsonx.Unmarshal(b, &env); err != nil {
		return nil, err
	}

	configurer := NewVisitorConfigurerByType(VisitorType(env.Type))
	if configurer == nil {
		return nil, fmt.Errorf("unknown visitor type: %s", env.Type)
	}
	if err := decodeJSONWithOptions(b, configurer, options); err != nil {
		return nil, fmt.Errorf("unmarshal VisitorConfig error: %v", err)
	}

	if len(env.Plugin) > 0 && !isJSONNull(env.Plugin) {
		plugin, err := DecodeVisitorPluginOptionsJSON(env.Plugin, options)
		if err != nil {
			return nil, fmt.Errorf("unmarshal visitor plugin error: %v", err)
		}
		configurer.GetBaseConfig().Plugin = plugin
	}
	return configurer, nil
}

func DecodeClientPluginOptionsJSON(b []byte, options DecodeOptions) (TypedClientPluginOptions, error) {
	if isJSONNull(b) {
		return TypedClientPluginOptions{}, nil
	}

	var env typedEnvelope
	if err := jsonx.Unmarshal(b, &env); err != nil {
		return TypedClientPluginOptions{}, err
	}
	if env.Type == "" {
		return TypedClientPluginOptions{}, errors.New("plugin type is empty")
	}

	v, ok := clientPluginOptionsTypeMap[env.Type]
	if !ok {
		return TypedClientPluginOptions{}, fmt.Errorf("unknown plugin type: %s", env.Type)
	}
	optionsStruct := reflect.New(v).Interface().(ClientPluginOptions)
	if err := decodeJSONWithOptions(b, optionsStruct, options); err != nil {
		return TypedClientPluginOptions{}, fmt.Errorf("unmarshal ClientPluginOptions error: %v", err)
	}
	return TypedClientPluginOptions{
		Type:                env.Type,
		ClientPluginOptions: optionsStruct,
	}, nil
}

func DecodeVisitorPluginOptionsJSON(b []byte, options DecodeOptions) (TypedVisitorPluginOptions, error) {
	if isJSONNull(b) {
		return TypedVisitorPluginOptions{}, nil
	}

	var env typedEnvelope
	if err := jsonx.Unmarshal(b, &env); err != nil {
		return TypedVisitorPluginOptions{}, err
	}
	if env.Type == "" {
		return TypedVisitorPluginOptions{}, errors.New("visitor plugin type is empty")
	}

	v, ok := visitorPluginOptionsTypeMap[env.Type]
	if !ok {
		return TypedVisitorPluginOptions{}, fmt.Errorf("unknown visitor plugin type: %s", env.Type)
	}
	optionsStruct := reflect.New(v).Interface().(VisitorPluginOptions)
	if err := decodeJSONWithOptions(b, optionsStruct, options); err != nil {
		return TypedVisitorPluginOptions{}, fmt.Errorf("unmarshal VisitorPluginOptions error: %v", err)
	}
	return TypedVisitorPluginOptions{
		Type:                 env.Type,
		VisitorPluginOptions: optionsStruct,
	}, nil
}

func DecodeClientConfigJSON(b []byte, options DecodeOptions) (ClientConfig, error) {
	type rawClientConfig struct {
		ClientCommonConfig
		Proxies  []jsonx.RawMessage `json:"proxies,omitempty"`
		Visitors []jsonx.RawMessage `json:"visitors,omitempty"`
	}

	raw := rawClientConfig{}
	if err := decodeJSONWithOptions(b, &raw, options); err != nil {
		return ClientConfig{}, err
	}

	cfg := ClientConfig{
		ClientCommonConfig: raw.ClientCommonConfig,
		Proxies:            make([]TypedProxyConfig, 0, len(raw.Proxies)),
		Visitors:           make([]TypedVisitorConfig, 0, len(raw.Visitors)),
	}

	for i, proxyData := range raw.Proxies {
		proxyCfg, err := DecodeProxyConfigurerJSON(proxyData, options)
		if err != nil {
			return ClientConfig{}, fmt.Errorf("decode proxy at index %d: %w", i, err)
		}
		cfg.Proxies = append(cfg.Proxies, TypedProxyConfig{
			Type:            proxyCfg.GetBaseConfig().Type,
			ProxyConfigurer: proxyCfg,
		})
	}

	for i, visitorData := range raw.Visitors {
		visitorCfg, err := DecodeVisitorConfigurerJSON(visitorData, options)
		if err != nil {
			return ClientConfig{}, fmt.Errorf("decode visitor at index %d: %w", i, err)
		}
		cfg.Visitors = append(cfg.Visitors, TypedVisitorConfig{
			Type:              visitorCfg.GetBaseConfig().Type,
			VisitorConfigurer: visitorCfg,
		})
	}

	return cfg, nil
}

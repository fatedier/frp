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

package validation

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/policy/featuregate"
	"github.com/fatedier/frp/pkg/policy/security"
)

func validateClientFeatureGates(t *testing.T, gates map[string]bool, virtualNetAddress string) error {
	t.Helper()

	cfg := &v1.ClientCommonConfig{
		FeatureGates: gates,
		VirtualNet: v1.VirtualNetConfig{
			Address: virtualNetAddress,
		},
	}
	require.NoError(t, cfg.Complete())

	_, err := NewConfigValidator(security.NewUnsafeFeatures(nil)).ValidateClientCommonConfig(cfg)
	return err
}

func TestValidateClientFeatureGates(t *testing.T) {
	tests := []struct {
		name              string
		featureGates      map[string]bool
		virtualNetAddress string
		wantErr           string
	}{
		{
			name:              "VirtualNet enabled",
			featureGates:      map[string]bool{"VirtualNet": true},
			virtualNetAddress: "100.86.0.4/24",
		},
		{
			name:              "VirtualNet explicitly disabled",
			featureGates:      map[string]bool{"VirtualNet": false},
			virtualNetAddress: "100.86.0.4/24",
			wantErr:           "VirtualNet feature is not enabled",
		},
		{
			name:              "VirtualNet disabled by default",
			virtualNetAddress: "100.86.0.4/24",
			wantErr:           "VirtualNet feature is not enabled",
		},
		{
			name:         "unknown feature gate",
			featureGates: map[string]bool{"UnknownFeature": true},
			wantErr:      "unrecognized feature gate: UnknownFeature",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateClientFeatureGates(t, tc.featureGates, tc.virtualNetAddress)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestGetClientConfigRequirements(t *testing.T) {
	virtualNetProxy := &v1.STCPProxyConfig{
		ProxyBaseConfig: v1.ProxyBaseConfig{
			ProxyBackend: v1.ProxyBackend{
				Plugin: v1.TypedClientPluginOptions{Type: v1.PluginVirtualNet},
			},
		},
	}
	virtualNetVisitor := &v1.STCPVisitorConfig{
		VisitorBaseConfig: v1.VisitorBaseConfig{
			Plugin: v1.TypedVisitorPluginOptions{Type: v1.VisitorPluginVirtualNet},
		},
	}

	tests := []struct {
		name     string
		common   *v1.ClientCommonConfig
		proxies  []v1.ProxyConfigurer
		visitors []v1.VisitorConfigurer
		wantVNet bool
	}{
		{name: "no requirements"},
		{
			name:     "common VirtualNet address",
			common:   &v1.ClientCommonConfig{VirtualNet: v1.VirtualNetConfig{Address: "100.86.0.4/24"}},
			wantVNet: true,
		},
		{name: "VirtualNet proxy", proxies: []v1.ProxyConfigurer{virtualNetProxy}, wantVNet: true},
		{name: "VirtualNet visitor", visitors: []v1.VisitorConfigurer{virtualNetVisitor}, wantVNet: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetClientConfigRequirements(tc.common, tc.proxies, tc.visitors)
			require.Equal(t, tc.wantVNet, got.VirtualNet)
		})
	}
}

func TestValidateClientFeatureGatesAreConfigScoped(t *testing.T) {
	defaultGatesBefore := featuregate.DefaultFeatureGates.String()

	require.NoError(t, validateClientFeatureGates(
		t,
		map[string]bool{"VirtualNet": true},
		"100.86.0.4/24",
	))
	require.Equal(t, defaultGatesBefore, featuregate.DefaultFeatureGates.String())

	err := validateClientFeatureGates(
		t,
		map[string]bool{"VirtualNet": false},
		"100.86.0.4/24",
	)
	require.ErrorContains(t, err, "VirtualNet feature is not enabled")
	require.Equal(t, defaultGatesBefore, featuregate.DefaultFeatureGates.String())
}

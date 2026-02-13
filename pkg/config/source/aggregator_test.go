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

package source

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// mockProxy creates a TCP proxy config for testing
func mockProxy(name string) v1.ProxyConfigurer {
	cfg := &v1.TCPProxyConfig{}
	cfg.Name = name
	cfg.Type = "tcp"
	cfg.LocalPort = 8080
	cfg.RemotePort = 9090
	return cfg
}

// mockVisitor creates a STCP visitor config for testing
func mockVisitor(name string) v1.VisitorConfigurer {
	cfg := &v1.STCPVisitorConfig{}
	cfg.Name = name
	cfg.Type = "stcp"
	cfg.ServerName = "test-server"
	return cfg
}

func newTestStoreSource(t *testing.T) *StoreSource {
	t.Helper()

	path := filepath.Join(t.TempDir(), "store.json")
	storeSource, err := NewStoreSource(StoreSourceConfig{Path: path})
	require.NoError(t, err)
	return storeSource
}

func newTestAggregator(t *testing.T, storeSource *StoreSource) *Aggregator {
	t.Helper()

	configSource := NewConfigSource()
	agg := NewAggregator(configSource)
	if storeSource != nil {
		agg.SetStoreSource(storeSource)
	}
	return agg
}

func TestNewAggregator_CreatesConfigSourceWhenNil(t *testing.T) {
	require := require.New(t)

	agg := NewAggregator(nil)
	require.NotNil(agg)
	require.NotNil(agg.ConfigSource())
	require.Nil(agg.StoreSource())
}

func TestNewAggregator_WithoutStore(t *testing.T) {
	require := require.New(t)

	configSource := NewConfigSource()
	agg := NewAggregator(configSource)
	require.NotNil(agg)
	require.Same(configSource, agg.ConfigSource())
	require.Nil(agg.StoreSource())
}

func TestNewAggregator_WithStore(t *testing.T) {
	require := require.New(t)

	storeSource := newTestStoreSource(t)
	configSource := NewConfigSource()
	agg := NewAggregator(configSource)
	agg.SetStoreSource(storeSource)

	require.Same(configSource, agg.ConfigSource())
	require.Same(storeSource, agg.StoreSource())
}

func TestAggregator_SetStoreSource_Overwrite(t *testing.T) {
	require := require.New(t)

	agg := newTestAggregator(t, nil)
	first := newTestStoreSource(t)
	second := newTestStoreSource(t)

	agg.SetStoreSource(first)
	require.Same(first, agg.StoreSource())

	agg.SetStoreSource(second)
	require.Same(second, agg.StoreSource())

	agg.SetStoreSource(nil)
	require.Nil(agg.StoreSource())
}

func TestAggregator_MergeBySourceOrder(t *testing.T) {
	require := require.New(t)

	storeSource := newTestStoreSource(t)
	agg := newTestAggregator(t, storeSource)

	configSource := agg.ConfigSource()

	configShared := mockProxy("shared").(*v1.TCPProxyConfig)
	configShared.LocalPort = 1111
	configOnly := mockProxy("only-in-config").(*v1.TCPProxyConfig)
	configOnly.LocalPort = 1112

	err := configSource.ReplaceAll([]v1.ProxyConfigurer{configShared, configOnly}, nil)
	require.NoError(err)

	storeShared := mockProxy("shared").(*v1.TCPProxyConfig)
	storeShared.LocalPort = 2222
	storeOnly := mockProxy("only-in-store").(*v1.TCPProxyConfig)
	storeOnly.LocalPort = 2223
	err = storeSource.AddProxy(storeShared)
	require.NoError(err)
	err = storeSource.AddProxy(storeOnly)
	require.NoError(err)

	proxies, visitors, err := agg.Load()
	require.NoError(err)
	require.Len(visitors, 0)
	require.Len(proxies, 3)

	var sharedProxy *v1.TCPProxyConfig
	for _, p := range proxies {
		if p.GetBaseConfig().Name == "shared" {
			sharedProxy = p.(*v1.TCPProxyConfig)
			break
		}
	}
	require.NotNil(sharedProxy)
	require.Equal(2222, sharedProxy.LocalPort)
}

func TestAggregator_DisabledEntryIsSourceLocalFilter(t *testing.T) {
	require := require.New(t)

	storeSource := newTestStoreSource(t)
	agg := newTestAggregator(t, storeSource)
	configSource := agg.ConfigSource()

	lowProxy := mockProxy("shared-proxy").(*v1.TCPProxyConfig)
	lowProxy.LocalPort = 1111
	err := configSource.ReplaceAll([]v1.ProxyConfigurer{lowProxy}, nil)
	require.NoError(err)

	disabled := false
	highProxy := mockProxy("shared-proxy").(*v1.TCPProxyConfig)
	highProxy.LocalPort = 2222
	highProxy.Enabled = &disabled
	err = storeSource.AddProxy(highProxy)
	require.NoError(err)

	proxies, visitors, err := agg.Load()
	require.NoError(err)
	require.Len(proxies, 1)
	require.Len(visitors, 0)

	proxy := proxies[0].(*v1.TCPProxyConfig)
	require.Equal("shared-proxy", proxy.Name)
	require.Equal(1111, proxy.LocalPort)
}

func TestAggregator_VisitorMerge(t *testing.T) {
	require := require.New(t)

	storeSource := newTestStoreSource(t)
	agg := newTestAggregator(t, storeSource)

	err := agg.ConfigSource().ReplaceAll(nil, []v1.VisitorConfigurer{mockVisitor("visitor1")})
	require.NoError(err)
	err = storeSource.AddVisitor(mockVisitor("visitor2"))
	require.NoError(err)

	_, visitors, err := agg.Load()
	require.NoError(err)
	require.Len(visitors, 2)
}

func TestAggregator_Load_ReturnsSharedReferences(t *testing.T) {
	require := require.New(t)

	agg := newTestAggregator(t, nil)
	err := agg.ConfigSource().ReplaceAll([]v1.ProxyConfigurer{mockProxy("ssh")}, nil)
	require.NoError(err)

	proxies, _, err := agg.Load()
	require.NoError(err)
	require.Len(proxies, 1)
	require.Equal("ssh", proxies[0].GetBaseConfig().Name)

	proxies[0].GetBaseConfig().Name = "alice.ssh"

	proxies2, _, err := agg.Load()
	require.NoError(err)
	require.Len(proxies2, 1)
	require.Equal("alice.ssh", proxies2[0].GetBaseConfig().Name)
}

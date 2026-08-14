package client

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatedier/frp/client/configmgmt"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/policy/security"
	"github.com/fatedier/frp/pkg/vnet"
)

func newTestRawTCPProxyConfig(name string) *v1.TCPProxyConfig {
	return &v1.TCPProxyConfig{
		ProxyBaseConfig: v1.ProxyBaseConfig{
			Name: name,
			Type: "tcp",
			ProxyBackend: v1.ProxyBackend{
				LocalPort: 10080,
			},
		},
	}
}

func newTestVirtualNetProxyConfig(name string) *v1.STCPProxyConfig {
	return &v1.STCPProxyConfig{
		ProxyBaseConfig: v1.ProxyBaseConfig{
			Name: name,
			Type: "stcp",
			ProxyBackend: v1.ProxyBackend{
				Plugin: v1.TypedClientPluginOptions{
					Type:                v1.PluginVirtualNet,
					ClientPluginOptions: &v1.VirtualNetPluginOptions{Type: v1.PluginVirtualNet},
				},
			},
		},
	}
}

func newTestVirtualNetVisitorConfig(name string) *v1.STCPVisitorConfig {
	return &v1.STCPVisitorConfig{
		VisitorBaseConfig: v1.VisitorBaseConfig{
			Name:       name,
			Type:       "stcp",
			ServerName: "vnet-server",
			SecretKey:  "secret",
			BindPort:   -1,
			Plugin: v1.TypedVisitorPluginOptions{
				Type: v1.VisitorPluginVirtualNet,
				VisitorPluginOptions: &v1.VirtualNetVisitorPluginOptions{
					Type:          v1.VisitorPluginVirtualNet,
					DestinationIP: "100.86.0.1",
				},
			},
		},
	}
}

func TestServiceConfigManagerReloadVirtualNetRuntimeDependency(t *testing.T) {
	const runtimeErr = "VirtualNet-dependent configuration requires a VirtualNet runtime enabled at startup"

	tests := []struct {
		name                  string
		startupVirtualNetAddr string
		nextConfig            string
		wantRuntimeDependency bool
	}{
		{
			name:       "unrelated common config",
			nextConfig: `serverAddr = "0.0.0.0"`,
		},
		{
			name: "VirtualNet address without startup runtime",
			nextConfig: `featureGates = { VirtualNet = true }
virtualNet.address = "100.86.0.4/24"
`,
			wantRuntimeDependency: true,
		},
		{
			name: "VirtualNet proxy without startup runtime",
			nextConfig: `[[proxies]]
name = "vnet-proxy"
type = "stcp"
secretKey = "secret"
[proxies.plugin]
type = "virtual_net"
`,
			wantRuntimeDependency: true,
		},
		{
			name: "VirtualNet visitor without startup runtime",
			nextConfig: `[[visitors]]
name = "vnet-visitor"
type = "stcp"
serverName = "vnet-server"
secretKey = "secret"
bindPort = -1
[visitors.plugin]
type = "virtual_net"
destinationIP = "100.86.0.1"
`,
			wantRuntimeDependency: true,
		},
		{
			name:                  "existing VirtualNet startup runtime",
			startupVirtualNetAddr: "100.86.0.4/24",
			nextConfig: `featureGates = { VirtualNet = true }
virtualNet.address = "100.86.0.5/24"
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			current := &v1.ClientCommonConfig{}
			if tc.startupVirtualNetAddr != "" {
				current.FeatureGates = map[string]bool{"VirtualNet": true}
				current.VirtualNet.Address = tc.startupVirtualNetAddr
			}
			if err := current.Complete(); err != nil {
				t.Fatalf("complete current config: %v", err)
			}

			configFile := filepath.Join(t.TempDir(), "frpc.toml")
			if err := os.WriteFile(configFile, []byte(tc.nextConfig), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}

			configSource := source.NewConfigSource()
			aggregator := source.NewAggregator(configSource)
			svr := &Service{
				common:         current,
				reloadCommon:   current,
				configFilePath: configFile,
				unsafeFeatures: security.NewUnsafeFeatures(nil),
				aggregator:     aggregator,
				configSource:   configSource,
			}
			if tc.startupVirtualNetAddr != "" {
				svr.vnetController = vnet.NewController(current.VirtualNet)
			}

			err := (&serviceConfigManager{svr: svr}).ReloadFromFile(true)
			if tc.wantRuntimeDependency {
				if !errors.Is(err, configmgmt.ErrApplyConfig) || !strings.Contains(err.Error(), runtimeErr) {
					t.Fatalf("expected VirtualNet runtime dependency error, got: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("reload config: %v", err)
			}
			if svr.common != current {
				t.Fatal("reload should not replace startup common config")
			}
			if tc.startupVirtualNetAddr == "" && svr.vnetController != nil {
				t.Fatal("reload should not enable startup-only VirtualNet runtime state")
			}
		})
	}
}

func TestServiceConfigManagerReloadVirtualNetRuntimeDependencyUsesMergedSources(t *testing.T) {
	const runtimeErr = "VirtualNet-dependent configuration requires a VirtualNet runtime enabled at startup"

	tests := []struct {
		name                  string
		nextConfig            string
		storeProxy            v1.ProxyConfigurer
		storeVisitor          v1.VisitorConfigurer
		wantRuntimeDependency bool
		wantProxyPlugin       string
	}{
		{
			name:                  "Store VirtualNet proxy is rejected",
			nextConfig:            `serverAddr = "0.0.0.0"`,
			storeProxy:            newTestVirtualNetProxyConfig("store-vnet"),
			wantRuntimeDependency: true,
		},
		{
			name:                  "Store VirtualNet visitor is rejected",
			nextConfig:            `serverAddr = "0.0.0.0"`,
			storeVisitor:          newTestVirtualNetVisitorConfig("store-vnet"),
			wantRuntimeDependency: true,
		},
		{
			name: "Store VirtualNet proxy overrides file proxy",
			nextConfig: `[[proxies]]
name = "shared"
type = "tcp"
localPort = 10080
remotePort = 10081
`,
			storeProxy:            newTestVirtualNetProxyConfig("shared"),
			wantRuntimeDependency: true,
		},
		{
			name: "Store non-VirtualNet proxy overrides file VirtualNet proxy",
			nextConfig: `[[proxies]]
name = "shared"
type = "stcp"
secretKey = "secret"
[proxies.plugin]
type = "virtual_net"
`,
			storeProxy:      newTestRawTCPProxyConfig("shared"),
			wantProxyPlugin: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			current := &v1.ClientCommonConfig{}
			if err := current.Complete(); err != nil {
				t.Fatalf("complete current config: %v", err)
			}

			configFile := filepath.Join(t.TempDir(), "frpc.toml")
			if err := os.WriteFile(configFile, []byte(tc.nextConfig), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}

			storeSource, err := source.NewStoreSource(source.StoreSourceConfig{
				Path: filepath.Join(t.TempDir(), "store.json"),
			})
			if err != nil {
				t.Fatalf("new store source: %v", err)
			}
			if tc.storeProxy != nil {
				if err := storeSource.AddProxy(tc.storeProxy); err != nil {
					t.Fatalf("add store proxy: %v", err)
				}
			}
			if tc.storeVisitor != nil {
				if err := storeSource.AddVisitor(tc.storeVisitor); err != nil {
					t.Fatalf("add store visitor: %v", err)
				}
			}

			configSource := source.NewConfigSource()
			aggregator := source.NewAggregator(configSource)
			aggregator.SetStoreSource(storeSource)
			svr := &Service{
				common:         current,
				reloadCommon:   current,
				configFilePath: configFile,
				unsafeFeatures: security.NewUnsafeFeatures(nil),
				aggregator:     aggregator,
				configSource:   configSource,
				storeSource:    storeSource,
			}

			err = (&serviceConfigManager{svr: svr}).ReloadFromFile(true)
			if tc.wantRuntimeDependency {
				if !errors.Is(err, configmgmt.ErrApplyConfig) || !strings.Contains(err.Error(), runtimeErr) {
					t.Fatalf("expected VirtualNet runtime dependency error, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("reload config: %v", err)
			}

			if len(svr.proxyCfgs) != 1 {
				t.Fatalf("expected one applied proxy, got %d", len(svr.proxyCfgs))
			}
			if got := svr.proxyCfgs[0].GetBaseConfig().Plugin.Type; got != tc.wantProxyPlugin {
				t.Fatalf("unexpected applied proxy plugin: %q", got)
			}
		})
	}
}

func TestServiceConfigManagerCreateStoreProxyConflict(t *testing.T) {
	storeSource, err := source.NewStoreSource(source.StoreSourceConfig{
		Path: filepath.Join(t.TempDir(), "store.json"),
	})
	if err != nil {
		t.Fatalf("new store source: %v", err)
	}
	if err := storeSource.AddProxy(newTestRawTCPProxyConfig("p1")); err != nil {
		t.Fatalf("seed proxy: %v", err)
	}

	agg := source.NewAggregator(source.NewConfigSource())
	agg.SetStoreSource(storeSource)

	mgr := &serviceConfigManager{
		svr: &Service{
			aggregator:   agg,
			configSource: agg.ConfigSource(),
			storeSource:  storeSource,
			reloadCommon: &v1.ClientCommonConfig{},
		},
	}

	_, err = mgr.CreateStoreProxy(newTestRawTCPProxyConfig("p1"))
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !errors.Is(err, configmgmt.ErrConflict) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceConfigManagerCreateStoreProxyKeepsStoreOnReloadFailure(t *testing.T) {
	storeSource, err := source.NewStoreSource(source.StoreSourceConfig{
		Path: filepath.Join(t.TempDir(), "store.json"),
	})
	if err != nil {
		t.Fatalf("new store source: %v", err)
	}

	mgr := &serviceConfigManager{
		svr: &Service{
			storeSource:  storeSource,
			reloadCommon: &v1.ClientCommonConfig{},
		},
	}

	_, err = mgr.CreateStoreProxy(newTestRawTCPProxyConfig("p1"))
	if err == nil {
		t.Fatal("expected apply config error")
	}
	if !errors.Is(err, configmgmt.ErrApplyConfig) {
		t.Fatalf("unexpected error: %v", err)
	}
	if storeSource.GetProxy("p1") == nil {
		t.Fatal("proxy should remain in store after reload failure")
	}
}

func TestServiceConfigManagerCreateStoreProxyStoreDisabled(t *testing.T) {
	mgr := &serviceConfigManager{
		svr: &Service{
			reloadCommon: &v1.ClientCommonConfig{},
		},
	}

	_, err := mgr.CreateStoreProxy(newTestRawTCPProxyConfig("p1"))
	if err == nil {
		t.Fatal("expected store disabled error")
	}
	if !errors.Is(err, configmgmt.ErrStoreDisabled) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceConfigManagerCreateStoreProxyDoesNotPersistRuntimeDefaults(t *testing.T) {
	storeSource, err := source.NewStoreSource(source.StoreSourceConfig{
		Path: filepath.Join(t.TempDir(), "store.json"),
	})
	if err != nil {
		t.Fatalf("new store source: %v", err)
	}
	agg := source.NewAggregator(source.NewConfigSource())
	agg.SetStoreSource(storeSource)

	mgr := &serviceConfigManager{
		svr: &Service{
			aggregator:   agg,
			configSource: agg.ConfigSource(),
			storeSource:  storeSource,
			reloadCommon: &v1.ClientCommonConfig{},
		},
	}

	persisted, err := mgr.CreateStoreProxy(newTestRawTCPProxyConfig("raw-proxy"))
	if err != nil {
		t.Fatalf("create store proxy: %v", err)
	}
	if persisted == nil {
		t.Fatal("expected persisted proxy to be returned")
	}

	got := storeSource.GetProxy("raw-proxy")
	if got == nil {
		t.Fatal("proxy not found in store")
	}
	if got.GetBaseConfig().LocalIP != "" {
		t.Fatalf("localIP was persisted with runtime default: %q", got.GetBaseConfig().LocalIP)
	}
	if got.GetBaseConfig().Transport.BandwidthLimitMode != "" {
		t.Fatalf("bandwidthLimitMode was persisted with runtime default: %q", got.GetBaseConfig().Transport.BandwidthLimitMode)
	}
}

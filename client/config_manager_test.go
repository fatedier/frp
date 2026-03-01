package client

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/fatedier/frp/client/configmgmt"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
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

	err = mgr.CreateStoreProxy(newTestRawTCPProxyConfig("p1"))
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

	err = mgr.CreateStoreProxy(newTestRawTCPProxyConfig("p1"))
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

	err := mgr.CreateStoreProxy(newTestRawTCPProxyConfig("p1"))
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

	err = mgr.CreateStoreProxy(newTestRawTCPProxyConfig("raw-proxy"))
	if err != nil {
		t.Fatalf("create store proxy: %v", err)
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

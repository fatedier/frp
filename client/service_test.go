package client

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

type failingConnector struct {
	err error
}

func (c *failingConnector) Open() error {
	return c.err
}

func (c *failingConnector) Connect() (net.Conn, error) {
	return nil, c.err
}

func (c *failingConnector) Close() error {
	return nil
}

func getFreeTCPPort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on ephemeral port: %v", err)
	}
	defer ln.Close()

	return ln.Addr().(*net.TCPAddr).Port
}

func TestRunStopsStartedComponentsOnInitialLoginFailure(t *testing.T) {
	port := getFreeTCPPort(t)
	agg := source.NewAggregator(source.NewConfigSource())

	svr, err := NewService(ServiceOptions{
		Common: &v1.ClientCommonConfig{
			LoginFailExit: lo.ToPtr(true),
			WebServer: v1.WebServerConfig{
				Addr: "127.0.0.1",
				Port: port,
			},
		},
		ConfigSourceAggregator: agg,
		ConnectorCreator: func(context.Context, *v1.ClientCommonConfig) Connector {
			return &failingConnector{err: errors.New("login boom")}
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	err = svr.Run(context.Background())
	if err == nil {
		t.Fatal("expected run error, got nil")
	}
	if !strings.Contains(err.Error(), "login boom") {
		t.Fatalf("unexpected error: %v", err)
	}
	if svr.webServer != nil {
		t.Fatal("expected web server to be cleaned up after initial login failure")
	}

	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("expected admin port to be released: %v", err)
	}
	_ = ln.Close()
}

func TestNewServiceDoesNotLeakAdminListenerOnAuthBuildFailure(t *testing.T) {
	port := getFreeTCPPort(t)
	agg := source.NewAggregator(source.NewConfigSource())

	_, err := NewService(ServiceOptions{
		Common: &v1.ClientCommonConfig{
			Auth: v1.AuthClientConfig{
				Method: v1.AuthMethodOIDC,
				OIDC: v1.AuthOIDCClientConfig{
					TokenEndpointURL: "://bad",
				},
			},
			WebServer: v1.WebServerConfig{
				Addr: "127.0.0.1",
				Port: port,
			},
		},
		ConfigSourceAggregator: agg,
	})
	if err == nil {
		t.Fatal("expected new service error, got nil")
	}
	if !strings.Contains(err.Error(), "auth.oidc.tokenEndpointURL") {
		t.Fatalf("unexpected error: %v", err)
	}

	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("expected admin port to remain free: %v", err)
	}
	_ = ln.Close()
}

func TestUpdateConfigSourceRollsBackReloadCommonOnReplaceAllFailure(t *testing.T) {
	prevCommon := &v1.ClientCommonConfig{User: "old-user"}
	newCommon := &v1.ClientCommonConfig{User: "new-user"}

	svr := &Service{
		configSource: source.NewConfigSource(),
		reloadCommon: prevCommon,
	}

	invalidProxy := &v1.TCPProxyConfig{}
	err := svr.UpdateConfigSource(newCommon, []v1.ProxyConfigurer{invalidProxy}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "proxy name cannot be empty") {
		t.Fatalf("unexpected error: %v", err)
	}

	if svr.reloadCommon != prevCommon {
		t.Fatalf("reloadCommon should roll back on ReplaceAll failure")
	}
}

func TestUpdateConfigSourceKeepsReloadCommonOnReloadFailure(t *testing.T) {
	prevCommon := &v1.ClientCommonConfig{User: "old-user"}
	newCommon := &v1.ClientCommonConfig{User: "new-user"}

	svr := &Service{
		// Keep configSource valid so ReplaceAll succeeds first.
		configSource: source.NewConfigSource(),
		reloadCommon: prevCommon,
		// Keep aggregator nil to force reload failure.
		aggregator: nil,
	}

	validProxy := &v1.TCPProxyConfig{
		ProxyBaseConfig: v1.ProxyBaseConfig{
			Name: "p1",
			Type: "tcp",
		},
	}
	err := svr.UpdateConfigSource(newCommon, []v1.ProxyConfigurer{validProxy}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "config aggregator is not initialized") {
		t.Fatalf("unexpected error: %v", err)
	}

	if svr.reloadCommon != newCommon {
		t.Fatalf("reloadCommon should keep new value on reload failure")
	}
}

func TestReloadConfigFromSourcesDoesNotMutateStoreConfigs(t *testing.T) {
	storeSource, err := source.NewStoreSource(source.StoreSourceConfig{
		Path: filepath.Join(t.TempDir(), "store.json"),
	})
	if err != nil {
		t.Fatalf("new store source: %v", err)
	}

	proxyCfg := &v1.TCPProxyConfig{
		ProxyBaseConfig: v1.ProxyBaseConfig{
			Name: "store-proxy",
			Type: "tcp",
		},
	}
	visitorCfg := &v1.STCPVisitorConfig{
		VisitorBaseConfig: v1.VisitorBaseConfig{
			Name: "store-visitor",
			Type: "stcp",
		},
	}
	if err := storeSource.AddProxy(proxyCfg); err != nil {
		t.Fatalf("add proxy to store: %v", err)
	}
	if err := storeSource.AddVisitor(visitorCfg); err != nil {
		t.Fatalf("add visitor to store: %v", err)
	}

	agg := source.NewAggregator(source.NewConfigSource())
	agg.SetStoreSource(storeSource)
	svr := &Service{
		aggregator:   agg,
		configSource: agg.ConfigSource(),
		storeSource:  storeSource,
		reloadCommon: &v1.ClientCommonConfig{},
	}

	if err := svr.reloadConfigFromSources(); err != nil {
		t.Fatalf("reload config from sources: %v", err)
	}

	gotProxy := storeSource.GetProxy("store-proxy")
	if gotProxy == nil {
		t.Fatalf("proxy not found in store")
	}
	if gotProxy.GetBaseConfig().LocalIP != "" {
		t.Fatalf("store proxy localIP should stay empty, got %q", gotProxy.GetBaseConfig().LocalIP)
	}

	gotVisitor := storeSource.GetVisitor("store-visitor")
	if gotVisitor == nil {
		t.Fatalf("visitor not found in store")
	}
	if gotVisitor.GetBaseConfig().BindAddr != "" {
		t.Fatalf("store visitor bindAddr should stay empty, got %q", gotVisitor.GetBaseConfig().BindAddr)
	}

	svr.cfgMu.RLock()
	defer svr.cfgMu.RUnlock()

	if len(svr.proxyCfgs) != 1 {
		t.Fatalf("expected 1 runtime proxy, got %d", len(svr.proxyCfgs))
	}
	if svr.proxyCfgs[0].GetBaseConfig().LocalIP != "127.0.0.1" {
		t.Fatalf("runtime proxy localIP should be defaulted, got %q", svr.proxyCfgs[0].GetBaseConfig().LocalIP)
	}

	if len(svr.visitorCfgs) != 1 {
		t.Fatalf("expected 1 runtime visitor, got %d", len(svr.visitorCfgs))
	}
	if svr.visitorCfgs[0].GetBaseConfig().BindAddr != "127.0.0.1" {
		t.Fatalf("runtime visitor bindAddr should be defaulted, got %q", svr.visitorCfgs[0].GetBaseConfig().BindAddr)
	}
}

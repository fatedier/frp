package client

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/auth"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
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

type scriptedConnector struct {
	cfg     *v1.ClientCommonConfig
	handler func(*v1.ClientCommonConfig, net.Conn)
}

func (c *scriptedConnector) Open() error {
	return nil
}

func (c *scriptedConnector) Connect() (net.Conn, error) {
	clientConn, serverConn := net.Pipe()
	go c.handler(c.cfg, serverConn)
	return clientConn, nil
}

func (c *scriptedConnector) Close() error {
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

func TestAutoTransportProxyURLShrinksCandidatesToTCP(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.Transport.Protocol = v1.TransportProtocolAuto
	common.Transport.ProxyURL = "http://127.0.0.1:8080"
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	candidates := manager.clientCandidates()
	if len(candidates) != 1 || candidates[0] != v1.TransportProtocolTCP {
		t.Fatalf("expected only tcp candidate, got %v", candidates)
	}
}

func TestAutoTransportBuildCandidatesUsesEndpointPorts(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}
	common.Transport.ProxyURL = ""

	manager := newAutoTransportManager(common, nil, nil)
	candidates := manager.buildCandidates(&msg.ServerHelloAuto{
		PreferOrder: []string{v1.TransportProtocolQUIC, v1.TransportProtocolTCP, v1.TransportProtocolKCP},
		Transports: []msg.TransportEndpoint{
			{Protocol: v1.TransportProtocolTCP, Addr: "0.0.0.0", Port: 7000, Enabled: true},
			{Protocol: v1.TransportProtocolKCP, Addr: "0.0.0.0", Port: 7000, Enabled: true},
			{Protocol: v1.TransportProtocolQUIC, Addr: "0.0.0.0", Port: 7002, Enabled: true},
		},
	})

	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(candidates))
	}
	if candidates[0].Protocol != v1.TransportProtocolQUIC || candidates[0].Port != 7002 {
		t.Fatalf("expected quic@7002 first, got %+v", candidates[0])
	}
	if candidates[1].Protocol != v1.TransportProtocolTCP || candidates[1].Port != 7000 {
		t.Fatalf("expected tcp@7000 second, got %+v", candidates[1])
	}
	if candidates[2].Protocol != v1.TransportProtocolKCP || candidates[2].Port != 7000 {
		t.Fatalf("expected kcp@7000 third, got %+v", candidates[2])
	}
	for _, candidate := range candidates {
		if candidate.Addr != "server.example.com" {
			t.Fatalf("expected unspecified endpoint addr to use configured server addr, got %q", candidate.Addr)
		}
	}
}

func TestAutoTransportAllowUDPFiltersUDPCandidates(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	common.Transport.Auto.AllowUDP = lo.ToPtr(false)
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	candidates := manager.buildCandidates(&msg.ServerHelloAuto{
		PreferOrder: []string{v1.TransportProtocolQUIC, v1.TransportProtocolTCP, v1.TransportProtocolKCP},
		Transports: []msg.TransportEndpoint{
			{Protocol: v1.TransportProtocolTCP, Addr: "0.0.0.0", Port: 7000, Enabled: true},
			{Protocol: v1.TransportProtocolKCP, Addr: "0.0.0.0", Port: 7000, Enabled: true},
			{Protocol: v1.TransportProtocolQUIC, Addr: "0.0.0.0", Port: 7002, Enabled: true},
		},
	})

	if len(candidates) != 1 {
		t.Fatalf("expected only one candidate, got %d", len(candidates))
	}
	if candidates[0].Protocol != v1.TransportProtocolTCP {
		t.Fatalf("expected only tcp when udp is disabled, got %+v", candidates[0])
	}
}

func TestAutoTransportSelectTransportCompletesBootstrapProbeAndSelection(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	common.Transport.Auto.ProbeCount = 1
	common.Transport.Auto.ProbeTimeoutMs = 500
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}
	common.Transport.ProxyURL = ""

	clientAuth, err := auth.BuildClientAuth(&common.Auth)
	if err != nil {
		t.Fatalf("build client auth: %v", err)
	}
	manager := newAutoTransportManager(common, clientAuth, func(_ context.Context, cfg *v1.ClientCommonConfig) Connector {
		return &scriptedConnector{
			cfg: cfg,
			handler: func(_ *v1.ClientCommonConfig, conn net.Conn) {
				defer conn.Close()
				rawMsg, err := msg.ReadMsg(conn)
				if err != nil {
					return
				}
				switch m := rawMsg.(type) {
				case *msg.ClientHelloAuto:
					_ = msg.WriteMsg(conn, &msg.ServerHelloAuto{
						ProtocolMode:       v1.TransportProtocolAuto,
						AutoEnabled:        true,
						AllowDynamicSwitch: true,
						PreferOrder:        []string{v1.TransportProtocolQUIC, v1.TransportProtocolTCP},
						Transports: []msg.TransportEndpoint{
							{Protocol: v1.TransportProtocolTCP, Addr: "0.0.0.0", Port: 7000, Enabled: true},
							{Protocol: v1.TransportProtocolQUIC, Addr: "0.0.0.0", Port: 7002, Enabled: true},
						},
						ServerAutoVersion: msg.AutoTransportVersion,
					})
				case *msg.ProbeTransport:
					_ = msg.WriteMsg(conn, &msg.ProbeTransportResp{
						Protocol:          m.Protocol,
						Port:              m.Port,
						ServerAutoVersion: msg.AutoTransportVersion,
					})
				}
			},
		}
	})
	manager.resolveIPAddrs = func(context.Context, string) ([]net.IPAddr, error) {
		return nil, errors.New("dns disabled in test")
	}

	selection, err := manager.selectTransport(context.Background(), autoTransportReasonStartup)
	if err != nil {
		t.Fatalf("select transport: %v", err)
	}
	if !selection.SendSelect {
		t.Fatal("expected selected auto transport to send SelectTransport")
	}
	if !selection.Dynamic {
		t.Fatal("expected dynamic mode with two common candidates")
	}
	if selection.Candidate.Protocol != v1.TransportProtocolQUIC || selection.Candidate.Port != 7002 {
		t.Fatalf("expected quic@7002 to be selected, got %+v", selection.Candidate)
	}
	if selection.Cfg.Transport.Protocol != v1.TransportProtocolQUIC || selection.Cfg.ServerPort != 7002 {
		t.Fatalf("expected selected config to use quic@7002, got protocol=%s port=%d",
			selection.Cfg.Transport.Protocol, selection.Cfg.ServerPort)
	}
	status := manager.status()
	if status.LastSuccessRates["quic@server.example.com:7002"] != 1 {
		t.Fatalf("expected quic success rate to be exposed, got %v", status.LastSuccessRates)
	}
	if status.LastSuccessRates["tcp@server.example.com:7000"] != 1 {
		t.Fatalf("expected tcp success rate to be exposed, got %v", status.LastSuccessRates)
	}
}

func TestAutoTransportExpandsDomainCandidatesToIPv4AndIPv6(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "frp.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	common.Transport.TLS.Enable = lo.ToPtr(true)
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}
	common.Transport.ProxyURL = ""

	manager := newAutoTransportManager(common, nil, nil)
	manager.resolveIPAddrs = func(_ context.Context, host string) ([]net.IPAddr, error) {
		if host != "frp.example.com" {
			t.Fatalf("expected resolver host frp.example.com, got %q", host)
		}
		return []net.IPAddr{
			{IP: net.ParseIP("192.0.2.10")},
			{IP: net.ParseIP("2001:db8::10")},
		}, nil
	}

	base := manager.buildCandidates(&msg.ServerHelloAuto{
		PreferOrder: []string{v1.TransportProtocolQUIC},
		Transports: []msg.TransportEndpoint{
			{Protocol: v1.TransportProtocolQUIC, Addr: "0.0.0.0", Port: 7002, Enabled: true},
		},
	})
	expanded := manager.expandCandidatesByResolvedAddr(context.Background(), base)
	if len(expanded) != 2 {
		t.Fatalf("expected ipv4 and ipv6 candidates, got %+v", expanded)
	}
	if expanded[0].Addr != "192.0.2.10" || expanded[1].Addr != "2001:db8::10" {
		t.Fatalf("expected resolved addresses to be used for dialing, got %+v", expanded)
	}
	for _, candidate := range expanded {
		if candidate.advertisedAddr() != "0.0.0.0" {
			t.Fatalf("expected advertised wildcard address to be preserved, got %+v", candidate)
		}
		if candidate.ServerName != "frp.example.com" {
			t.Fatalf("expected server name to be original domain, got %+v", candidate)
		}
		cfg := manager.configForCandidate(candidate)
		if cfg.ServerAddr != candidate.Addr {
			t.Fatalf("expected config to dial resolved address, got %q", cfg.ServerAddr)
		}
		if cfg.Transport.TLS.ServerName != "frp.example.com" {
			t.Fatalf("expected TLS server name to keep original domain, got %q", cfg.Transport.TLS.ServerName)
		}
	}
}

func TestAutoTransportServerVersionMismatchFallsBackStatic(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	common.Transport.Auto.ProbeCount = 1
	common.Transport.Auto.ProbeTimeoutMs = 500
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}
	common.Transport.ProxyURL = ""

	clientAuth, err := auth.BuildClientAuth(&common.Auth)
	if err != nil {
		t.Fatalf("build client auth: %v", err)
	}
	manager := newAutoTransportManager(common, clientAuth, func(_ context.Context, cfg *v1.ClientCommonConfig) Connector {
		return &scriptedConnector{
			cfg: cfg,
			handler: func(_ *v1.ClientCommonConfig, conn net.Conn) {
				defer conn.Close()
				if _, err := msg.ReadMsg(conn); err != nil {
					return
				}
				_ = msg.WriteMsg(conn, &msg.ServerHelloAuto{
					ProtocolMode:      v1.TransportProtocolAuto,
					AutoEnabled:       true,
					ServerAutoVersion: msg.AutoTransportVersion + 1,
				})
			},
		}
	})

	selection, err := manager.selectTransport(context.Background(), autoTransportReasonStartup)
	if err != nil {
		t.Fatalf("select transport: %v", err)
	}
	if selection.SendSelect {
		t.Fatal("static fallback should not send SelectTransport")
	}
	if selection.Candidate.Protocol != v1.TransportProtocolTCP {
		t.Fatalf("expected fallback tcp, got %+v", selection.Candidate)
	}
	if manager.status().State != autoTransportStateFallbackStatic {
		t.Fatalf("expected fallback static state, got %s", manager.status().State)
	}
}

func TestAutoTransportFixedModeKeepsSelectedCandidate(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	manager.selected = &autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000}
	candidates := manager.fixedModeCandidatesLocked([]autoTransportCandidate{
		{Protocol: v1.TransportProtocolQUIC, Addr: "server.example.com", Port: 7002},
		{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000},
	})

	if len(candidates) != 1 {
		t.Fatalf("expected fixed mode to keep one selected candidate, got %d", len(candidates))
	}
	if candidates[0].Protocol != v1.TransportProtocolTCP || candidates[0].Port != 7000 {
		t.Fatalf("expected selected tcp@7000 to be kept, got %+v", candidates[0])
	}
}

func TestAutoTransportFixedModeUsesLastGoodCandidate(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	manager.lastGood = v1.TransportProtocolQUIC
	candidates := manager.fixedModeCandidatesLocked([]autoTransportCandidate{
		{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000},
		{Protocol: v1.TransportProtocolQUIC, Addr: "server.example.com", Port: 7002},
	})

	if len(candidates) != 1 {
		t.Fatalf("expected fixed mode to keep one last good candidate, got %d", len(candidates))
	}
	if candidates[0].Protocol != v1.TransportProtocolQUIC || candidates[0].Port != 7002 {
		t.Fatalf("expected last good quic@7002 to be kept, got %+v", candidates[0])
	}
}

func TestAutoTransportPersistsLastGoodProtocol(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}
	common.Transport.ProxyURL = ""

	statePath := filepath.Join(t.TempDir(), "frpc.toml.auto_transport_state")
	manager := newAutoTransportManager(common, nil, nil, statePath)
	manager.reportLoginSuccess(v1.TransportProtocolQUIC)

	reloaded := newAutoTransportManager(common, nil, nil, statePath)
	if reloaded.lastGood != v1.TransportProtocolQUIC {
		t.Fatalf("expected persisted last good quic, got %q", reloaded.lastGood)
	}
}

func TestAutoTransportPersistLastGoodDisabled(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	common.Transport.Auto.PersistLastGood = lo.ToPtr(false)
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}
	common.Transport.ProxyURL = ""

	statePath := filepath.Join(t.TempDir(), "frpc.toml.auto_transport_state")
	manager := newAutoTransportManager(common, nil, nil, statePath)
	manager.reportLoginSuccess(v1.TransportProtocolQUIC)

	if _, err := os.Stat(statePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no persisted state, stat err: %v", err)
	}
}

func TestAutoTransportHysteresisKeepsCurrent(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	manager.selected = &autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000}
	manager.selectedAt = time.Now().Add(-time.Duration(common.Transport.Auto.StickyDurationSec+1) * time.Second)
	manager.lastSwitchAt = time.Now().Add(-time.Duration(common.Transport.Auto.CooldownSec+1) * time.Second)

	selected := manager.chooseCandidateByPolicy(autoTransportReasonStartup, []autoTransportProbeResult{
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolQUIC, Addr: "server.example.com", Port: 7002},
			Successes: 2,
			Score:     11999,
		},
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000},
			Successes: 2,
			Score:     10000,
		},
	})

	if selected.Candidate.Protocol != v1.TransportProtocolTCP {
		t.Fatalf("expected current tcp to be kept by hysteresis, got %+v", selected.Candidate)
	}
}

func TestAutoTransportHysteresisAllowsClearWinner(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	manager.selected = &autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000}
	manager.selectedAt = time.Now().Add(-time.Duration(common.Transport.Auto.StickyDurationSec+1) * time.Second)
	manager.lastSwitchAt = time.Now().Add(-time.Duration(common.Transport.Auto.CooldownSec+1) * time.Second)

	selected := manager.chooseCandidateByPolicy(autoTransportReasonStartup, []autoTransportProbeResult{
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolQUIC, Addr: "server.example.com", Port: 7002},
			Successes: 2,
			Score:     12000,
		},
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000},
			Successes: 2,
			Score:     10000,
		},
	})

	if selected.Candidate.Protocol != v1.TransportProtocolQUIC {
		t.Fatalf("expected quic to win after hysteresis threshold, got %+v", selected.Candidate)
	}
}

func TestAutoTransportStickyAndCooldownKeepCurrent(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	manager.selected = &autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000}
	manager.selectedAt = time.Now()

	selected := manager.chooseCandidateByPolicy(autoTransportReasonStartup, []autoTransportProbeResult{
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolQUIC, Addr: "server.example.com", Port: 7002},
			Successes: 2,
			Score:     20000,
		},
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000},
			Successes: 2,
			Score:     10000,
		},
	})
	if selected.Candidate.Protocol != v1.TransportProtocolTCP {
		t.Fatalf("expected sticky current tcp, got %+v", selected.Candidate)
	}

	manager.selectedAt = time.Now().Add(-time.Duration(common.Transport.Auto.StickyDurationSec+1) * time.Second)
	manager.lastSwitchAt = time.Now()
	selected = manager.chooseCandidateByPolicy(autoTransportReasonStartup, []autoTransportProbeResult{
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolQUIC, Addr: "server.example.com", Port: 7002},
			Successes: 2,
			Score:     20000,
		},
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000},
			Successes: 2,
			Score:     10000,
		},
	})
	if selected.Candidate.Protocol != v1.TransportProtocolTCP {
		t.Fatalf("expected cooldown current tcp, got %+v", selected.Candidate)
	}
}

func TestAutoTransportForcedReasonBypassesSticky(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	manager.selected = &autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000}
	manager.selectedAt = time.Now()
	manager.lastSwitchAt = time.Now()

	selected := manager.chooseCandidateByPolicy(autoTransportReasonControlClosed, []autoTransportProbeResult{
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolQUIC, Addr: "server.example.com", Port: 7002},
			Successes: 2,
			Score:     20000,
		},
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000},
			Successes: 2,
			Score:     10000,
		},
	})
	if selected.Candidate.Protocol != v1.TransportProtocolQUIC {
		t.Fatalf("expected forced reason to select best quic, got %+v", selected.Candidate)
	}
}

func TestAutoTransportStatusExposesScoresAndRemainingTime(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	manager.selected = &autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000}
	manager.selectedScore = 10000
	manager.previousProtocol = v1.TransportProtocolQUIC
	manager.selectedAt = time.Now()
	manager.lastSwitchAt = time.Now()
	manager.lastScores = map[string]int64{"tcp@server.example.com:7000": 10000}
	manager.lastScoreDetails = map[string]AutoTransportScoreDetail{
		"tcp@server.example.com:7000": {
			Strategy:        v1.AutoTransportStrategyBalanced,
			Total:           10000,
			Successes:       2,
			ProbeCount:      2,
			SuccessRate:     1,
			AvgRTTMs:        12,
			Priority:        0,
			SuccessScore:    20000,
			LatencyPenalty:  12,
			PriorityPenalty: 0,
		},
	}
	manager.lastSuccessRates = map[string]float64{"tcp@server.example.com:7000": 1}
	manager.lastProbeRTTMs = map[string]int64{"tcp@server.example.com:7000": 12}
	manager.lastProbeErrors = map[string]string{"quic@server.example.com:7002": "timeout"}
	manager.blacklistUntil[v1.TransportProtocolQUIC] = time.Now().Add(time.Minute)

	status := manager.status()
	if status.CurrentScore != 10000 {
		t.Fatalf("expected current score 10000, got %d", status.CurrentScore)
	}
	if status.PreviousProtocol != v1.TransportProtocolQUIC {
		t.Fatalf("expected previous protocol quic, got %q", status.PreviousProtocol)
	}
	if status.StickyRemainingSec <= 0 {
		t.Fatalf("expected sticky remaining to be positive")
	}
	if status.CooldownRemainingSec <= 0 {
		t.Fatalf("expected cooldown remaining to be positive")
	}
	if status.LastScores["tcp@server.example.com:7000"] != 10000 {
		t.Fatalf("expected endpoint score to be exposed, got %v", status.LastScores)
	}
	if status.Strategy != v1.AutoTransportStrategyBalanced {
		t.Fatalf("expected strategy to be exposed, got %q", status.Strategy)
	}
	if status.LastScoreDetails["tcp@server.example.com:7000"].LatencyPenalty != 12 {
		t.Fatalf("expected score detail to be exposed, got %+v", status.LastScoreDetails)
	}
	if status.LastSuccessRates["tcp@server.example.com:7000"] != 1 {
		t.Fatalf("expected endpoint success rate to be exposed, got %v", status.LastSuccessRates)
	}
	if status.LastProbeRTTMs["tcp@server.example.com:7000"] != 12 {
		t.Fatalf("expected endpoint probe RTT to be exposed, got %v", status.LastProbeRTTMs)
	}
	if status.LastProbeErrors["quic@server.example.com:7002"] != "timeout" {
		t.Fatalf("expected endpoint probe error to be exposed, got %v", status.LastProbeErrors)
	}
	if len(status.BlacklistProtocols) != 1 || status.BlacklistProtocols[0] != v1.TransportProtocolQUIC {
		t.Fatalf("expected quic in blacklist, got %v", status.BlacklistProtocols)
	}
}

func TestAutoTransportScoringStrategies(t *testing.T) {
	input := autoTransportScoringInput{
		Candidate:  autoTransportCandidate{Protocol: v1.TransportProtocolQUIC, Priority: 2},
		Successes:  2,
		ProbeCount: 2,
		AvgRTT:     100 * time.Millisecond,
		LastGood:   v1.TransportProtocolQUIC,
		Failures:   1,
	}

	balanced := autoTransportStrategyByName(v1.AutoTransportStrategyBalanced).Score(input)
	if balanced.Total != 20000-100-200+500-300 {
		t.Fatalf("unexpected balanced score detail: %+v", balanced)
	}

	latency := autoTransportStrategyByName(v1.AutoTransportStrategyLatency).Score(input)
	if latency.LatencyPenalty <= balanced.LatencyPenalty {
		t.Fatalf("latency strategy should weigh RTT more heavily: balanced=%+v latency=%+v", balanced, latency)
	}

	stability := autoTransportStrategyByName(v1.AutoTransportStrategyStability).Score(input)
	if stability.FailurePenalty <= balanced.FailurePenalty {
		t.Fatalf("stability strategy should penalize failures more heavily: balanced=%+v stability=%+v", balanced, stability)
	}

	fallback := autoTransportStrategyByName("unknown").Score(input)
	if fallback.Strategy != v1.AutoTransportStrategyBalanced {
		t.Fatalf("unknown strategy should fall back to balanced, got %+v", fallback)
	}
	if len(supportedAutoTransportScoreStrategyNames()) != len(v1.SupportedAutoTransportStrategies) {
		t.Fatalf("strategy registry and config validation are out of sync")
	}
}

func TestAutoTransportProbeStatusFromResults(t *testing.T) {
	successRates, rtts, probeErrors := probeStatusFromResults([]autoTransportProbeResult{
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000},
			Successes: 2,
			AvgRTT:    15 * time.Millisecond,
			Score:     10000,
		},
		{
			Candidate: autoTransportCandidate{Protocol: v1.TransportProtocolQUIC, Addr: "server.example.com", Port: 7002},
			Successes: 1,
			AvgRTT:    20 * time.Millisecond,
			Err:       errors.New("probe timeout"),
		},
	}, 2)

	if successRates["tcp@server.example.com:7000"] != 1 {
		t.Fatalf("expected tcp success rate 1, got %v", successRates)
	}
	if successRates["quic@server.example.com:7002"] != 0.5 {
		t.Fatalf("expected quic success rate 0.5, got %v", successRates)
	}
	if rtts["tcp@server.example.com:7000"] != 15 {
		t.Fatalf("expected tcp RTT 15ms, got %v", rtts)
	}
	if probeErrors["quic@server.example.com:7002"] != "probe timeout" {
		t.Fatalf("expected quic probe error, got %v", probeErrors)
	}
}

func TestAutoTransportWorkConnFailuresTriggerSwitching(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	common.Transport.Auto.DegradeThreshold = 2
	common.Transport.Auto.FailureThreshold = 2
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	manager.selected = &autoTransportCandidate{Protocol: v1.TransportProtocolQUIC, Addr: "server.example.com", Port: 7002}

	if manager.reportWorkConnFailure("work_conn_connect_error") {
		t.Fatalf("first work conn failure should not trigger switching")
	}
	status := manager.status()
	if status.State != autoTransportStateDegraded {
		t.Fatalf("expected degraded state after first failure, got %s", status.State)
	}
	if status.WorkConnFailures != 1 {
		t.Fatalf("expected one work conn failure, got %d", status.WorkConnFailures)
	}

	if !manager.reportWorkConnFailure("work_conn_connect_error") {
		t.Fatalf("second work conn failure should trigger switching")
	}
	status = manager.status()
	if status.State != autoTransportStateSwitching {
		t.Fatalf("expected switching state after threshold, got %s", status.State)
	}
	if manager.failures[v1.TransportProtocolQUIC] != 1 {
		t.Fatalf("expected one protocol runtime failure, got %d", manager.failures[v1.TransportProtocolQUIC])
	}
}

func TestAutoTransportHeartbeatTimeoutsAccumulateUntilPong(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	common.Transport.Auto.DegradeThreshold = 2
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	manager.selected = &autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000}

	if manager.reportHeartbeatTimeout() {
		t.Fatalf("first heartbeat timeout should not trigger switching")
	}
	manager.reportLoginSuccess(v1.TransportProtocolTCP)
	if !manager.reportHeartbeatTimeout() {
		t.Fatalf("second consecutive heartbeat timeout should trigger switching")
	}

	status := manager.status()
	if status.State != autoTransportStateSwitching {
		t.Fatalf("expected switching state, got %s", status.State)
	}
	if status.HeartbeatTimeouts != 2 {
		t.Fatalf("expected two heartbeat timeouts, got %d", status.HeartbeatTimeouts)
	}

	if manager.reportHeartbeatRTT(50 * time.Millisecond) {
		t.Fatalf("healthy pong should not trigger switching")
	}
	status = manager.status()
	if status.HeartbeatTimeouts != 0 {
		t.Fatalf("expected heartbeat timeouts reset after pong, got %d", status.HeartbeatTimeouts)
	}
}

func TestAutoTransportHeartbeatRTTDegradation(t *testing.T) {
	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "server.example.com"
	common.ServerPort = 7000
	common.Transport.Protocol = v1.TransportProtocolAuto
	common.Transport.Auto.DegradeThreshold = 2
	if err := common.Complete(); err != nil {
		t.Fatalf("complete common config: %v", err)
	}

	manager := newAutoTransportManager(common, nil, nil)
	manager.selected = &autoTransportCandidate{Protocol: v1.TransportProtocolTCP, Addr: "server.example.com", Port: 7000}

	if manager.reportHeartbeatRTT(100 * time.Millisecond) {
		t.Fatalf("baseline RTT should not trigger switching")
	}
	if manager.reportHeartbeatRTT(2 * time.Second) {
		t.Fatalf("first degraded RTT should not trigger switching")
	}
	if !manager.reportHeartbeatRTT(7 * time.Second) {
		t.Fatalf("second degraded RTT should trigger switching")
	}

	status := manager.status()
	if status.State != autoTransportStateSwitching {
		t.Fatalf("expected switching state, got %s", status.State)
	}
	if status.QualityDegradeCount != 2 {
		t.Fatalf("expected two quality degrade events, got %d", status.QualityDegradeCount)
	}
}

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

package client_test

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"

	frpc "github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	frps "github.com/fatedier/frp/server"
)

type autoTransportIntegrationConfig struct {
	serverProtocol   string
	advertise        []string
	prefer           []string
	withQUIC         bool
	clientProtocol   string
	clientCandidates []string
	clientAllowUDP   *bool
	wantProtocol     string
	wantDynamic      bool
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on ephemeral tcp port: %v", err)
	}
	defer ln.Close()

	return ln.Addr().(*net.TCPAddr).Port
}

func freeUDPPort(t *testing.T) int {
	t.Helper()

	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on ephemeral udp port: %v", err)
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).Port
}

func TestAutoTransportRealServerSelectsQUICControlSession(t *testing.T) {
	status := runAutoTransportIntegration(t, autoTransportIntegrationConfig{
		serverProtocol: v1.TransportProtocolAuto,
		advertise: []string{
			v1.TransportProtocolQUIC,
			v1.TransportProtocolTCP,
		},
		prefer: []string{
			v1.TransportProtocolQUIC,
			v1.TransportProtocolTCP,
		},
		withQUIC:       true,
		clientProtocol: v1.TransportProtocolAuto,
		clientCandidates: []string{
			v1.TransportProtocolQUIC,
			v1.TransportProtocolTCP,
		},
		wantProtocol: v1.TransportProtocolQUIC,
		wantDynamic:  true,
	})
	if status.CurrentPort == 0 {
		t.Fatalf("expected selected quic port in status, got %+v", status)
	}
}

func TestAutoTransportRealServerWithoutQUICSelectsTCP(t *testing.T) {
	status := runAutoTransportIntegration(t, autoTransportIntegrationConfig{
		serverProtocol: v1.TransportProtocolAuto,
		advertise: []string{
			v1.TransportProtocolQUIC,
			v1.TransportProtocolTCP,
		},
		prefer: []string{
			v1.TransportProtocolQUIC,
			v1.TransportProtocolTCP,
		},
		clientProtocol: v1.TransportProtocolAuto,
		clientCandidates: []string{
			v1.TransportProtocolQUIC,
			v1.TransportProtocolTCP,
		},
		wantProtocol: v1.TransportProtocolTCP,
		wantDynamic:  false,
	})
	for endpoint := range status.LastScores {
		if strings.HasPrefix(endpoint, "quic@") {
			t.Fatalf("quic without a server endpoint must not be probed, got scores %+v", status.LastScores)
		}
	}
}

func TestAutoTransportRealClientDisablesUDPAndSelectsTCP(t *testing.T) {
	status := runAutoTransportIntegration(t, autoTransportIntegrationConfig{
		serverProtocol: v1.TransportProtocolAuto,
		advertise: []string{
			v1.TransportProtocolQUIC,
			v1.TransportProtocolTCP,
		},
		prefer: []string{
			v1.TransportProtocolQUIC,
			v1.TransportProtocolTCP,
		},
		withQUIC:       true,
		clientProtocol: v1.TransportProtocolAuto,
		clientCandidates: []string{
			v1.TransportProtocolQUIC,
			v1.TransportProtocolTCP,
		},
		clientAllowUDP: lo.ToPtr(false),
		wantProtocol:   v1.TransportProtocolTCP,
		wantDynamic:    false,
	})
	for endpoint := range status.LastScores {
		if strings.HasPrefix(endpoint, "quic@") {
			t.Fatalf("quic must not be probed when client disables udp, got scores %+v", status.LastScores)
		}
	}
}

func TestAutoTransportRealStaticServerFallsBackToTCP(t *testing.T) {
	status := runAutoTransportIntegration(t, autoTransportIntegrationConfig{
		serverProtocol: v1.TransportProtocolTCP,
		withQUIC:       true,
		clientProtocol: v1.TransportProtocolAuto,
		clientCandidates: []string{
			v1.TransportProtocolQUIC,
			v1.TransportProtocolTCP,
		},
		wantProtocol: v1.TransportProtocolTCP,
		wantDynamic:  false,
	})
	if status.SwitchCount != 0 {
		t.Fatalf("static fallback should not count as a dynamic switch, got %+v", status)
	}
}

func runAutoTransportIntegration(t *testing.T, cfg autoTransportIntegrationConfig) frpc.AutoTransportStatus {
	t.Helper()

	t.Setenv("http_proxy", "")
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("https_proxy", "")
	t.Setenv("HTTPS_PROXY", "")

	bindPort := freeTCPPort(t)
	quicPort := 0
	if cfg.withQUIC {
		quicPort = freeUDPPort(t)
	}

	serverCfg := &v1.ServerConfig{
		BindAddr:     "127.0.0.1",
		BindPort:     bindPort,
		QUICBindPort: quicPort,
	}
	serverCfg.Transport.Protocol = cfg.serverProtocol
	serverCfg.Transport.TCPMux = lo.ToPtr(false)
	if len(cfg.advertise) > 0 {
		serverCfg.Transport.Auto.AdvertiseProtocols = cfg.advertise
	}
	if len(cfg.prefer) > 0 {
		serverCfg.Transport.Auto.PreferOrder = cfg.prefer
	}
	if err := serverCfg.Complete(); err != nil {
		t.Fatalf("complete server config: %v", err)
	}
	serverSvc, err := frps.NewService(serverCfg)
	if err != nil {
		t.Fatalf("new server service: %v", err)
	}
	serverCtx, serverCancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		serverCancel()
		_ = serverSvc.Close()
	})
	go serverSvc.Run(serverCtx)

	common := &v1.ClientCommonConfig{}
	common.ServerAddr = "127.0.0.1"
	common.ServerPort = bindPort
	common.LoginFailExit = lo.ToPtr(true)
	common.Transport.Protocol = cfg.clientProtocol
	common.Transport.TCPMux = lo.ToPtr(false)
	common.Transport.TLS.Enable = lo.ToPtr(false)
	if len(cfg.clientCandidates) > 0 {
		common.Transport.Auto.Candidates = cfg.clientCandidates
	}
	if cfg.clientAllowUDP != nil {
		common.Transport.Auto.AllowUDP = cfg.clientAllowUDP
	}
	common.Transport.Auto.ProbeCount = 1
	common.Transport.Auto.ProbeTimeoutMs = 1000
	if err := common.Complete(); err != nil {
		t.Fatalf("complete client config: %v", err)
	}
	common.Transport.ProxyURL = ""

	clientSvc, err := frpc.NewService(frpc.ServiceOptions{
		Common:                 common,
		ConfigSourceAggregator: source.NewAggregator(source.NewConfigSource()),
	})
	if err != nil {
		t.Fatalf("new client service: %v", err)
	}
	clientCtx, clientCancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		clientCancel()
		clientSvc.Close()
	})
	done := make(chan error, 1)
	go func() {
		done <- clientSvc.Run(clientCtx)
	}()

	deadline := time.Now().Add(5 * time.Second)
	var status frpc.AutoTransportStatus
	for {
		statusAny, err := clientSvc.TransportStatus(nil)
		if err != nil {
			t.Fatalf("transport status: %v", err)
		}
		status = statusAny.(frpc.AutoTransportStatus)
		if status.State == "CONNECTED" && status.CurrentProtocol == cfg.wantProtocol && status.Dynamic == cfg.wantDynamic {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for protocol %s dynamic %v, last status: %+v",
				cfg.wantProtocol, cfg.wantDynamic, status)
		}
		time.Sleep(50 * time.Millisecond)
	}

	clientCancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("client run returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for client shutdown")
	}
	return status
}

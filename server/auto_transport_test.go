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

package server

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	splugin "github.com/fatedier/frp/pkg/plugin/server"
)

func newAutoTransportServiceForTest(t *testing.T, cfg *v1.ServerConfig) *Service {
	t.Helper()

	if err := cfg.Complete(); err != nil {
		t.Fatalf("complete server config: %v", err)
	}
	authRuntime, err := auth.BuildServerAuth(&cfg.Auth)
	if err != nil {
		t.Fatalf("build server auth: %v", err)
	}
	return &Service{
		auth: authRuntime,
		cfg:  cfg,
	}
}

func validAutoClientHello(t *testing.T, token string) *msg.ClientHelloAuto {
	t.Helper()

	clientAuth, err := auth.BuildClientAuth(&v1.AuthClientConfig{
		Method: v1.AuthMethodToken,
		Token:  token,
	})
	if err != nil {
		t.Fatalf("build client auth: %v", err)
	}

	login := &msg.Login{Timestamp: time.Now().Unix()}
	if err := clientAuth.Setter.SetLogin(login); err != nil {
		t.Fatalf("set login auth: %v", err)
	}
	return &msg.ClientHelloAuto{
		ProtocolMode:      v1.TransportProtocolAuto,
		ClientAutoVersion: msg.AutoTransportVersion,
		PrivilegeKey:      login.PrivilegeKey,
		Timestamp:         login.Timestamp,
	}
}

func readServerHelloFromHandler(t *testing.T, svr *Service, hello *msg.ClientHelloAuto) msg.ServerHelloAuto {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	if err := clientConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		svr.handleClientHelloAuto(serverConn, hello)
	}()

	var resp msg.ServerHelloAuto
	if err := msg.ReadMsgInto(clientConn, &resp); err != nil {
		t.Fatalf("read server hello: %v", err)
	}
	<-done
	return resp
}

func readProbeRespFromHandler(
	t *testing.T,
	svr *Service,
	probe *msg.ProbeTransport,
	entry autoTransportEntry,
) msg.ProbeTransportResp {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	if err := clientConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		svr.handleProbeTransport(serverConn, probe, entry)
	}()

	var resp msg.ProbeTransportResp
	if err := msg.ReadMsgInto(clientConn, &resp); err != nil {
		t.Fatalf("read probe response: %v", err)
	}
	<-done
	return resp
}

func hasEndpoint(endpoints []msg.TransportEndpoint, protocol string, port int) bool {
	for _, ep := range endpoints {
		if ep.Enabled && ep.Protocol == protocol && ep.Port == port {
			return true
		}
	}
	return false
}

func TestAutoTransportServerHelloAdvertisesFixedEndpointModel(t *testing.T) {
	cfg := &v1.ServerConfig{
		BindPort:     7000,
		KCPBindPort:  7000,
		QUICBindPort: 7002,
	}
	cfg.Auth.Token = "secret"
	cfg.Transport.Protocol = v1.TransportProtocolAuto
	svr := newAutoTransportServiceForTest(t, cfg)

	resp := readServerHelloFromHandler(t, svr, validAutoClientHello(t, "secret"))
	if resp.Error != "" {
		t.Fatalf("unexpected server hello error: %s", resp.Error)
	}
	if !resp.AutoEnabled {
		t.Fatal("expected auto transport to be enabled")
	}
	if !resp.AllowDynamicSwitch {
		t.Fatal("expected dynamic switch to be allowed")
	}
	if !hasEndpoint(resp.Transports, v1.TransportProtocolTCP, 7000) {
		t.Fatalf("expected tcp@7000 endpoint, got %+v", resp.Transports)
	}
	if !hasEndpoint(resp.Transports, v1.TransportProtocolKCP, 7000) {
		t.Fatalf("expected kcp@7000 endpoint, got %+v", resp.Transports)
	}
	if !hasEndpoint(resp.Transports, v1.TransportProtocolQUIC, 7002) {
		t.Fatalf("expected quic@7002 endpoint, got %+v", resp.Transports)
	}
	if !hasEndpoint(resp.Transports, v1.TransportProtocolWebsocket, 7000) {
		t.Fatalf("expected websocket@7000 endpoint, got %+v", resp.Transports)
	}
	if !hasEndpoint(resp.Transports, v1.TransportProtocolWSS, 7000) {
		t.Fatalf("expected wss@7000 endpoint, got %+v", resp.Transports)
	}
}

func TestValidateSelectedTransportRequiresAdvertisedEndpoint(t *testing.T) {
	cfg := &v1.ServerConfig{
		BindPort:     7000,
		KCPBindPort:  7000,
		QUICBindPort: 7002,
	}
	cfg.Transport.Protocol = v1.TransportProtocolAuto
	svr := newAutoTransportServiceForTest(t, cfg)

	if err := svr.validateSelectedTransport(v1.TransportProtocolTCP, "server.example.com", 7000); err != nil {
		t.Fatalf("expected tcp@7000 to be valid: %v", err)
	}
	if err := svr.validateSelectedTransport(v1.TransportProtocolKCP, "server.example.com", 7000); err != nil {
		t.Fatalf("expected kcp@7000 to be valid: %v", err)
	}
	if err := svr.validateSelectedTransport(v1.TransportProtocolQUIC, "server.example.com", 7002); err != nil {
		t.Fatalf("expected quic@7002 to be valid: %v", err)
	}
	if err := svr.validateSelectedTransport(v1.TransportProtocolQUIC, "server.example.com", 7000); err == nil {
		t.Fatal("expected quic@7000 to be rejected")
	}
	if err := svr.validateSelectedTransport(v1.TransportProtocolKCP, "server.example.com", 7002); err == nil {
		t.Fatal("expected kcp@7002 to be rejected")
	}
}

func TestValidateSelectedTransportRequiresMatchingConnectionEntry(t *testing.T) {
	cfg := &v1.ServerConfig{
		BindPort:     7000,
		KCPBindPort:  7000,
		QUICBindPort: 7002,
	}
	cfg.Transport.Protocol = v1.TransportProtocolAuto
	svr := newAutoTransportServiceForTest(t, cfg)

	tcpEntry := autoTransportEntry{
		Protocols: []string{v1.TransportProtocolTCP},
		Port:      7000,
	}
	if err := svr.validateSelectedTransportForEntry(v1.TransportProtocolTCP, "server.example.com", 7000, tcpEntry); err != nil {
		t.Fatalf("expected tcp on tcp entry to be valid: %v", err)
	}
	if err := svr.validateSelectedTransportForEntry(v1.TransportProtocolQUIC, "server.example.com", 7002, tcpEntry); err == nil {
		t.Fatal("expected quic selected on tcp entry to be rejected")
	}

	tlsEntry := autoTransportEntry{
		Protocols: []string{v1.TransportProtocolTCP, v1.TransportProtocolWSS},
		Port:      7000,
	}
	if err := svr.validateSelectedTransportForEntry(v1.TransportProtocolWSS, "server.example.com", 7000, tlsEntry); err != nil {
		t.Fatalf("expected wss on tls entry to be valid: %v", err)
	}
	if err := svr.validateSelectedTransportForEntry(v1.TransportProtocolWebsocket, "server.example.com", 7000, tlsEntry); err == nil {
		t.Fatal("expected websocket selected on tls entry to be rejected")
	}
}

func TestValidateSelectedTransportRequiresAdvertisedAddrWhenSpecific(t *testing.T) {
	cfg := &v1.ServerConfig{
		BindAddr:     "127.0.0.1",
		BindPort:     7000,
		KCPBindPort:  7000,
		QUICBindPort: 7002,
	}
	cfg.Transport.Protocol = v1.TransportProtocolAuto
	svr := newAutoTransportServiceForTest(t, cfg)

	if err := svr.validateSelectedTransport(v1.TransportProtocolTCP, "127.0.0.1", 7000); err != nil {
		t.Fatalf("expected tcp@127.0.0.1:7000 to be valid: %v", err)
	}
	if err := svr.validateSelectedTransport(v1.TransportProtocolTCP, "192.0.2.1", 7000); err == nil {
		t.Fatal("expected tcp@192.0.2.1:7000 to be rejected")
	}
}

func TestPeekAutoTransportWebsocketReplaysPrefix(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	payload := []byte("GET " + "/~!frp" + " HTTP/1.1\r\n\r\n")
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer clientConn.Close()
		_, _ = clientConn.Write(payload)
	}()

	conn, isWebsocket := peekAutoTransportWebsocket(serverConn)
	defer conn.Close()
	if !isWebsocket {
		t.Fatal("expected websocket prefix to be detected")
	}

	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read replayed payload: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("expected replayed payload %q, got %q", payload, got)
	}
	<-done
}

type rejectingLoginPlugin struct{}

func (rejectingLoginPlugin) Name() string { return "rejecting-login" }

func (rejectingLoginPlugin) IsSupport(op string) bool { return op == splugin.OpLogin }

func (rejectingLoginPlugin) Handle(
	_ context.Context,
	_ string,
	content any,
) (*splugin.Response, any, error) {
	return &splugin.Response{
		Reject:       true,
		RejectReason: "blocked by plugin",
	}, content, nil
}

func TestClientHelloAutoRunsLoginPlugin(t *testing.T) {
	cfg := &v1.ServerConfig{
		BindPort:     7000,
		KCPBindPort:  7000,
		QUICBindPort: 7002,
	}
	cfg.Auth.Token = "secret"
	cfg.Transport.Protocol = v1.TransportProtocolAuto
	svr := newAutoTransportServiceForTest(t, cfg)
	svr.pluginManager = splugin.NewManager()
	svr.pluginManager.Register(rejectingLoginPlugin{})

	resp := readServerHelloFromHandler(t, svr, validAutoClientHello(t, "secret"))
	if resp.Error == "" {
		t.Fatal("expected plugin rejection")
	}
	if !strings.Contains(resp.Error, "blocked by plugin") {
		t.Fatalf("unexpected plugin error: %s", resp.Error)
	}
	if resp.AutoEnabled {
		t.Fatal("expected auto transport to be disabled in failed plugin response")
	}
}

type requiringLoginIdentityPlugin struct{}

func (requiringLoginIdentityPlugin) Name() string { return "requiring-login-identity" }

func (requiringLoginIdentityPlugin) IsSupport(op string) bool { return op == splugin.OpLogin }

func (requiringLoginIdentityPlugin) Handle(
	_ context.Context,
	_ string,
	content any,
) (*splugin.Response, any, error) {
	loginContent := content.(splugin.LoginContent)
	login := loginContent.Login
	if login.User != "alice" || login.ClientID != "client-1" || login.Metas["role"] != "edge" {
		return &splugin.Response{
			Reject:       true,
			RejectReason: "missing auto login identity",
		}, content, nil
	}
	return &splugin.Response{Unchange: true}, content, nil
}

func TestClientHelloAutoPassesLoginIdentityToPlugin(t *testing.T) {
	cfg := &v1.ServerConfig{
		BindPort:     7000,
		KCPBindPort:  7000,
		QUICBindPort: 7002,
	}
	cfg.Auth.Token = "secret"
	cfg.Transport.Protocol = v1.TransportProtocolAuto
	svr := newAutoTransportServiceForTest(t, cfg)
	svr.pluginManager = splugin.NewManager()
	svr.pluginManager.Register(requiringLoginIdentityPlugin{})

	hello := validAutoClientHello(t, "secret")
	hello.Login = &msg.Login{
		PrivilegeKey: hello.PrivilegeKey,
		Timestamp:    hello.Timestamp,
		User:         "alice",
		ClientID:     "client-1",
		Metas:        map[string]string{"role": "edge"},
	}
	resp := readServerHelloFromHandler(t, svr, hello)
	if resp.Error != "" {
		t.Fatalf("unexpected plugin rejection: %s", resp.Error)
	}
}

func TestProbeTransportPassesLoginIdentityToPlugin(t *testing.T) {
	cfg := &v1.ServerConfig{
		BindPort:     7000,
		KCPBindPort:  7000,
		QUICBindPort: 7002,
	}
	cfg.Auth.Token = "secret"
	cfg.Transport.Protocol = v1.TransportProtocolAuto
	svr := newAutoTransportServiceForTest(t, cfg)
	svr.pluginManager = splugin.NewManager()
	svr.pluginManager.Register(requiringLoginIdentityPlugin{})

	hello := validAutoClientHello(t, "secret")
	probe := &msg.ProbeTransport{
		Protocol:          v1.TransportProtocolTCP,
		Addr:              "server.example.com",
		Port:              7000,
		ClientAutoVersion: msg.AutoTransportVersion,
		PrivilegeKey:      hello.PrivilegeKey,
		Timestamp:         hello.Timestamp,
		Login: &msg.Login{
			PrivilegeKey: hello.PrivilegeKey,
			Timestamp:    hello.Timestamp,
			User:         "alice",
			ClientID:     "client-1",
			Metas:        map[string]string{"role": "edge"},
		},
	}
	resp := readProbeRespFromHandler(t, svr, probe, autoTransportEntry{
		Protocols: []string{v1.TransportProtocolTCP},
		Port:      7000,
	})
	if resp.Error != "" {
		t.Fatalf("unexpected plugin rejection: %s", resp.Error)
	}
}

func TestClientHelloAutoRejectsInvalidAuth(t *testing.T) {
	cfg := &v1.ServerConfig{
		BindPort:     7000,
		KCPBindPort:  7000,
		QUICBindPort: 7002,
	}
	cfg.Auth.Token = "secret"
	cfg.Transport.Protocol = v1.TransportProtocolAuto
	svr := newAutoTransportServiceForTest(t, cfg)

	resp := readServerHelloFromHandler(t, svr, &msg.ClientHelloAuto{
		ProtocolMode:      v1.TransportProtocolAuto,
		ClientAutoVersion: msg.AutoTransportVersion,
		PrivilegeKey:      "wrong",
		Timestamp:         time.Now().Unix(),
	})
	if resp.Error == "" {
		t.Fatal("expected auth error")
	}
	if !strings.Contains(resp.Error, "auto transport auth failed") {
		t.Fatalf("unexpected auth error: %s", resp.Error)
	}
	if resp.AutoEnabled {
		t.Fatal("expected auto transport to be disabled in failed auth response")
	}
}

func TestClientHelloAutoRejectsUnsupportedVersion(t *testing.T) {
	cfg := &v1.ServerConfig{
		BindPort:     7000,
		KCPBindPort:  7000,
		QUICBindPort: 7002,
	}
	cfg.Auth.Token = "secret"
	cfg.Transport.Protocol = v1.TransportProtocolAuto
	svr := newAutoTransportServiceForTest(t, cfg)

	hello := validAutoClientHello(t, "secret")
	hello.ClientAutoVersion = msg.AutoTransportVersion + 1
	resp := readServerHelloFromHandler(t, svr, hello)
	if resp.Error == "" {
		t.Fatal("expected version error")
	}
	if !strings.Contains(resp.Error, "unsupported client auto transport version") {
		t.Fatalf("unexpected version error: %s", resp.Error)
	}
	if resp.AutoEnabled {
		t.Fatal("expected auto transport to be disabled in failed version response")
	}
}

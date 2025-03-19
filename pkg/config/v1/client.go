// Copyright 2023 The frp Authors
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
	"os"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/util/util"
)

type ClientConfig struct {
	ClientCommonConfig

	Proxies  []TypedProxyConfig   `json:"proxies,omitempty"`
	Visitors []TypedVisitorConfig `json:"visitors,omitempty"`
}

type ClientCommonConfig struct {
	APIMetadata

	Auth AuthClientConfig `json:"auth,omitempty"`
	// User specifies a prefix for proxy names to distinguish them from other
	// clients. If this value is not "", proxy names will automatically be
	// changed to "{user}.{proxy_name}".
	User string `json:"user,omitempty"`

	// ServerAddr specifies the address of the server to connect to. By
	// default, this value is "0.0.0.0".
	ServerAddr string `json:"serverAddr,omitempty"`
	// ServerPort specifies the port to connect to the server on. By default,
	// this value is 7000.
	ServerPort int `json:"serverPort,omitempty"`
	// STUN server to help penetrate NAT hole.
	NatHoleSTUNServer string `json:"natHoleStunServer,omitempty"`
	// DNSServer specifies a DNS server address for FRPC to use. If this value
	// is "", the default DNS will be used.
	DNSServer string `json:"dnsServer,omitempty"`
	// LoginFailExit controls whether or not the client should exit after a
	// failed login attempt. If false, the client will retry until a login
	// attempt succeeds. By default, this value is true.
	LoginFailExit *bool `json:"loginFailExit,omitempty"`
	// Start specifies a set of enabled proxies by name. If this set is empty,
	// all supplied proxies are enabled. By default, this value is an empty
	// set.
	Start []string `json:"start,omitempty"`

	Log       LogConfig             `json:"log,omitempty"`
	WebServer WebServerConfig       `json:"webServer,omitempty"`
	Transport ClientTransportConfig `json:"transport,omitempty"`

	// UDPPacketSize specifies the udp packet size
	// By default, this value is 1500
	UDPPacketSize int64 `json:"udpPacketSize,omitempty"`
	// Client metadata info
	Metadatas map[string]string `json:"metadatas,omitempty"`

	// Include other config files for proxies.
	IncludeConfigFiles []string `json:"includes,omitempty"`
}

func (c *ClientCommonConfig) Complete() {
	c.ServerAddr = util.EmptyOr(c.ServerAddr, "0.0.0.0")
	c.ServerPort = util.EmptyOr(c.ServerPort, 7000)
	c.LoginFailExit = util.EmptyOr(c.LoginFailExit, lo.ToPtr(true))
	c.NatHoleSTUNServer = util.EmptyOr(c.NatHoleSTUNServer, "stun.easyvoip.com:3478")

	c.Auth.Complete()
	c.Log.Complete()
	c.Transport.Complete()
	c.WebServer.Complete()

	c.UDPPacketSize = util.EmptyOr(c.UDPPacketSize, 1500)
}

type ClientTransportConfig struct {
	// Protocol specifies the protocol to use when interacting with the server.
	// Valid values are "tcp", "kcp", "quic", "websocket" and "wss". By default, this value
	// is "tcp".
	Protocol string `json:"protocol,omitempty"`
	// The maximum amount of time a dial to server will wait for a connect to complete.
	DialServerTimeout int64 `json:"dialServerTimeout,omitempty"`
	// DialServerKeepAlive specifies the interval between keep-alive probes for an active network connection between frpc and frps.
	// If negative, keep-alive probes are disabled.
	DialServerKeepAlive int64 `json:"dialServerKeepalive,omitempty"`
	// ConnectServerLocalIP specifies the address of the client bind when it connect to server.
	// Note: This value only use in TCP/Websocket protocol. Not support in KCP protocol.
	ConnectServerLocalIP string `json:"connectServerLocalIP,omitempty"`
	// ProxyURL specifies a proxy address to connect to the server through. If
	// this value is "", the server will be connected to directly. By default,
	// this value is read from the "http_proxy" environment variable.
	ProxyURL string `json:"proxyURL,omitempty"`
	// PoolCount specifies the number of connections the client will make to
	// the server in advance.
	PoolCount int `json:"poolCount,omitempty"`
	// TCPMux toggles TCP stream multiplexing. This allows multiple requests
	// from a client to share a single TCP connection. If this value is true,
	// the server must have TCP multiplexing enabled as well. By default, this
	// value is true.
	TCPMux *bool `json:"tcpMux,omitempty"`
	// TCPMuxKeepaliveInterval specifies the keep alive interval for TCP stream multiplier.
	// If TCPMux is true, heartbeat of application layer is unnecessary because it can only rely on heartbeat in TCPMux.
	TCPMuxKeepaliveInterval int64 `json:"tcpMuxKeepaliveInterval,omitempty"`
	// QUIC protocol options.
	QUIC QUICOptions `json:"quic,omitempty"`
	// HeartBeatInterval specifies at what interval heartbeats are sent to the
	// server, in seconds. It is not recommended to change this value. By
	// default, this value is 30. Set negative value to disable it.
	HeartbeatInterval int64 `json:"heartbeatInterval,omitempty"`
	// HeartBeatTimeout specifies the maximum allowed heartbeat response delay
	// before the connection is terminated, in seconds. It is not recommended
	// to change this value. By default, this value is 90. Set negative value to disable it.
	HeartbeatTimeout int64 `json:"heartbeatTimeout,omitempty"`
	// TLS specifies TLS settings for the connection to the server.
	TLS TLSClientConfig `json:"tls,omitempty"`
}

func (c *ClientTransportConfig) Complete() {
	c.Protocol = util.EmptyOr(c.Protocol, "tcp")
	c.DialServerTimeout = util.EmptyOr(c.DialServerTimeout, 10)
	c.DialServerKeepAlive = util.EmptyOr(c.DialServerKeepAlive, 7200)
	c.ProxyURL = util.EmptyOr(c.ProxyURL, os.Getenv("http_proxy"))
	c.PoolCount = util.EmptyOr(c.PoolCount, 1)
	c.TCPMux = util.EmptyOr(c.TCPMux, lo.ToPtr(true))
	c.TCPMuxKeepaliveInterval = util.EmptyOr(c.TCPMuxKeepaliveInterval, 30)
	if lo.FromPtr(c.TCPMux) {
		// If TCPMux is enabled, heartbeat of application layer is unnecessary because we can rely on heartbeat in tcpmux.
		c.HeartbeatInterval = util.EmptyOr(c.HeartbeatInterval, -1)
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, -1)
	} else {
		c.HeartbeatInterval = util.EmptyOr(c.HeartbeatInterval, 30)
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, 90)
	}
	c.QUIC.Complete()
	c.TLS.Complete()
}

type TLSClientConfig struct {
	// TLSEnable specifies whether or not TLS should be used when communicating
	// with the server. If "tls.certFile" and "tls.keyFile" are valid,
	// client will load the supplied tls configuration.
	// Since v0.50.0, the default value has been changed to true, and tls is enabled by default.
	Enable *bool `json:"enable,omitempty"`
	// If DisableCustomTLSFirstByte is set to false, frpc will establish a connection with frps using the
	// first custom byte when tls is enabled.
	// Since v0.50.0, the default value has been changed to true, and the first custom byte is disabled by default.
	DisableCustomTLSFirstByte *bool `json:"disableCustomTLSFirstByte,omitempty"`

	TLSConfig
}

func (c *TLSClientConfig) Complete() {
	c.Enable = util.EmptyOr(c.Enable, lo.ToPtr(true))
	c.DisableCustomTLSFirstByte = util.EmptyOr(c.DisableCustomTLSFirstByte, lo.ToPtr(true))
}

type AuthClientConfig struct {
	// Method specifies what authentication method to use to
	// authenticate frpc with frps. If "token" is specified - token will be
	// read into login message. If "oidc" is specified - OIDC (Open ID Connect)
	// token will be issued using OIDC settings. By default, this value is "token".
	Method AuthMethod `json:"method,omitempty"`
	// Specify whether to include auth info in additional scope.
	// Current supported scopes are: "HeartBeats", "NewWorkConns".
	AdditionalScopes []AuthScope `json:"additionalScopes,omitempty"`
	// Token specifies the authorization token used to create keys to be sent
	// to the server. The server must have a matching token for authorization
	// to succeed.  By default, this value is "".
	Token string               `json:"token,omitempty"`
	OIDC  AuthOIDCClientConfig `json:"oidc,omitempty"`

	Secret string `json:"secret"`
}

func (c *AuthClientConfig) Complete() {
	c.Method = util.EmptyOr(c.Method, "token")
}

type AuthOIDCClientConfig struct {
	// ClientID specifies the client ID to use to get a token in OIDC authentication.
	ClientID string `json:"clientID,omitempty"`
	// ClientSecret specifies the client secret to use to get a token in OIDC
	// authentication.
	ClientSecret string `json:"clientSecret,omitempty"`
	// Audience specifies the audience of the token in OIDC authentication.
	Audience string `json:"audience,omitempty"`
	// Scope specifies the scope of the token in OIDC authentication.
	Scope string `json:"scope,omitempty"`
	// TokenEndpointURL specifies the URL which implements OIDC Token Endpoint.
	// It will be used to get an OIDC token.
	TokenEndpointURL string `json:"tokenEndpointURL,omitempty"`
	// AdditionalEndpointParams specifies additional parameters to be sent
	// this field will be transfer to map[string][]string in OIDC token generator.
	AdditionalEndpointParams map[string]string `json:"additionalEndpointParams,omitempty"`
}

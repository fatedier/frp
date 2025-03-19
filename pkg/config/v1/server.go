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
	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/config/types"
	"github.com/fatedier/frp/pkg/util/util"
)

type ServerConfig struct {
	APIMetadata

	Auth AuthServerConfig `json:"auth,omitempty"`
	// BindAddr specifies the address that the server binds to. By default,
	// this value is "0.0.0.0".
	BindAddr string `json:"bindAddr,omitempty"`
	// BindPort specifies the port that the server listens on. By default, this
	// value is 7000.
	BindPort int `json:"bindPort,omitempty"`
	// KCPBindPort specifies the KCP port that the server listens on. If this
	// value is 0, the server will not listen for KCP connections.
	KCPBindPort int `json:"kcpBindPort,omitempty"`
	// QUICBindPort specifies the QUIC port that the server listens on.
	// Set this value to 0 will disable this feature.
	QUICBindPort int `json:"quicBindPort,omitempty"`
	// ProxyBindAddr specifies the address that the proxy binds to. This value
	// may be the same as BindAddr.
	ProxyBindAddr string `json:"proxyBindAddr,omitempty"`
	// VhostHTTPPort specifies the port that the server listens for HTTP Vhost
	// requests. If this value is 0, the server will not listen for HTTP
	// requests.
	VhostHTTPPort int `json:"vhostHTTPPort,omitempty"`
	// VhostHTTPTimeout specifies the response header timeout for the Vhost
	// HTTP server, in seconds. By default, this value is 60.
	VhostHTTPTimeout int64 `json:"vhostHTTPTimeout,omitempty"`
	// VhostHTTPSPort specifies the port that the server listens for HTTPS
	// Vhost requests. If this value is 0, the server will not listen for HTTPS
	// requests.
	VhostHTTPSPort int `json:"vhostHTTPSPort,omitempty"`
	// TCPMuxHTTPConnectPort specifies the port that the server listens for TCP
	// HTTP CONNECT requests. If the value is 0, the server will not multiplex TCP
	// requests on one single port. If it's not - it will listen on this value for
	// HTTP CONNECT requests.
	TCPMuxHTTPConnectPort int `json:"tcpmuxHTTPConnectPort,omitempty"`
	// If TCPMuxPassthrough is true, frps won't do any update on traffic.
	TCPMuxPassthrough bool `json:"tcpmuxPassthrough,omitempty"`
	// SubDomainHost specifies the domain that will be attached to sub-domains
	// requested by the client when using Vhost proxying. For example, if this
	// value is set to "frps.com" and the client requested the subdomain
	// "test", the resulting URL would be "test.frps.com".
	SubDomainHost string `json:"subDomainHost,omitempty"`
	// Custom404Page specifies a path to a custom 404 page to display. If this
	// value is "", a default page will be displayed.
	Custom404Page string `json:"custom404Page,omitempty"`

	SSHTunnelGateway SSHTunnelGateway `json:"sshTunnelGateway,omitempty"`

	WebServer WebServerConfig `json:"webServer,omitempty"`
	// EnablePrometheus will export prometheus metrics on webserver address
	// in /metrics api.
	EnablePrometheus bool `json:"enablePrometheus,omitempty"`

	Log LogConfig `json:"log,omitempty"`

	Transport ServerTransportConfig `json:"transport,omitempty"`

	// DetailedErrorsToClient defines whether to send the specific error (with
	// debug info) to frpc. By default, this value is true.
	DetailedErrorsToClient *bool `json:"detailedErrorsToClient,omitempty"`
	// MaxPortsPerClient specifies the maximum number of ports a single client
	// may proxy to. If this value is 0, no limit will be applied.
	MaxPortsPerClient int64 `json:"maxPortsPerClient,omitempty"`
	// UserConnTimeout specifies the maximum time to wait for a work
	// connection. By default, this value is 10.
	UserConnTimeout int64 `json:"userConnTimeout,omitempty"`
	// UDPPacketSize specifies the UDP packet size
	// By default, this value is 1500
	UDPPacketSize int64 `json:"udpPacketSize,omitempty"`
	// NatHoleAnalysisDataReserveHours specifies the hours to reserve nat hole analysis data.
	NatHoleAnalysisDataReserveHours int64 `json:"natholeAnalysisDataReserveHours,omitempty"`

	AllowPorts []types.PortsRange `json:"allowPorts,omitempty"`

	HTTPPlugins []HTTPPluginOptions `json:"httpPlugins,omitempty"`
}

func (c *ServerConfig) Complete() {
	c.Auth.Complete()
	c.Log.Complete()
	c.Transport.Complete()
	c.WebServer.Complete()
	c.SSHTunnelGateway.Complete()

	c.BindAddr = util.EmptyOr(c.BindAddr, "0.0.0.0")
	c.BindPort = util.EmptyOr(c.BindPort, 7000)
	if c.ProxyBindAddr == "" {
		c.ProxyBindAddr = c.BindAddr
	}

	if c.WebServer.Port > 0 {
		c.WebServer.Addr = util.EmptyOr(c.WebServer.Addr, "0.0.0.0")
	}

	c.VhostHTTPTimeout = util.EmptyOr(c.VhostHTTPTimeout, 60)
	c.DetailedErrorsToClient = util.EmptyOr(c.DetailedErrorsToClient, lo.ToPtr(true))
	c.UserConnTimeout = util.EmptyOr(c.UserConnTimeout, 10)
	c.UDPPacketSize = util.EmptyOr(c.UDPPacketSize, 1500)
	c.NatHoleAnalysisDataReserveHours = util.EmptyOr(c.NatHoleAnalysisDataReserveHours, 7*24)
}

type AuthServerConfig struct {
	Method           AuthMethod           `json:"method,omitempty"`
	AdditionalScopes []AuthScope          `json:"additionalScopes,omitempty"`
	Token            string               `json:"token,omitempty"`
	OIDC             AuthOIDCServerConfig `json:"oidc,omitempty"`
}

func (c *AuthServerConfig) Complete() {
	c.Method = util.EmptyOr(c.Method, "token")
}

type AuthOIDCServerConfig struct {
	// Issuer specifies the issuer to verify OIDC tokens with. This issuer
	// will be used to load public keys to verify signature and will be compared
	// with the issuer claim in the OIDC token.
	Issuer string `json:"issuer,omitempty"`
	// Audience specifies the audience OIDC tokens should contain when validated.
	// If this value is empty, audience ("client ID") verification will be skipped.
	Audience string `json:"audience,omitempty"`
	// SkipExpiryCheck specifies whether to skip checking if the OIDC token is
	// expired.
	SkipExpiryCheck bool `json:"skipExpiryCheck,omitempty"`
	// SkipIssuerCheck specifies whether to skip checking if the OIDC token's
	// issuer claim matches the issuer specified in OidcIssuer.
	SkipIssuerCheck bool `json:"skipIssuerCheck,omitempty"`
}

type ServerTransportConfig struct {
	// TCPMux toggles TCP stream multiplexing. This allows multiple requests
	// from a client to share a single TCP connection. By default, this value
	// is true.
	// $HideFromDoc
	TCPMux *bool `json:"tcpMux,omitempty"`
	// TCPMuxKeepaliveInterval specifies the keep alive interval for TCP stream multiplier.
	// If TCPMux is true, heartbeat of application layer is unnecessary because it can only rely on heartbeat in TCPMux.
	TCPMuxKeepaliveInterval int64 `json:"tcpMuxKeepaliveInterval,omitempty"`
	// TCPKeepAlive specifies the interval between keep-alive probes for an active network connection between frpc and frps.
	// If negative, keep-alive probes are disabled.
	TCPKeepAlive int64 `json:"tcpKeepalive,omitempty"`
	// MaxPoolCount specifies the maximum pool size for each proxy. By default,
	// this value is 5.
	MaxPoolCount int64 `json:"maxPoolCount,omitempty"`
	// HeartBeatTimeout specifies the maximum time to wait for a heartbeat
	// before terminating the connection. It is not recommended to change this
	// value. By default, this value is 90. Set negative value to disable it.
	HeartbeatTimeout int64 `json:"heartbeatTimeout,omitempty"`
	// QUIC options.
	QUIC QUICOptions `json:"quic,omitempty"`
	// TLS specifies TLS settings for the connection from the client.
	TLS TLSServerConfig `json:"tls,omitempty"`
}

func (c *ServerTransportConfig) Complete() {
	c.TCPMux = util.EmptyOr(c.TCPMux, lo.ToPtr(true))
	c.TCPMuxKeepaliveInterval = util.EmptyOr(c.TCPMuxKeepaliveInterval, 30)
	c.TCPKeepAlive = util.EmptyOr(c.TCPKeepAlive, 7200)
	c.MaxPoolCount = util.EmptyOr(c.MaxPoolCount, 5)
	if lo.FromPtr(c.TCPMux) {
		// If TCPMux is enabled, heartbeat of application layer is unnecessary because we can rely on heartbeat in tcpmux.
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, -1)
	} else {
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, 90)
	}
	c.QUIC.Complete()
	if c.TLS.TrustedCaFile != "" {
		c.TLS.Force = true
	}
}

type TLSServerConfig struct {
	// Force specifies whether to only accept TLS-encrypted connections.
	Force bool `json:"force,omitempty"`

	TLSConfig
}

type SSHTunnelGateway struct {
	BindPort              int    `json:"bindPort,omitempty"`
	PrivateKeyFile        string `json:"privateKeyFile,omitempty"`
	AutoGenPrivateKeyPath string `json:"autoGenPrivateKeyPath,omitempty"`
	AuthorizedKeysFile    string `json:"authorizedKeysFile,omitempty"`
}

func (c *SSHTunnelGateway) Complete() {
	c.AutoGenPrivateKeyPath = util.EmptyOr(c.AutoGenPrivateKeyPath, "./.autogen_ssh_key")
}

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
	"maps"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/util/util"
)

type AuthScope string

const (
	AuthScopeHeartBeats   AuthScope = "HeartBeats"
	AuthScopeNewWorkConns AuthScope = "NewWorkConns"
)

type AuthMethod string

const (
	AuthMethodToken AuthMethod = "token"
	AuthMethodOIDC  AuthMethod = "oidc"
)

// QUIC protocol options
type QUICOptions struct {
	KeepalivePeriod    int `json:"keepalivePeriod,omitempty"`
	MaxIdleTimeout     int `json:"maxIdleTimeout,omitempty"`
	MaxIncomingStreams int `json:"maxIncomingStreams,omitempty"`
}

func (c *QUICOptions) Complete() {
	c.KeepalivePeriod = util.EmptyOr(c.KeepalivePeriod, 10)
	c.MaxIdleTimeout = util.EmptyOr(c.MaxIdleTimeout, 30)
	c.MaxIncomingStreams = util.EmptyOr(c.MaxIncomingStreams, 100000)
}

const (
	TransportProtocolTCP       = "tcp"
	TransportProtocolKCP       = "kcp"
	TransportProtocolQUIC      = "quic"
	TransportProtocolWebsocket = "websocket"
	TransportProtocolWSS       = "wss"
	TransportProtocolAuto      = "auto"

	AutoTransportStrategyBalanced  = "balanced"
	AutoTransportStrategyLatency   = "latency"
	AutoTransportStrategyStability = "stability"
)

var DefaultAutoTransportCandidates = []string{
	TransportProtocolQUIC,
	TransportProtocolTCP,
	TransportProtocolWSS,
	TransportProtocolWebsocket,
	TransportProtocolKCP,
}

var SupportedAutoTransportStrategies = []string{
	AutoTransportStrategyBalanced,
	AutoTransportStrategyLatency,
	AutoTransportStrategyStability,
}

type ClientAutoTransportConfig struct {
	Enabled            *bool    `json:"enabled,omitempty"`
	Candidates         []string `json:"candidates,omitempty"`
	AllowUDP           *bool    `json:"allowUDP,omitempty"`
	Strategy           string   `json:"strategy,omitempty"`
	ProbeTimeoutMs     int      `json:"probeTimeoutMs,omitempty"`
	ProbeCount         int      `json:"probeCount,omitempty"`
	StickyDurationSec  int      `json:"stickyDurationSec,omitempty"`
	CooldownSec        int      `json:"cooldownSec,omitempty"`
	FailureThreshold   int      `json:"failureThreshold,omitempty"`
	DegradeThreshold   int      `json:"degradeThreshold,omitempty"`
	RecheckIntervalSec int      `json:"recheckIntervalSec,omitempty"`
	PersistLastGood    *bool    `json:"persistLastGood,omitempty"`
	BootstrapProtocol  string   `json:"bootstrapProtocol,omitempty"`
	BootstrapPort      int      `json:"bootstrapPort,omitempty"`
}

func (c *ClientAutoTransportConfig) Complete(protocol string, serverPort int) {
	if protocol == TransportProtocolAuto && c.Enabled == nil {
		c.Enabled = util.EmptyOr(c.Enabled, lo.ToPtr(true))
	}
	if len(c.Candidates) == 0 {
		c.Candidates = append([]string(nil), DefaultAutoTransportCandidates...)
	}
	c.AllowUDP = util.EmptyOr(c.AllowUDP, lo.ToPtr(true))
	c.Strategy = util.EmptyOr(c.Strategy, AutoTransportStrategyBalanced)
	c.ProbeTimeoutMs = util.EmptyOr(c.ProbeTimeoutMs, 1200)
	c.ProbeCount = util.EmptyOr(c.ProbeCount, 2)
	c.StickyDurationSec = util.EmptyOr(c.StickyDurationSec, 1800)
	c.CooldownSec = util.EmptyOr(c.CooldownSec, 300)
	c.FailureThreshold = util.EmptyOr(c.FailureThreshold, 3)
	c.DegradeThreshold = util.EmptyOr(c.DegradeThreshold, 5)
	c.RecheckIntervalSec = util.EmptyOr(c.RecheckIntervalSec, 300)
	c.PersistLastGood = util.EmptyOr(c.PersistLastGood, lo.ToPtr(true))
	c.BootstrapProtocol = util.EmptyOr(c.BootstrapProtocol, TransportProtocolTCP)
	c.BootstrapPort = util.EmptyOr(c.BootstrapPort, serverPort)
}

type ServerAutoTransportConfig struct {
	Enabled            *bool    `json:"enabled,omitempty"`
	AllowDynamicSwitch *bool    `json:"allowDynamicSwitch,omitempty"`
	AdvertiseProtocols []string `json:"advertiseProtocols,omitempty"`
	PreferOrder        []string `json:"preferOrder,omitempty"`
	SwitchCooldownSec  int      `json:"switchCooldownSec,omitempty"`
}

func (c *ServerAutoTransportConfig) Complete(protocol string) {
	if protocol == TransportProtocolAuto {
		if c.Enabled == nil {
			c.Enabled = util.EmptyOr(c.Enabled, lo.ToPtr(true))
		}
		if c.AllowDynamicSwitch == nil {
			c.AllowDynamicSwitch = util.EmptyOr(c.AllowDynamicSwitch, lo.ToPtr(true))
		}
	}
	if len(c.AdvertiseProtocols) == 0 {
		c.AdvertiseProtocols = append([]string(nil), DefaultAutoTransportCandidates...)
	}
	if len(c.PreferOrder) == 0 {
		c.PreferOrder = append([]string(nil), DefaultAutoTransportCandidates...)
	}
	c.SwitchCooldownSec = util.EmptyOr(c.SwitchCooldownSec, 300)
}

type WebServerConfig struct {
	// This is the network address to bind on for serving the web interface and API.
	// By default, this value is "127.0.0.1".
	Addr string `json:"addr,omitempty"`
	// Port specifies the port for the web server to listen on. If this
	// value is 0, the admin server will not be started.
	Port int `json:"port,omitempty"`
	// User specifies the username that the web server will use for login.
	User string `json:"user,omitempty"`
	// Password specifies the password that the admin server will use for login.
	Password string `json:"password,omitempty"`
	// AssetsDir specifies the local directory that the admin server will load
	// resources from. If this value is "", assets will be loaded from the
	// bundled executable using embed package.
	AssetsDir string `json:"assetsDir,omitempty"`
	// Enable golang pprof handlers.
	PprofEnable bool `json:"pprofEnable,omitempty"`
	// Enable TLS if TLSConfig is not nil.
	TLS *TLSConfig `json:"tls,omitempty"`
}

func (c *WebServerConfig) Complete() {
	c.Addr = util.EmptyOr(c.Addr, "127.0.0.1")
}

type TLSConfig struct {
	// CertFile specifies the path of the cert file that client will load.
	CertFile string `json:"certFile,omitempty"`
	// KeyFile specifies the path of the secret key file that client will load.
	KeyFile string `json:"keyFile,omitempty"`
	// TrustedCaFile specifies the path of the trusted ca file that will load.
	TrustedCaFile string `json:"trustedCaFile,omitempty"`
	// ServerName specifies the custom server name of tls certificate. By
	// default, server name if same to ServerAddr.
	ServerName string `json:"serverName,omitempty"`
}

// NatTraversalConfig defines configuration options for NAT traversal
type NatTraversalConfig struct {
	// DisableAssistedAddrs disables the use of local network interfaces
	// for assisted connections during NAT traversal. When enabled,
	// only STUN-discovered public addresses will be used.
	DisableAssistedAddrs bool `json:"disableAssistedAddrs,omitempty"`
}

func (c *NatTraversalConfig) Clone() *NatTraversalConfig {
	if c == nil {
		return nil
	}
	out := *c
	return &out
}

type LogConfig struct {
	// This is destination where frp should write the logs.
	// If "console" is used, logs will be printed to stdout, otherwise,
	// logs will be written to the specified file.
	// By default, this value is "console".
	To string `json:"to,omitempty"`
	// Level specifies the minimum log level. Valid values are "trace",
	// "debug", "info", "warn", and "error". By default, this value is "info".
	Level string `json:"level,omitempty"`
	// MaxDays specifies the maximum number of days to store log information
	// before deletion.
	MaxDays int64 `json:"maxDays"`
	// DisablePrintColor disables log colors when log.to is "console".
	DisablePrintColor bool `json:"disablePrintColor,omitempty"`
}

func (c *LogConfig) Complete() {
	c.To = util.EmptyOr(c.To, "console")
	c.Level = util.EmptyOr(c.Level, "info")
	c.MaxDays = util.EmptyOr(c.MaxDays, 3)
}

type HTTPPluginOptions struct {
	Name      string   `json:"name"`
	Addr      string   `json:"addr"`
	Path      string   `json:"path"`
	Ops       []string `json:"ops"`
	TLSVerify bool     `json:"tlsVerify,omitempty"`
}

type HeaderOperations struct {
	Set map[string]string `json:"set,omitempty"`
}

func (o HeaderOperations) Clone() HeaderOperations {
	return HeaderOperations{
		Set: maps.Clone(o.Set),
	}
}

type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

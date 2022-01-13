// Copyright 2020 The frp Authors
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

package config

import (
	"fmt"
	"strings"

	"github.com/fatedier/frp/pkg/auth"
	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/pkg/util/util"

	"github.com/go-playground/validator/v10"
	"gopkg.in/ini.v1"
)

// ServerCommonConf contains information for a server service. It is
// recommended to use GetDefaultServerConf instead of creating this object
// directly, so that all unspecified fields have reasonable default values.
type ServerCommonConf struct {
	auth.ServerConfig `ini:",extends"`

	// BindAddr specifies the address that the server binds to. By default,
	// this value is "0.0.0.0".
	BindAddr string `ini:"bind_addr" json:"bind_addr"`
	// BindPort specifies the port that the server listens on. By default, this
	// value is 7000.
	BindPort int `ini:"bind_port" json:"bind_port" validate:"gte=0,lte=65535"`
	// BindUDPPort specifies the UDP port that the server listens on. If this
	// value is 0, the server will not listen for UDP connections. By default,
	// this value is 0
	BindUDPPort int `ini:"bind_udp_port" json:"bind_udp_port" validate:"gte=0,lte=65535"`
	// KCPBindPort specifies the KCP port that the server listens on. If this
	// value is 0, the server will not listen for KCP connections. By default,
	// this value is 0.
	KCPBindPort int `ini:"kcp_bind_port" json:"kcp_bind_port" validate:"gte=0,lte=65535"`
	// ProxyBindAddr specifies the address that the proxy binds to. This value
	// may be the same as BindAddr.
	ProxyBindAddr string `ini:"proxy_bind_addr" json:"proxy_bind_addr"`
	// VhostHTTPPort specifies the port that the server listens for HTTP Vhost
	// requests. If this value is 0, the server will not listen for HTTP
	// requests. By default, this value is 0.
	VhostHTTPPort int `ini:"vhost_http_port" json:"vhost_http_port" validate:"gte=0,lte=65535"`
	// VhostHTTPSPort specifies the port that the server listens for HTTPS
	// Vhost requests. If this value is 0, the server will not listen for HTTPS
	// requests. By default, this value is 0.
	VhostHTTPSPort int `ini:"vhost_https_port" json:"vhost_https_port" validate:"gte=0,lte=65535"`
	// TCPMuxHTTPConnectPort specifies the port that the server listens for TCP
	// HTTP CONNECT requests. If the value is 0, the server will not multiplex TCP
	// requests on one single port. If it's not - it will listen on this value for
	// HTTP CONNECT requests. By default, this value is 0.
	TCPMuxHTTPConnectPort int `ini:"tcpmux_httpconnect_port" json:"tcpmux_httpconnect_port" validate:"gte=0,lte=65535"`
	// VhostHTTPTimeout specifies the response header timeout for the Vhost
	// HTTP server, in seconds. By default, this value is 60.
	VhostHTTPTimeout int64 `ini:"vhost_http_timeout" json:"vhost_http_timeout"`
	// DashboardAddr specifies the address that the dashboard binds to. By
	// default, this value is "0.0.0.0".
	DashboardAddr string `ini:"dashboard_addr" json:"dashboard_addr"`
	// DashboardPort specifies the port that the dashboard listens on. If this
	// value is 0, the dashboard will not be started. By default, this value is
	// 0.
	DashboardPort int `ini:"dashboard_port" json:"dashboard_port" validate:"gte=0,lte=65535"`
	// DashboardUser specifies the username that the dashboard will use for
	// login.
	DashboardUser string `ini:"dashboard_user" json:"dashboard_user"`
	// DashboardPwd specifies the password that the dashboard will use for
	// login.
	DashboardPwd string `ini:"dashboard_pwd" json:"dashboard_pwd"`
	// EnablePrometheus will export prometheus metrics on {dashboard_addr}:{dashboard_port}
	// in /metrics api.
	EnablePrometheus bool `ini:"enable_prometheus" json:"enable_prometheus"`
	// AssetsDir specifies the local directory that the dashboard will load
	// resources from. If this value is "", assets will be loaded from the
	// bundled executable using statik. By default, this value is "".
	AssetsDir string `ini:"assets_dir" json:"assets_dir"`
	// LogFile specifies a file where logs will be written to. This value will
	// only be used if LogWay is set appropriately. By default, this value is
	// "console".
	LogFile string `ini:"log_file" json:"log_file"`
	// LogWay specifies the way logging is managed. Valid values are "console"
	// or "file". If "console" is used, logs will be printed to stdout. If
	// "file" is used, logs will be printed to LogFile. By default, this value
	// is "console".
	LogWay string `ini:"log_way" json:"log_way"`
	// LogLevel specifies the minimum log level. Valid values are "trace",
	// "debug", "info", "warn", and "error". By default, this value is "info".
	LogLevel string `ini:"log_level" json:"log_level"`
	// LogMaxDays specifies the maximum number of days to store log information
	// before deletion. This is only used if LogWay == "file". By default, this
	// value is 0.
	LogMaxDays int64 `ini:"log_max_days" json:"log_max_days"`
	// DisableLogColor disables log colors when LogWay == "console" when set to
	// true. By default, this value is false.
	DisableLogColor bool `ini:"disable_log_color" json:"disable_log_color"`
	// DetailedErrorsToClient defines whether to send the specific error (with
	// debug info) to frpc. By default, this value is true.
	DetailedErrorsToClient bool `ini:"detailed_errors_to_client" json:"detailed_errors_to_client"`

	// SubDomainHost specifies the domain that will be attached to sub-domains
	// requested by the client when using Vhost proxying. For example, if this
	// value is set to "frps.com" and the client requested the subdomain
	// "test", the resulting URL would be "test.frps.com". By default, this
	// value is "".
	SubDomainHost string `ini:"subdomain_host" json:"subdomain_host"`
	// TCPMux toggles TCP stream multiplexing. This allows multiple requests
	// from a client to share a single TCP connection. By default, this value
	// is true.
	TCPMux bool `ini:"tcp_mux" json:"tcp_mux"`
	// TCPMuxKeepaliveInterval specifies the keep alive interval for TCP stream multipler.
	// If TCPMux is true, heartbeat of application layer is unnecessary because it can only rely on heartbeat in TCPMux.
	TCPMuxKeepaliveInterval int64 `ini:"tcp_mux_keepalive_interval" json:"tcp_mux_keepalive_interval"`
	// Custom404Page specifies a path to a custom 404 page to display. If this
	// value is "", a default page will be displayed. By default, this value is
	// "".
	Custom404Page string `ini:"custom_404_page" json:"custom_404_page"`

	// AllowPorts specifies a set of ports that clients are able to proxy to.
	// If the length of this value is 0, all ports are allowed. By default,
	// this value is an empty set.
	AllowPorts map[int]struct{} `ini:"-" json:"-"`
	// MaxPoolCount specifies the maximum pool size for each proxy. By default,
	// this value is 5.
	MaxPoolCount int64 `ini:"max_pool_count" json:"max_pool_count"`
	// MaxPortsPerClient specifies the maximum number of ports a single client
	// may proxy to. If this value is 0, no limit will be applied. By default,
	// this value is 0.
	MaxPortsPerClient int64 `ini:"max_ports_per_client" json:"max_ports_per_client"`
	// TLSOnly specifies whether to only accept TLS-encrypted connections.
	// By default, the value is false.
	TLSOnly bool `ini:"tls_only" json:"tls_only"`
	// TLSCertFile specifies the path of the cert file that the server will
	// load. If "tls_cert_file", "tls_key_file" are valid, the server will use this
	// supplied tls configuration. Otherwise, the server will use the tls
	// configuration generated by itself.
	TLSCertFile string `ini:"tls_cert_file" json:"tls_cert_file"`
	// TLSKeyFile specifies the path of the secret key that the server will
	// load. If "tls_cert_file", "tls_key_file" are valid, the server will use this
	// supplied tls configuration. Otherwise, the server will use the tls
	// configuration generated by itself.
	TLSKeyFile string `ini:"tls_key_file" json:"tls_key_file"`
	// TLSTrustedCaFile specifies the paths of the client cert files that the
	// server will load. It only works when "tls_only" is true. If
	// "tls_trusted_ca_file" is valid, the server will verify each client's
	// certificate.
	TLSTrustedCaFile string `ini:"tls_trusted_ca_file" json:"tls_trusted_ca_file"`
	// HeartBeatTimeout specifies the maximum time to wait for a heartbeat
	// before terminating the connection. It is not recommended to change this
	// value. By default, this value is 90. Set negative value to disable it.
	HeartbeatTimeout int64 `ini:"heartbeat_timeout" json:"heartbeat_timeout"`
	// UserConnTimeout specifies the maximum time to wait for a work
	// connection. By default, this value is 10.
	UserConnTimeout int64 `ini:"user_conn_timeout" json:"user_conn_timeout"`
	// HTTPPlugins specify the server plugins support HTTP protocol.
	HTTPPlugins map[string]plugin.HTTPPluginOptions `ini:"-" json:"http_plugins"`
	// UDPPacketSize specifies the UDP packet size
	// By default, this value is 1500
	UDPPacketSize int64 `ini:"udp_packet_size" json:"udp_packet_size"`
}

// GetDefaultServerConf returns a server configuration with reasonable
// defaults.
func GetDefaultServerConf() ServerCommonConf {
	return ServerCommonConf{
		ServerConfig:            auth.GetDefaultServerConf(),
		BindAddr:                "0.0.0.0",
		BindPort:                7000,
		BindUDPPort:             0,
		KCPBindPort:             0,
		ProxyBindAddr:           "",
		VhostHTTPPort:           0,
		VhostHTTPSPort:          0,
		TCPMuxHTTPConnectPort:   0,
		VhostHTTPTimeout:        60,
		DashboardAddr:           "0.0.0.0",
		DashboardPort:           0,
		DashboardUser:           "",
		DashboardPwd:            "",
		EnablePrometheus:        false,
		AssetsDir:               "",
		LogFile:                 "console",
		LogWay:                  "console",
		LogLevel:                "info",
		LogMaxDays:              3,
		DisableLogColor:         false,
		DetailedErrorsToClient:  true,
		SubDomainHost:           "",
		TCPMux:                  true,
		TCPMuxKeepaliveInterval: 60,
		AllowPorts:              make(map[int]struct{}),
		MaxPoolCount:            5,
		MaxPortsPerClient:       0,
		TLSOnly:                 false,
		TLSCertFile:             "",
		TLSKeyFile:              "",
		TLSTrustedCaFile:        "",
		HeartbeatTimeout:        90,
		UserConnTimeout:         10,
		Custom404Page:           "",
		HTTPPlugins:             make(map[string]plugin.HTTPPluginOptions),
		UDPPacketSize:           1500,
	}
}

func UnmarshalServerConfFromIni(source interface{}) (ServerCommonConf, error) {

	f, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:         false,
		InsensitiveSections: false,
		InsensitiveKeys:     false,
		IgnoreInlineComment: true,
		AllowBooleanKeys:    true,
	}, source)
	if err != nil {
		return ServerCommonConf{}, err
	}

	s, err := f.GetSection("common")
	if err != nil {
		return ServerCommonConf{}, err
	}

	common := GetDefaultServerConf()
	err = s.MapTo(&common)
	if err != nil {
		return ServerCommonConf{}, err
	}

	// allow_ports
	allowPortStr := s.Key("allow_ports").String()
	if allowPortStr != "" {
		allowPorts, err := util.ParseRangeNumbers(allowPortStr)
		if err != nil {
			return ServerCommonConf{}, fmt.Errorf("invalid allow_ports: %v", err)
		}
		for _, port := range allowPorts {
			common.AllowPorts[int(port)] = struct{}{}
		}
	}

	// plugin.xxx
	pluginOpts := make(map[string]plugin.HTTPPluginOptions)
	for _, section := range f.Sections() {
		name := section.Name()
		if !strings.HasPrefix(name, "plugin.") {
			continue
		}

		opt, err := loadHTTPPluginOpt(section)
		if err != nil {
			return ServerCommonConf{}, err
		}

		pluginOpts[opt.Name] = *opt
	}
	common.HTTPPlugins = pluginOpts

	return common, nil
}

func (cfg *ServerCommonConf) Complete() {
	if cfg.LogFile == "console" {
		cfg.LogWay = "console"
	} else {
		cfg.LogWay = "file"
	}

	if cfg.ProxyBindAddr == "" {
		cfg.ProxyBindAddr = cfg.BindAddr
	}

	if cfg.TLSTrustedCaFile != "" {
		cfg.TLSOnly = true
	}
}

func (cfg *ServerCommonConf) Validate() error {
	return validator.New().Struct(cfg)
}

func loadHTTPPluginOpt(section *ini.Section) (*plugin.HTTPPluginOptions, error) {
	name := strings.TrimSpace(strings.TrimPrefix(section.Name(), "plugin."))

	opt := new(plugin.HTTPPluginOptions)
	err := section.MapTo(opt)
	if err != nil {
		return nil, err
	}

	opt.Name = name

	return opt, nil
}

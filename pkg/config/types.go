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
	"reflect"

	"github.com/fatedier/frp/pkg/auth"
	"github.com/fatedier/frp/pkg/consts"
	"github.com/fatedier/frp/pkg/msg"
	plugin "github.com/fatedier/frp/pkg/plugin/server"

	"gopkg.in/ini.v1"
)

// ClientCommonConf contains information for a client service. It is
// recommended to use GetDefaultClientConf instead of creating this object
// directly, so that all unspecified fields have reasonable default values.
type ClientCommonConf struct {
	auth.ClientConfig `ini:",,,,extends",json:"inline"`
	// ServerAddr specifies the address of the server to connect to. By
	// default, this value is "0.0.0.0".
	ServerAddr string `ini:"server_addr",josn:"server_addr"`
	// ServerPort specifies the port to connect to the server on. By default,
	// this value is 7000.
	ServerPort int `ini:"server_port",json:"server_port"`
	// HTTPProxy specifies a proxy address to connect to the server through. If
	// this value is "", the server will be connected to directly. By default,
	// this value is read from the "http_proxy" environment variable.
	HTTPProxy string `ini:"http_proxy",json:"http_proxy"`
	// LogFile specifies a file where logs will be written to. This value will
	// only be used if LogWay is set appropriately. By default, this value is
	// "console".
	LogFile string `ini:"log_file",json:"log_file"`
	// LogWay specifies the way logging is managed. Valid values are "console"
	// or "file". If "console" is used, logs will be printed to stdout. If
	// "file" is used, logs will be printed to LogFile. By default, this value
	// is "console".
	LogWay string `ini:"log_way",json:"log_way"`
	// LogLevel specifies the minimum log level. Valid values are "trace",
	// "debug", "info", "warn", and "error". By default, this value is "info".
	LogLevel string `ini:"log_level",json:"log_level"`
	// LogMaxDays specifies the maximum number of days to store log information
	// before deletion. This is only used if LogWay == "file". By default, this
	// value is 0.
	LogMaxDays int64 `ini:"log_max_days",json:"log_max_days"`
	// DisableLogColor disables log colors when LogWay == "console" when set to
	// true. By default, this value is false.
	DisableLogColor bool `ini:"disable_log_color",json:"disable_log_color"`
	// AdminAddr specifies the address that the admin server binds to. By
	// default, this value is "127.0.0.1".
	AdminAddr string `ini:"admin_addr",json:"admin_addr"`
	// AdminPort specifies the port for the admin server to listen on. If this
	// value is 0, the admin server will not be started. By default, this value
	// is 0.
	AdminPort int `ini:"admin_port",json:"admin_port"`
	// AdminUser specifies the username that the admin server will use for
	// login. By default, this value is "admin".
	AdminUser string `ini:"admin_user",json:"admin_user"`
	// AdminPwd specifies the password that the admin server will use for
	// login. By default, this value is "admin".
	AdminPwd string `ini:"admin_pwd",json:"admin_pwd"`
	// AssetsDir specifies the local directory that the admin server will load
	// resources from. If this value is "", assets will be loaded from the
	// bundled executable using statik. By default, this value is "".
	AssetsDir string `ini:"assets_dir",json:"assets_dir"`
	// PoolCount specifies the number of connections the client will make to
	// the server in advance. By default, this value is 0.
	PoolCount int `ini:"pool_count",json:"pool_count"`
	// TCPMux toggles TCP stream multiplexing. This allows multiple requests
	// from a client to share a single TCP connection. If this value is true,
	// the server must have TCP multiplexing enabled as well. By default, this
	// value is true.
	TCPMux bool `ini:"tcp_mux",json:"tcp_mux"`
	// User specifies a prefix for proxy names to distinguish them from other
	// clients. If this value is not "", proxy names will automatically be
	// changed to "{user}.{proxy_name}". By default, this value is "".
	User string `ini:"user",json:"user"`
	// DNSServer specifies a DNS server address for FRPC to use. If this value
	// is "", the default DNS will be used. By default, this value is "".
	DNSServer string `ini:"dns_server",json:"dns_server"`
	// LoginFailExit controls whether or not the client should exit after a
	// failed login attempt. If false, the client will retry until a login
	// attempt succeeds. By default, this value is true.
	LoginFailExit bool `ini:"login_fail_exit",json:"login_fail_exit"`
	// Start specifies a set of enabled proxies by name. If this set is empty,
	// all supplied proxies are enabled. By default, this value is an empty
	// set.
	Start []string `ini:"start",json:"start"`
	//Start map[string]struct{} `json:"start"`
	// Protocol specifies the protocol to use when interacting with the server.
	// Valid values are "tcp", "kcp" and "websocket". By default, this value
	// is "tcp".
	Protocol string `ini:"protocol",json:"protocol"`
	// TLSEnable specifies whether or not TLS should be used when communicating
	// with the server. If "tls_cert_file" and "tls_key_file" are valid,
	// client will load the supplied tls configuration.
	TLSEnable bool `ini:"tls_enable",json:"tls_enable"`
	// ClientTLSCertPath specifies the path of the cert file that client will
	// load. It only works when "tls_enable" is true and "tls_key_file" is valid.
	TLSCertFile string `ini:"tls_cert_file",json:"tls_cert_file"`
	// ClientTLSKeyPath specifies the path of the secret key file that client
	// will load. It only works when "tls_enable" is true and "tls_cert_file"
	// are valid.
	TLSKeyFile string `ini:"tls_key_file",json:"tls_key_file"`
	// TrustedCaFile specifies the path of the trusted ca file that will load.
	// It only works when "tls_enable" is valid and tls configuration of server
	// has been specified.
	TLSTrustedCaFile string `ini:"tls_trusted_ca_file",json:"tls_trusted_ca_file"`
	// HeartBeatInterval specifies at what interval heartbeats are sent to the
	// server, in seconds. It is not recommended to change this value. By
	// default, this value is 30.
	HeartbeatInterval int64 `ini:"heartbeat_interval",json:"heartbeat_interval"`
	// HeartBeatTimeout specifies the maximum allowed heartbeat response delay
	// before the connection is terminated, in seconds. It is not recommended
	// to change this value. By default, this value is 90.
	HeartbeatTimeout int64 `ini:"heartbeat_timeout",json:"heartbeat_timeout"`
	// Client meta info
	Metas map[string]string `ini:"-",json:"metas"`
	// UDPPacketSize specifies the udp packet size
	// By default, this value is 1500
	UDPPacketSize int64 `ini:"udp_packet_size",json:"udp_packet_size"`
}

// ServerCommonConf contains information for a server service. It is
// recommended to use GetDefaultServerConf instead of creating this object
// directly, so that all unspecified fields have reasonable default values.
type ServerCommonConf struct {
	auth.ServerConfig `ini:",,,,extends",json:"inline"`
	// BindAddr specifies the address that the server binds to. By default,
	// this value is "0.0.0.0".
	BindAddr string `ini:"bind_addr",json:"bind_addr"`
	// BindPort specifies the port that the server listens on. By default, this
	// value is 7000.
	BindPort int `ini:"bind_port",json:"bind_port"`
	// BindUDPPort specifies the UDP port that the server listens on. If this
	// value is 0, the server will not listen for UDP connections. By default,
	// this value is 0
	BindUDPPort int `ini:"bind_udp_port",json:"bind_udp_port"`
	// KCPBindPort specifies the KCP port that the server listens on. If this
	// value is 0, the server will not listen for KCP connections. By default,
	// this value is 0.
	KCPBindPort int `ini:"kcp_bind_port",json:"kcp_bind_port"`
	// ProxyBindAddr specifies the address that the proxy binds to. This value
	// may be the same as BindAddr. By default, this value is "0.0.0.0".
	ProxyBindAddr string `ini:"proxy_bind_addr",json:"proxy_bind_addr"`
	// VhostHTTPPort specifies the port that the server listens for HTTP Vhost
	// requests. If this value is 0, the server will not listen for HTTP
	// requests. By default, this value is 0.
	VhostHTTPPort int `ini:"vhost_http_port",json:"vhost_http_port"`
	// VhostHTTPSPort specifies the port that the server listens for HTTPS
	// Vhost requests. If this value is 0, the server will not listen for HTTPS
	// requests. By default, this value is 0.
	VhostHTTPSPort int `ini:"vhost_https_port",json:"vhost_https_port"`
	// TCPMuxHTTPConnectPort specifies the port that the server listens for TCP
	// HTTP CONNECT requests. If the value is 0, the server will not multiplex TCP
	// requests on one single port. If it's not - it will listen on this value for
	// HTTP CONNECT requests. By default, this value is 0.
	TCPMuxHTTPConnectPort int `ini:"tcpmux_httpconnect_port",json:"tcpmux_httpconnect_port"`
	// VhostHTTPTimeout specifies the response header timeout for the Vhost
	// HTTP server, in seconds. By default, this value is 60.
	VhostHTTPTimeout int64 `ini:"vhost_http_timeout",json:"vhost_http_timeout"`
	// DashboardAddr specifies the address that the dashboard binds to. By
	// default, this value is "0.0.0.0".
	DashboardAddr string `ini:"dashboard_addr",json:"dashboard_addr"`
	// DashboardPort specifies the port that the dashboard listens on. If this
	// value is 0, the dashboard will not be started. By default, this value is
	// 0.
	DashboardPort int `ini:"dashboard_port",json:"dashboard_port"`
	// DashboardUser specifies the username that the dashboard will use for
	// login. By default, this value is "admin".
	DashboardUser string `ini:"dashboard_user",json:"dashboard_user"`
	// DashboardUser specifies the password that the dashboard will use for
	// login. By default, this value is "admin".
	DashboardPwd string `ini:"dashboard_pwd",json:"dashboard_pwd"`
	// EnablePrometheus will export prometheus metrics on {dashboard_addr}:{dashboard_port}
	// in /metrics api.
	EnablePrometheus bool `ini:"enable_prometheus",json:"enable_prometheus"`
	// AssetsDir specifies the local directory that the dashboard will load
	// resources from. If this value is "", assets will be loaded from the
	// bundled executable using statik. By default, this value is "".
	AssetsDir string `ini:"assets_dir",json:"assets_dir"`
	// LogFile specifies a file where logs will be written to. This value will
	// only be used if LogWay is set appropriately. By default, this value is
	// "console".
	LogFile string `ini:"log_file",json:"log_file"`
	// LogWay specifies the way logging is managed. Valid values are "console"
	// or "file". If "console" is used, logs will be printed to stdout. If
	// "file" is used, logs will be printed to LogFile. By default, this value
	// is "console".
	LogWay string `ini:"log_way",json:"log_way"`
	// LogLevel specifies the minimum log level. Valid values are "trace",
	// "debug", "info", "warn", and "error". By default, this value is "info".
	LogLevel string `ini:"log_level",json:"log_level"`
	// LogMaxDays specifies the maximum number of days to store log information
	// before deletion. This is only used if LogWay == "file". By default, this
	// value is 0.
	LogMaxDays int64 `ini:"log_max_days",json:"log_max_days"`
	// DisableLogColor disables log colors when LogWay == "console" when set to
	// true. By default, this value is false.
	DisableLogColor bool `ini:"disable_log_color",json:"disable_log_color"`
	// DetailedErrorsToClient defines whether to send the specific error (with
	// debug info) to frpc. By default, this value is true.
	DetailedErrorsToClient bool `ini:"detailed_errors_to_client",json:"detailed_errors_to_client"`

	// SubDomainHost specifies the domain that will be attached to sub-domains
	// requested by the client when using Vhost proxying. For example, if this
	// value is set to "frps.com" and the client requested the subdomain
	// "test", the resulting URL would be "test.frps.com". By default, this
	// value is "".
	SubDomainHost string `ini:"subdomain_host",json:"subdomain_host"`
	// TCPMux toggles TCP stream multiplexing. This allows multiple requests
	// from a client to share a single TCP connection. By default, this value
	// is true.
	TCPMux bool `ini:"tcp_mux",json:"tcp_mux"`
	// Custom404Page specifies a path to a custom 404 page to display. If this
	// value is "", a default page will be displayed. By default, this value is
	// "".
	Custom404Page string `ini:"custom_404_page",json:"custom_404_page"`

	// AllowPorts specifies a set of ports that clients are able to proxy to.
	// If the length of this value is 0, all ports are allowed. By default,
	// this value is an empty set.
	AllowPorts map[int]struct{} `ini:"-",json:"-"`
	// MaxPoolCount specifies the maximum pool size for each proxy. By default,
	// this value is 5.
	MaxPoolCount int64 `ini:"max_pool_count",json:"max_pool_count"`
	// MaxPortsPerClient specifies the maximum number of ports a single client
	// may proxy to. If this value is 0, no limit will be applied. By default,
	// this value is 0.
	MaxPortsPerClient int64 `ini:"max_ports_per_client",json:"max_ports_per_client"`
	// TLSOnly specifies whether to only accept TLS-encrypted connections.
	// By default, the value is false.
	TLSOnly bool `ini:"tls_only",json:"tls_only"`
	// TLSCertFile specifies the path of the cert file that the server will
	// load. If "tls_cert_file", "tls_key_file" are valid, the server will use this
	// supplied tls configuration. Otherwise, the server will use the tls
	// configuration generated by itself.
	TLSCertFile string `ini:"tls_cert_file",json:"tls_cert_file"`
	// TLSKeyFile specifies the path of the secret key that the server will
	// load. If "tls_cert_file", "tls_key_file" are valid, the server will use this
	// supplied tls configuration. Otherwise, the server will use the tls
	// configuration generated by itself.
	TLSKeyFile string `ini:"tls_key_file",json:"tls_key_file"`
	// TLSTrustedCaFile specifies the paths of the client cert files that the
	// server will load. It only works when "tls_only" is true. If
	// "tls_trusted_ca_file" is valid, the server will verify each client's
	// certificate.
	TLSTrustedCaFile string `ini:"tls_trusted_ca_file",json:"tls_trusted_ca_file"`
	// HeartBeatTimeout specifies the maximum time to wait for a heartbeat
	// before terminating the connection. It is not recommended to change this
	// value. By default, this value is 90.
	HeartbeatTimeout int64 `ini:"heartbeat_timeout",json:"heartbeat_timeout"`
	// UserConnTimeout specifies the maximum time to wait for a work
	// connection. By default, this value is 10.
	UserConnTimeout int64 `ini:"user_conn_timeout",json:"user_conn_timeout"`
	// HTTPPlugins specify the server plugins support HTTP protocol.
	HTTPPlugins map[string]plugin.HTTPPluginOptions `ini:"-",json:"http_plugins"`
	// UDPPacketSize specifies the UDP packet size
	// By default, this value is 1500
	UDPPacketSize int64 `ini:"udp_packet_size",json:"udp_packet_size"`
}

// Proxy
var (
	ProxyConfTypeMap = map[string]reflect.Type{
		consts.TCPProxy:    reflect.TypeOf(TCPProxyConf{}),
		consts.TCPMuxProxy: reflect.TypeOf(TCPMuxProxyConf{}),
		consts.UDPProxy:    reflect.TypeOf(UDPProxyConf{}),
		consts.HTTPProxy:   reflect.TypeOf(HTTPProxyConf{}),
		consts.HTTPSProxy:  reflect.TypeOf(HTTPSProxyConf{}),
		consts.STCPProxy:   reflect.TypeOf(STCPProxyConf{}),
		consts.XTCPProxy:   reflect.TypeOf(XTCPProxyConf{}),
		consts.SUDPProxy:   reflect.TypeOf(SUDPProxyConf{}),
	}
)

type ProxyConf interface {
	GetBaseInfo() *BaseProxyConf
	UnmarshalFromMsg(*msg.NewProxy)
	UnmarshalFromIni(string, string, *ini.Section) error
	MarshalToMsg(*msg.NewProxy)
	CheckForCli() error
	CheckForSvr(ServerCommonConf) error
	Compare(ProxyConf) bool
}

// LocalSvrConf configures what location the client will to, or what
// plugin will be used.
type LocalSvrConf struct {
	// LocalIP specifies the IP address or host name to to.
	LocalIP string `ini:"local_ip",json:"local_ip"`
	// LocalPort specifies the port to to.
	LocalPort int `ini:"local_port",json:"local_port"`

	// Plugin specifies what plugin should be used for ng. If this value
	// is set, the LocalIp and LocalPort values will be ignored. By default,
	// this value is "".
	Plugin string `ini:"plugin",json:"plugin"`
	// PluginParams specify parameters to be passed to the plugin, if one is
	// being used. By default, this value is an empty map.
	PluginParams map[string]string `ini:"-"`
}

// HealthCheckConf configures health checking. This can be useful for load
// balancing purposes to detect and remove proxies to failing services.
type HealthCheckConf struct {
	// HealthCheckType specifies what protocol to use for health checking.
	// Valid values include "tcp", "http", and "". If this value is "", health
	// checking will not be performed. By default, this value is "".
	//
	// If the type is "tcp", a connection will be attempted to the target
	// server. If a connection cannot be established, the health check fails.
	//
	// If the type is "http", a GET request will be made to the endpoint
	// specified by HealthCheckURL. If the response is not a 200, the health
	// check fails.
	HealthCheckType string `ini:"health_check_type",json:"health_check_type"` // tcp | http
	// HealthCheckTimeoutS specifies the number of seconds to wait for a health
	// check attempt to connect. If the timeout is reached, this counts as a
	// health check failure. By default, this value is 3.
	HealthCheckTimeoutS int `ini:"health_check_timeout_s",json:"health_check_timeout_s"`
	// HealthCheckMaxFailed specifies the number of allowed failures before the
	// is stopped. By default, this value is 1.
	HealthCheckMaxFailed int `ini:"health_check_max_failed",json:"health_check_max_failed"`
	// HealthCheckIntervalS specifies the time in seconds between health
	// checks. By default, this value is 10.
	HealthCheckIntervalS int `ini:"health_check_interval_s",json:"health_check_interval_s"`
	// HealthCheckURL specifies the address to send health checks to if the
	// health check type is "http".
	HealthCheckURL string `ini:"health_check_url",json:"health_check_interval_s"`
	// HealthCheckAddr specifies the address to connect to if the health check
	// type is "tcp".
	HealthCheckAddr string `ini:"-"`
}

// BaseProxyConf provides configuration info that is common to all types.
type BaseProxyConf struct {
	// ProxyName is the name of this
	ProxyName string `ini:"name",json:"name"`
	// ProxyType specifies the type of this  Valid values include "tcp",
	// "udp", "http", "https", "stcp", and "xtcp". By default, this value is
	// "tcp".
	ProxyType string `ini:"type",json:"type"`

	// UseEncryption controls whether or not communication with the server will
	// be encrypted. Encryption is done using the tokens supplied in the server
	// and client configuration. By default, this value is false.
	UseEncryption bool `ini:"use_encryption",json:"use_encryption"`
	// UseCompression controls whether or not communication with the server
	// will be compressed. By default, this value is false.
	UseCompression bool `ini:"use_compression",json:"use_compression"`
	// Group specifies which group the is a part of. The server will use
	// this information to load balance proxies in the same group. If the value
	// is "", this will not be in a group. By default, this value is "".
	Group string `ini:"group",json:"group"`
	// GroupKey specifies a group key, which should be the same among proxies
	// of the same group. By default, this value is "".
	GroupKey string `ini:"group_key",json:"group_key"`

	// ProxyProtocolVersion specifies which protocol version to use. Valid
	// values include "v1", "v2", and "". If the value is "", a protocol
	// version will be automatically selected. By default, this value is "".
	ProxyProtocolVersion string `ini:"proxy_protocol_version",json:"proxy_protocol_version"`

	// BandwidthLimit limit the bandwidth
	// 0 means no limit
	BandwidthLimit BandwidthQuantity `ini:"bandwidth_limit",json:"bandwidth_limit"`

	// meta info for each proxy
	Metas map[string]string `ini:"-",json:"metas"`

	// TODO: LocalSvrConf => LocalAppConf
	LocalSvrConf    `ini:",,,,extends",json:"inline"`
	HealthCheckConf `ini:",,,,extends",json:"inline"`
}

type DomainConf struct {
	CustomDomains []string `ini:"custom_domains",json:"custom_domains"`
	SubDomain     string   `ini:"subdomain",json:"subdomain"`
}

// HTTP
type HTTPProxyConf struct {
	BaseProxyConf `ini:",,,,extends",json:"inline"`
	HTTPProxySpec `ini:",,,,extends",json:"inline"`
}

type HTTPProxySpec struct {
	DomainConf        `ini:",,,,extends",json:"inline"`
	Locations         []string          `ini:"locations",json:"locations"`
	HTTPUser          string            `ini:"http_user",json:"http_user"`
	HTTPPwd           string            `ini:"http_pwd",json:"http_pwd"`
	HostHeaderRewrite string            `ini:"host_header_rewrite",json:"host_header_rewrite"`
	Headers           map[string]string `ini:"-",json:"headers"`
}

// HTTPS
type HTTPSProxyConf struct {
	BaseProxyConf  `ini:",,,,extends",json:"inline"`
	HTTPSProxySpec `ini:",,,,extends",json:"inline"`
}

type HTTPSProxySpec struct {
	DomainConf `ini:",,,,extends",json:"inline"`
}

// TCP
type TCPProxyConf struct {
	BaseProxyConf `ini:",,,,extends",json:"inline"`
	TCPProxySpec  `ini:",,,,extends",json:"inline"`
}

type TCPProxySpec struct {
	RemotePort int `ini:"remote_port",json:"remote_port"`
}

// TCPMux
type TCPMuxProxyConf struct {
	BaseProxyConf   `ini:",,,,extends",json:"inline"`
	TCPMuxProxySpec `ini:",,,,extends",json:"inline"`
}

type TCPMuxProxySpec struct {
	DomainConf  `ini:",,,,extends",json:"inline"`
	Multiplexer string `ini:"multiplexer"`
}

// STCP
type STCPProxyConf struct {
	BaseProxyConf `ini:",,,,extends",json:"inline"`
	STCPProxySpec `ini:",,,,extends",json:"inline"`
}

type STCPProxySpec struct {
	Role string `ini:"role",json:"role"`
	Sk   string `ini:"sk",json:"sk"`
}

// XTCP
type XTCPProxyConf struct {
	BaseProxyConf `ini:",,,,extends",json:"inline"`
	XTCPProxySpec `ini:",,,,extends",json:"inline"`
}

type XTCPProxySpec struct {
	Role string `ini:"role",json:"role"`
	Sk   string `ini:"sk",json:"sk"`
}

// UDP
type UDPProxyConf struct {
	BaseProxyConf `ini:",,,,extends",json:"inline"`
	UDPProxySpec  `ini:",,,,extends",json:"inline"`
}

type UDPProxySpec struct {
	RemotePort int `ini:"remote_port",json:"remote_port"`
}

// SUDP
type SUDPProxyConf struct {
	BaseProxyConf `ini:",,,,extends",json:"inline"`
	SUDPProxySpec `ini:",,,,extends",json:"inline"`
}

type SUDPProxySpec struct {
	Role string `ini:"role",json:"role"`
	Sk   string `ini:"sk",json:"sk"`
}

// Visitor
var (
	VisitorConfTypeMap = map[string]reflect.Type{
		consts.STCPProxy: reflect.TypeOf(STCPVisitorConf{}),
		consts.XTCPProxy: reflect.TypeOf(XTCPVisitorConf{}),
		consts.SUDPProxy: reflect.TypeOf(SUDPVisitorConf{}),
	}
)

type VisitorConf interface {
	GetBaseInfo() *BaseVisitorConf
	Compare(cmp VisitorConf) bool
	UnmarshalFromIni(prefix string, name string, section *ini.Section) error
	Check() error
}

type BaseVisitorConf struct {
	ProxyName      string `ini:"name",json:"name"`
	ProxyType      string `ini:"type",json:"type"`
	UseEncryption  bool   `ini:"use_encryption",json:"use_encryption"`
	UseCompression bool   `ini:"use_compression",json:"use_compression"`
	Role           string `ini:"role",json:"role"`
	Sk             string `ini:"sk",json:"sk"`
	ServerName     string `ini:"server_name",json:"server_name"`
	BindAddr       string `ini:"bind_addr",json:"bind_addr"`
	BindPort       int    `ini:"bind_port",json:"bind_port"`
}

type SUDPVisitorConf struct {
	BaseVisitorConf `ini:",,,,extends",json:"inline"`
}

type STCPVisitorConf struct {
	BaseVisitorConf `ini:",,,,extends",json:"inline"`
}

type XTCPVisitorConf struct {
	BaseVisitorConf `ini:",,,,extends",json:"inline"`
}

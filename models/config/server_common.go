// Copyright 2016 fatedier, fatedier@gmail.com
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
	"strconv"
	"strings"

	ini "github.com/vaughan0/go-ini"

	"github.com/fatedier/frp/models/auth"
	plugin "github.com/fatedier/frp/models/plugin/server"
	"github.com/fatedier/frp/utils/util"
)

// ServerCommonConf contains information for a server service. It is
// recommended to use GetDefaultServerConf instead of creating this object
// directly, so that all unspecified fields have reasonable default values.
type ServerCommonConf struct {
	auth.AuthServerConfig
	// BindAddr specifies the address that the server binds to. By default,
	// this value is "0.0.0.0".
	BindAddr string `json:"bind_addr"`
	// BindPort specifies the port that the server listens on. By default, this
	// value is 7000.
	BindPort int `json:"bind_port"`
	// BindUdpPort specifies the UDP port that the server listens on. If this
	// value is 0, the server will not listen for UDP connections. By default,
	// this value is 0
	BindUdpPort int `json:"bind_udp_port"`
	// BindKcpPort specifies the KCP port that the server listens on. If this
	// value is 0, the server will not listen for KCP connections. By default,
	// this value is 0.
	KcpBindPort int `json:"kcp_bind_port"`
	// ProxyBindAddr specifies the address that the proxy binds to. This value
	// may be the same as BindAddr. By default, this value is "0.0.0.0".
	ProxyBindAddr string `json:"proxy_bind_addr"`
	// VhostHttpPort specifies the port that the server listens for HTTP Vhost
	// requests. If this value is 0, the server will not listen for HTTP
	// requests. By default, this value is 0.
	VhostHttpPort int `json:"vhost_http_port"`
	// VhostHttpsPort specifies the port that the server listens for HTTPS
	// Vhost requests. If this value is 0, the server will not listen for HTTPS
	// requests. By default, this value is 0.
	VhostHttpsPort int `json:"vhost_https_port"`
	// TcpMuxHttpConnectPort specifies the port that the server listens for TCP
	// HTTP CONNECT requests. If the value is 0, the server will not multiplex TCP
	// requests on one single port. If it's not - it will listen on this value for
	// HTTP CONNECT requests. By default, this value is 0.
	TcpMuxHttpConnectPort int `json:"tcpmux_httpconnect_port"`
	// VhostHttpTimeout specifies the response header timeout for the Vhost
	// HTTP server, in seconds. By default, this value is 60.
	VhostHttpTimeout int64 `json:"vhost_http_timeout"`
	// DashboardAddr specifies the address that the dashboard binds to. By
	// default, this value is "0.0.0.0".
	DashboardAddr string `json:"dashboard_addr"`
	// DashboardPort specifies the port that the dashboard listens on. If this
	// value is 0, the dashboard will not be started. By default, this value is
	// 0.
	DashboardPort int `json:"dashboard_port"`
	// DashboardUser specifies the username that the dashboard will use for
	// login. By default, this value is "admin".
	DashboardUser string `json:"dashboard_user"`
	// DashboardUser specifies the password that the dashboard will use for
	// login. By default, this value is "admin".
	DashboardPwd string `json:"dashboard_pwd"`
	// EnablePrometheus will export prometheus metrics on {dashboard_addr}:{dashboard_port}
	// in /metrics api.
	EnablePrometheus bool `json:"enable_prometheus"`
	// AssetsDir specifies the local directory that the dashboard will load
	// resources from. If this value is "", assets will be loaded from the
	// bundled executable using statik. By default, this value is "".
	AssetsDir string `json:"asserts_dir"`
	// LogFile specifies a file where logs will be written to. This value will
	// only be used if LogWay is set appropriately. By default, this value is
	// "console".
	LogFile string `json:"log_file"`
	// LogWay specifies the way logging is managed. Valid values are "console"
	// or "file". If "console" is used, logs will be printed to stdout. If
	// "file" is used, logs will be printed to LogFile. By default, this value
	// is "console".
	LogWay string `json:"log_way"`
	// LogLevel specifies the minimum log level. Valid values are "trace",
	// "debug", "info", "warn", and "error". By default, this value is "info".
	LogLevel string `json:"log_level"`
	// LogMaxDays specifies the maximum number of days to store log information
	// before deletion. This is only used if LogWay == "file". By default, this
	// value is 0.
	LogMaxDays int64 `json:"log_max_days"`
	// DisableLogColor disables log colors when LogWay == "console" when set to
	// true. By default, this value is false.
	DisableLogColor bool `json:"disable_log_color"`
	// DetailedErrorsToClient defines whether to send the specific error (with
	// debug info) to frpc. By default, this value is true.
	DetailedErrorsToClient bool `json:"detailed_errors_to_client"`

	// SubDomainHost specifies the domain that will be attached to sub-domains
	// requested by the client when using Vhost proxying. For example, if this
	// value is set to "frps.com" and the client requested the subdomain
	// "test", the resulting URL would be "test.frps.com". By default, this
	// value is "".
	SubDomainHost string `json:"subdomain_host"`
	// TcpMux toggles TCP stream multiplexing. This allows multiple requests
	// from a client to share a single TCP connection. By default, this value
	// is true.
	TcpMux bool `json:"tcp_mux"`
	// Custom404Page specifies a path to a custom 404 page to display. If this
	// value is "", a default page will be displayed. By default, this value is
	// "".
	Custom404Page string `json:"custom_404_page"`

	// AllowPorts specifies a set of ports that clients are able to proxy to.
	// If the length of this value is 0, all ports are allowed. By default,
	// this value is an empty set.
	AllowPorts map[int]struct{}
	// MaxPoolCount specifies the maximum pool size for each proxy. By default,
	// this value is 5.
	MaxPoolCount int64 `json:"max_pool_count"`
	// MaxPortsPerClient specifies the maximum number of ports a single client
	// may proxy to. If this value is 0, no limit will be applied. By default,
	// this value is 0.
	MaxPortsPerClient int64 `json:"max_ports_per_client"`
	// TlsOnly specifies whether to only accept TLS-encrypted connections. By
	// default, the value is false.
	TlsOnly bool `json:"tls_only"`
	// HeartBeatTimeout specifies the maximum time to wait for a heartbeat
	// before terminating the connection. It is not recommended to change this
	// value. By default, this value is 90.
	HeartBeatTimeout int64 `json:"heart_beat_timeout"`
	// UserConnTimeout specifies the maximum time to wait for a work
	// connection. By default, this value is 10.
	UserConnTimeout int64 `json:"user_conn_timeout"`
	// HTTPPlugins specify the server plugins support HTTP protocol.
	HTTPPlugins map[string]plugin.HTTPPluginOptions `json:"http_plugins"`
}

// GetDefaultServerConf returns a server configuration with reasonable
// defaults.
func GetDefaultServerConf() ServerCommonConf {
	return ServerCommonConf{
		BindAddr:               "0.0.0.0",
		BindPort:               7000,
		BindUdpPort:            0,
		KcpBindPort:            0,
		ProxyBindAddr:          "0.0.0.0",
		VhostHttpPort:          0,
		VhostHttpsPort:         0,
		TcpMuxHttpConnectPort:  0,
		VhostHttpTimeout:       60,
		DashboardAddr:          "0.0.0.0",
		DashboardPort:          0,
		DashboardUser:          "admin",
		DashboardPwd:           "admin",
		EnablePrometheus:       false,
		AssetsDir:              "",
		LogFile:                "console",
		LogWay:                 "console",
		LogLevel:               "info",
		LogMaxDays:             3,
		DisableLogColor:        false,
		DetailedErrorsToClient: true,
		SubDomainHost:          "",
		TcpMux:                 true,
		AllowPorts:             make(map[int]struct{}),
		MaxPoolCount:           5,
		MaxPortsPerClient:      0,
		TlsOnly:                false,
		HeartBeatTimeout:       90,
		UserConnTimeout:        10,
		Custom404Page:          "",
		HTTPPlugins:            make(map[string]plugin.HTTPPluginOptions),
	}
}

// UnmarshalServerConfFromIni parses the contents of a server configuration ini
// file and returns the resulting server configuration.
func UnmarshalServerConfFromIni(content string) (cfg ServerCommonConf, err error) {
	cfg = GetDefaultServerConf()

	conf, err := ini.Load(strings.NewReader(content))
	if err != nil {
		err = fmt.Errorf("parse ini conf file error: %v", err)
		return ServerCommonConf{}, err
	}

	UnmarshalPluginsFromIni(conf, &cfg)

	cfg.AuthServerConfig = auth.UnmarshalAuthServerConfFromIni(conf)

	var (
		tmpStr string
		ok     bool
		v      int64
	)
	if tmpStr, ok = conf.Get("common", "bind_addr"); ok {
		cfg.BindAddr = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "bind_port"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			err = fmt.Errorf("Parse conf error: invalid bind_port")
			return
		} else {
			cfg.BindPort = int(v)
		}
	}

	if tmpStr, ok = conf.Get("common", "bind_udp_port"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			err = fmt.Errorf("Parse conf error: invalid bind_udp_port")
			return
		} else {
			cfg.BindUdpPort = int(v)
		}
	}

	if tmpStr, ok = conf.Get("common", "kcp_bind_port"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			err = fmt.Errorf("Parse conf error: invalid kcp_bind_port")
			return
		} else {
			cfg.KcpBindPort = int(v)
		}
	}

	if tmpStr, ok = conf.Get("common", "proxy_bind_addr"); ok {
		cfg.ProxyBindAddr = tmpStr
	} else {
		cfg.ProxyBindAddr = cfg.BindAddr
	}

	if tmpStr, ok = conf.Get("common", "vhost_http_port"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			err = fmt.Errorf("Parse conf error: invalid vhost_http_port")
			return
		} else {
			cfg.VhostHttpPort = int(v)
		}
	} else {
		cfg.VhostHttpPort = 0
	}

	if tmpStr, ok = conf.Get("common", "vhost_https_port"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			err = fmt.Errorf("Parse conf error: invalid vhost_https_port")
			return
		} else {
			cfg.VhostHttpsPort = int(v)
		}
	} else {
		cfg.VhostHttpsPort = 0
	}

	if tmpStr, ok = conf.Get("common", "tcpmux_httpconnect_port"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			err = fmt.Errorf("Parse conf error: invalid tcpmux_httpconnect_port")
			return
		} else {
			cfg.TcpMuxHttpConnectPort = int(v)
		}
	} else {
		cfg.TcpMuxHttpConnectPort = 0
	}

	if tmpStr, ok = conf.Get("common", "vhost_http_timeout"); ok {
		v, errRet := strconv.ParseInt(tmpStr, 10, 64)
		if errRet != nil || v < 0 {
			err = fmt.Errorf("Parse conf error: invalid vhost_http_timeout")
			return
		} else {
			cfg.VhostHttpTimeout = v
		}
	}

	if tmpStr, ok = conf.Get("common", "dashboard_addr"); ok {
		cfg.DashboardAddr = tmpStr
	} else {
		cfg.DashboardAddr = cfg.BindAddr
	}

	if tmpStr, ok = conf.Get("common", "dashboard_port"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			err = fmt.Errorf("Parse conf error: invalid dashboard_port")
			return
		} else {
			cfg.DashboardPort = int(v)
		}
	} else {
		cfg.DashboardPort = 0
	}

	if tmpStr, ok = conf.Get("common", "dashboard_user"); ok {
		cfg.DashboardUser = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "dashboard_pwd"); ok {
		cfg.DashboardPwd = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "enable_prometheus"); ok && tmpStr == "true" {
		cfg.EnablePrometheus = true
	}

	if tmpStr, ok = conf.Get("common", "assets_dir"); ok {
		cfg.AssetsDir = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "log_file"); ok {
		cfg.LogFile = tmpStr
		if cfg.LogFile == "console" {
			cfg.LogWay = "console"
		} else {
			cfg.LogWay = "file"
		}
	}

	if tmpStr, ok = conf.Get("common", "log_level"); ok {
		cfg.LogLevel = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "log_max_days"); ok {
		v, err = strconv.ParseInt(tmpStr, 10, 64)
		if err == nil {
			cfg.LogMaxDays = v
		}
	}

	if tmpStr, ok = conf.Get("common", "disable_log_color"); ok && tmpStr == "true" {
		cfg.DisableLogColor = true
	}

	if tmpStr, ok = conf.Get("common", "detailed_errors_to_client"); ok && tmpStr == "false" {
		cfg.DetailedErrorsToClient = false
	} else {
		cfg.DetailedErrorsToClient = true
	}

	if allowPortsStr, ok := conf.Get("common", "allow_ports"); ok {
		// e.g. 1000-2000,2001,2002,3000-4000
		ports, errRet := util.ParseRangeNumbers(allowPortsStr)
		if errRet != nil {
			err = fmt.Errorf("Parse conf error: allow_ports: %v", errRet)
			return
		}

		for _, port := range ports {
			cfg.AllowPorts[int(port)] = struct{}{}
		}
	}

	if tmpStr, ok = conf.Get("common", "max_pool_count"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			err = fmt.Errorf("Parse conf error: invalid max_pool_count")
			return
		} else {
			if v < 0 {
				err = fmt.Errorf("Parse conf error: invalid max_pool_count")
				return
			}
			cfg.MaxPoolCount = v
		}
	}

	if tmpStr, ok = conf.Get("common", "max_ports_per_client"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			err = fmt.Errorf("Parse conf error: invalid max_ports_per_client")
			return
		} else {
			if v < 0 {
				err = fmt.Errorf("Parse conf error: invalid max_ports_per_client")
				return
			}
			cfg.MaxPortsPerClient = v
		}
	}

	if tmpStr, ok = conf.Get("common", "subdomain_host"); ok {
		cfg.SubDomainHost = strings.ToLower(strings.TrimSpace(tmpStr))
	}

	if tmpStr, ok = conf.Get("common", "tcp_mux"); ok && tmpStr == "false" {
		cfg.TcpMux = false
	} else {
		cfg.TcpMux = true
	}

	if tmpStr, ok = conf.Get("common", "custom_404_page"); ok {
		cfg.Custom404Page = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "heartbeat_timeout"); ok {
		v, errRet := strconv.ParseInt(tmpStr, 10, 64)
		if errRet != nil {
			err = fmt.Errorf("Parse conf error: heartbeat_timeout is incorrect")
			return
		} else {
			cfg.HeartBeatTimeout = v
		}
	}

	if tmpStr, ok = conf.Get("common", "tls_only"); ok && tmpStr == "true" {
		cfg.TlsOnly = true
	} else {
		cfg.TlsOnly = false
	}
	return
}

func UnmarshalPluginsFromIni(sections ini.File, cfg *ServerCommonConf) {
	for name, section := range sections {
		if strings.HasPrefix(name, "plugin.") {
			name = strings.TrimSpace(strings.TrimPrefix(name, "plugin."))
			options := plugin.HTTPPluginOptions{
				Name: name,
				Addr: section["addr"],
				Path: section["path"],
				Ops:  strings.Split(section["ops"], ","),
			}
			for i, _ := range options.Ops {
				options.Ops[i] = strings.TrimSpace(options.Ops[i])
			}
			cfg.HTTPPlugins[name] = options
		}
	}
}

func (cfg *ServerCommonConf) Check() (err error) {
	return
}

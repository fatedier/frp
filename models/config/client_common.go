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
	"os"
	"strconv"
	"strings"

	ini "github.com/vaughan0/go-ini"

	"github.com/fatedier/frp/models/auth"
)

// ClientCommonConf contains information for a client service. It is
// recommended to use GetDefaultClientConf instead of creating this object
// directly, so that all unspecified fields have reasonable default values.
type ClientCommonConf struct {
	auth.AuthClientConfig
	// ServerAddr specifies the address of the server to connect to. By
	// default, this value is "0.0.0.0".
	ServerAddr string `json:"server_addr"`
	// ServerPort specifies the port to connect to the server on. By default,
	// this value is 7000.
	ServerPort int `json:"server_port"`
	// HttpProxy specifies a proxy address to connect to the server through. If
	// this value is "", the server will be connected to directly. By default,
	// this value is read from the "http_proxy" environment variable.
	HttpProxy string `json:"http_proxy"`
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
	// AdminAddr specifies the address that the admin server binds to. By
	// default, this value is "127.0.0.1".
	AdminAddr string `json:"admin_addr"`
	// AdminPort specifies the port for the admin server to listen on. If this
	// value is 0, the admin server will not be started. By default, this value
	// is 0.
	AdminPort int `json:"admin_port"`
	// AdminUser specifies the username that the admin server will use for
	// login. By default, this value is "admin".
	AdminUser string `json:"admin_user"`
	// AdminPwd specifies the password that the admin server will use for
	// login. By default, this value is "admin".
	AdminPwd string `json:"admin_pwd"`
	// AssetsDir specifies the local directory that the admin server will load
	// resources from. If this value is "", assets will be loaded from the
	// bundled executable using statik. By default, this value is "".
	AssetsDir string `json:"assets_dir"`
	// PoolCount specifies the number of connections the client will make to
	// the server in advance. By default, this value is 0.
	PoolCount int `json:"pool_count"`
	// TcpMux toggles TCP stream multiplexing. This allows multiple requests
	// from a client to share a single TCP connection. If this value is true,
	// the server must have TCP multiplexing enabled as well. By default, this
	// value is true.
	TcpMux bool `json:"tcp_mux"`
	// User specifies a prefix for proxy names to distinguish them from other
	// clients. If this value is not "", proxy names will automatically be
	// changed to "{user}.{proxy_name}". By default, this value is "".
	User string `json:"user"`
	// DnsServer specifies a DNS server address for FRPC to use. If this value
	// is "", the default DNS will be used. By default, this value is "".
	DnsServer string `json:"dns_server"`
	// LoginFailExit controls whether or not the client should exit after a
	// failed login attempt. If false, the client will retry until a login
	// attempt succeeds. By default, this value is true.
	LoginFailExit bool `json:"login_fail_exit"`
	// Start specifies a set of enabled proxies by name. If this set is empty,
	// all supplied proxies are enabled. By default, this value is an empty
	// set.
	Start map[string]struct{} `json:"start"`
	// Protocol specifies the protocol to use when interacting with the server.
	// Valid values are "tcp", "kcp", and "websocket". By default, this value
	// is "tcp".
	Protocol string `json:"protocol"`
	// TLSEnable specifies whether or not TLS should be used when communicating
	// with the server.
	TLSEnable bool `json:"tls_enable"`
	// HeartBeatInterval specifies at what interval heartbeats are sent to the
	// server, in seconds. It is not recommended to change this value. By
	// default, this value is 30.
	HeartBeatInterval int64 `json:"heartbeat_interval"`
	// HeartBeatTimeout specifies the maximum allowed heartbeat response delay
	// before the connection is terminated, in seconds. It is not recommended
	// to change this value. By default, this value is 90.
	HeartBeatTimeout int64 `json:"heartbeat_timeout"`
	// Client meta info
	Metas map[string]string `json:"metas"`
}

// GetDefaultClientConf returns a client configuration with default values.
func GetDefaultClientConf() ClientCommonConf {
	return ClientCommonConf{
		ServerAddr:        "0.0.0.0",
		ServerPort:        7000,
		HttpProxy:         os.Getenv("http_proxy"),
		LogFile:           "console",
		LogWay:            "console",
		LogLevel:          "info",
		LogMaxDays:        3,
		DisableLogColor:   false,
		AdminAddr:         "127.0.0.1",
		AdminPort:         0,
		AdminUser:         "",
		AdminPwd:          "",
		AssetsDir:         "",
		PoolCount:         1,
		TcpMux:            true,
		User:              "",
		DnsServer:         "",
		LoginFailExit:     true,
		Start:             make(map[string]struct{}),
		Protocol:          "tcp",
		TLSEnable:         false,
		HeartBeatInterval: 30,
		HeartBeatTimeout:  90,
		Metas:             make(map[string]string),
	}
}

func UnmarshalClientConfFromIni(content string) (cfg ClientCommonConf, err error) {
	cfg = GetDefaultClientConf()

	conf, err := ini.Load(strings.NewReader(content))
	if err != nil {
		return ClientCommonConf{}, fmt.Errorf("parse ini conf file error: %v", err)
	}

	cfg.AuthClientConfig = auth.UnmarshalAuthClientConfFromIni(conf)

	var (
		tmpStr string
		ok     bool
		v      int64
	)
	if tmpStr, ok = conf.Get("common", "server_addr"); ok {
		cfg.ServerAddr = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "server_port"); ok {
		v, err = strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			err = fmt.Errorf("Parse conf error: invalid server_port")
			return
		}
		cfg.ServerPort = int(v)
	}

	if tmpStr, ok = conf.Get("common", "disable_log_color"); ok && tmpStr == "true" {
		cfg.DisableLogColor = true
	}

	if tmpStr, ok = conf.Get("common", "http_proxy"); ok {
		cfg.HttpProxy = tmpStr
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
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err == nil {
			cfg.LogMaxDays = v
		}
	}

	if tmpStr, ok = conf.Get("common", "admin_addr"); ok {
		cfg.AdminAddr = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "admin_port"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err == nil {
			cfg.AdminPort = int(v)
		} else {
			err = fmt.Errorf("Parse conf error: invalid admin_port")
			return
		}
	}

	if tmpStr, ok = conf.Get("common", "admin_user"); ok {
		cfg.AdminUser = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "admin_pwd"); ok {
		cfg.AdminPwd = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "assets_dir"); ok {
		cfg.AssetsDir = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "pool_count"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err == nil {
			cfg.PoolCount = int(v)
		}
	}

	if tmpStr, ok = conf.Get("common", "tcp_mux"); ok && tmpStr == "false" {
		cfg.TcpMux = false
	} else {
		cfg.TcpMux = true
	}

	if tmpStr, ok = conf.Get("common", "user"); ok {
		cfg.User = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "dns_server"); ok {
		cfg.DnsServer = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "start"); ok {
		proxyNames := strings.Split(tmpStr, ",")
		for _, name := range proxyNames {
			cfg.Start[strings.TrimSpace(name)] = struct{}{}
		}
	}

	if tmpStr, ok = conf.Get("common", "login_fail_exit"); ok && tmpStr == "false" {
		cfg.LoginFailExit = false
	} else {
		cfg.LoginFailExit = true
	}

	if tmpStr, ok = conf.Get("common", "protocol"); ok {
		// Now it only support tcp and kcp and websocket.
		if tmpStr != "tcp" && tmpStr != "kcp" && tmpStr != "websocket" {
			err = fmt.Errorf("Parse conf error: invalid protocol")
			return
		}
		cfg.Protocol = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "tls_enable"); ok && tmpStr == "true" {
		cfg.TLSEnable = true
	} else {
		cfg.TLSEnable = false
	}

	if tmpStr, ok = conf.Get("common", "heartbeat_timeout"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			err = fmt.Errorf("Parse conf error: invalid heartbeat_timeout")
			return
		} else {
			cfg.HeartBeatTimeout = v
		}
	}

	if tmpStr, ok = conf.Get("common", "heartbeat_interval"); ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			err = fmt.Errorf("Parse conf error: invalid heartbeat_interval")
			return
		} else {
			cfg.HeartBeatInterval = v
		}
	}
	for k, v := range conf.Section("common") {
		if strings.HasPrefix(k, "meta_") {
			cfg.Metas[strings.TrimPrefix(k, "meta_")] = v
		}
	}
	return
}

func (cfg *ClientCommonConf) Check() (err error) {
	if cfg.HeartBeatInterval <= 0 {
		err = fmt.Errorf("Parse conf error: invalid heartbeat_interval")
		return
	}

	if cfg.HeartBeatTimeout < cfg.HeartBeatInterval {
		err = fmt.Errorf("Parse conf error: invalid heartbeat_timeout, heartbeat_timeout is less than heartbeat_interval")
		return
	}
	return
}

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

	"github.com/fatedier/frp/utils/util"
	ini "github.com/vaughan0/go-ini"
)

var ServerCommonCfg *ServerCommonConf

// common config
type ServerCommonConf struct {
	ConfigFile    string
	BindAddr      string
	BindPort      int64
	KcpBindPort   int64
	ProxyBindAddr string

	// If VhostHttpPort equals 0, don't listen a public port for http protocol.
	VhostHttpPort int64

	// if VhostHttpsPort equals 0, don't listen a public port for https protocol
	VhostHttpsPort int64

	// if DashboardPort equals 0, dashboard is not available
	DashboardPort  int64
	DashboardUser  string
	DashboardPwd   string
	AssetsDir      string
	LogFile        string
	LogWay         string // console or file
	LogLevel       string
	LogMaxDays     int64
	PrivilegeMode  bool
	PrivilegeToken string
	AuthTimeout    int64
	SubDomainHost  string
	TcpMux         bool

	// if PrivilegeAllowPorts is not nil, tcp proxies which remote port exist in this map can be connected
	PrivilegeAllowPorts [][2]int64
	MaxPoolCount        int64
	HeartBeatTimeout    int64
	UserConnTimeout     int64
}

func GetDefaultServerCommonConf() *ServerCommonConf {
	return &ServerCommonConf{
		ConfigFile:       "./frps.ini",
		BindAddr:         "0.0.0.0",
		BindPort:         7000,
		KcpBindPort:      0,
		ProxyBindAddr:    "0.0.0.0",
		VhostHttpPort:    0,
		VhostHttpsPort:   0,
		DashboardPort:    0,
		DashboardUser:    "admin",
		DashboardPwd:     "admin",
		AssetsDir:        "",
		LogFile:          "console",
		LogWay:           "console",
		LogLevel:         "info",
		LogMaxDays:       3,
		PrivilegeMode:    true,
		PrivilegeToken:   "",
		AuthTimeout:      900,
		SubDomainHost:    "",
		TcpMux:           true,
		MaxPoolCount:     5,
		HeartBeatTimeout: 90,
		UserConnTimeout:  10,
	}
}

// Load server common configure.
func LoadServerCommonConf(conf ini.File) (cfg *ServerCommonConf, err error) {
	var (
		tmpStr string
		ok     bool
		v      int64
	)
	cfg = GetDefaultServerCommonConf()

	tmpStr, ok = conf.Get("common", "bind_addr")
	if ok {
		cfg.BindAddr = tmpStr
	}

	tmpStr, ok = conf.Get("common", "bind_port")
	if ok {
		v, err = strconv.ParseInt(tmpStr, 10, 64)
		if err == nil {
			cfg.BindPort = v
		}
	}

	tmpStr, ok = conf.Get("common", "kcp_bind_port")
	if ok {
		v, err = strconv.ParseInt(tmpStr, 10, 64)
		if err == nil && v > 0 {
			cfg.KcpBindPort = v
		}
	}

	tmpStr, ok = conf.Get("common", "proxy_bind_addr")
	if ok {
		cfg.ProxyBindAddr = tmpStr
	} else {
		cfg.ProxyBindAddr = cfg.BindAddr
	}

	tmpStr, ok = conf.Get("common", "vhost_http_port")
	if ok {
		cfg.VhostHttpPort, err = strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			err = fmt.Errorf("Parse conf error: vhost_http_port is incorrect")
			return
		}
	} else {
		cfg.VhostHttpPort = 0
	}

	tmpStr, ok = conf.Get("common", "vhost_https_port")
	if ok {
		cfg.VhostHttpsPort, err = strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			err = fmt.Errorf("Parse conf error: vhost_https_port is incorrect")
			return
		}
	} else {
		cfg.VhostHttpsPort = 0
	}

	tmpStr, ok = conf.Get("common", "dashboard_port")
	if ok {
		cfg.DashboardPort, err = strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			err = fmt.Errorf("Parse conf error: dashboard_port is incorrect")
			return
		}
	} else {
		cfg.DashboardPort = 0
	}

	tmpStr, ok = conf.Get("common", "dashboard_user")
	if ok {
		cfg.DashboardUser = tmpStr
	}

	tmpStr, ok = conf.Get("common", "dashboard_pwd")
	if ok {
		cfg.DashboardPwd = tmpStr
	}

	tmpStr, ok = conf.Get("common", "assets_dir")
	if ok {
		cfg.AssetsDir = tmpStr
	}

	tmpStr, ok = conf.Get("common", "log_file")
	if ok {
		cfg.LogFile = tmpStr
		if cfg.LogFile == "console" {
			cfg.LogWay = "console"
		} else {
			cfg.LogWay = "file"
		}
	}

	tmpStr, ok = conf.Get("common", "log_level")
	if ok {
		cfg.LogLevel = tmpStr
	}

	tmpStr, ok = conf.Get("common", "log_max_days")
	if ok {
		v, err = strconv.ParseInt(tmpStr, 10, 64)
		if err == nil {
			cfg.LogMaxDays = v
		}
	}

	tmpStr, ok = conf.Get("common", "privilege_mode")
	if ok {
		if tmpStr == "true" {
			cfg.PrivilegeMode = true
		}
	}

	// PrivilegeMode configure
	if cfg.PrivilegeMode == true {
		cfg.PrivilegeToken, _ = conf.Get("common", "privilege_token")

		allowPortsStr, ok := conf.Get("common", "privilege_allow_ports")
		// TODO: check if conflicts exist in port ranges
		if ok {
			cfg.PrivilegeAllowPorts, err = util.GetPortRanges(allowPortsStr)
			if err != nil {
				err = fmt.Errorf("Parse conf error: privilege_allow_ports is incorrect, %v", err)
				return
			}
		}
	}

	tmpStr, ok = conf.Get("common", "max_pool_count")
	if ok {
		v, err = strconv.ParseInt(tmpStr, 10, 64)
		if err == nil && v >= 0 {
			cfg.MaxPoolCount = v
		}
	}

	tmpStr, ok = conf.Get("common", "authentication_timeout")
	if ok {
		v, errRet := strconv.ParseInt(tmpStr, 10, 64)
		if errRet != nil {
			err = fmt.Errorf("Parse conf error: authentication_timeout is incorrect")
			return
		} else {
			cfg.AuthTimeout = v
		}
	}

	tmpStr, ok = conf.Get("common", "subdomain_host")
	if ok {
		cfg.SubDomainHost = strings.ToLower(strings.TrimSpace(tmpStr))
	}

	tmpStr, ok = conf.Get("common", "tcp_mux")
	if ok && tmpStr == "false" {
		cfg.TcpMux = false
	} else {
		cfg.TcpMux = true
	}

	tmpStr, ok = conf.Get("common", "heartbeat_timeout")
	if ok {
		v, errRet := strconv.ParseInt(tmpStr, 10, 64)
		if errRet != nil {
			err = fmt.Errorf("Parse conf error: heartbeat_timeout is incorrect")
			return
		} else {
			cfg.HeartBeatTimeout = v
		}
	}
	return
}

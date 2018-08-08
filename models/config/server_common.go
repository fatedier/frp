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

	"github.com/fatedier/frp/utils/util"
)

var (
	// server global configure used for generate proxy conf used in frps
	proxyBindAddr  string
	subDomainHost  string
	vhostHttpPort  int
	vhostHttpsPort int
)

func InitServerCfg(cfg *ServerCommonConf) {
	proxyBindAddr = cfg.ProxyBindAddr
	subDomainHost = cfg.SubDomainHost
	vhostHttpPort = cfg.VhostHttpPort
	vhostHttpsPort = cfg.VhostHttpsPort
}

// common config
type ServerCommonConf struct {
	BindAddr      string `json:"bind_addr"`
	BindPort      int    `json:"bind_port"`
	BindUdpPort   int    `json:"bind_udp_port"`
	KcpBindPort   int    `json:"kcp_bind_port"`
	ProxyBindAddr string `json:"proxy_bind_addr"`

	// If VhostHttpPort equals 0, don't listen a public port for http protocol.
	VhostHttpPort int `json:"vhost_http_port"`

	// if VhostHttpsPort equals 0, don't listen a public port for https protocol
	VhostHttpsPort int `json:"vhost_http_port"`

	VhostHttpTimeout int64 `json:"vhost_http_timeout"`

	DashboardAddr string `json:"dashboard_addr"`

	// if DashboardPort equals 0, dashboard is not available
	DashboardPort int    `json:"dashboard_port"`
	DashboardUser string `json:"dashboard_user"`
	DashboardPwd  string `json:"dashboard_pwd"`
	AssetsDir     string `json:"asserts_dir"`
	LogFile       string `json:"log_file"`
	LogWay        string `json:"log_way"` // console or file
	LogLevel      string `json:"log_level"`
	LogMaxDays    int64  `json:"log_max_days"`
	Token         string `json:"token"`
	AuthTimeout   int64  `json:"auth_timeout"`
	SubDomainHost string `json:"subdomain_host"`
	TcpMux        bool   `json:"tcp_mux"`

	AllowPorts        map[int]struct{}
	MaxPoolCount      int64 `json:"max_pool_count"`
	MaxPortsPerClient int64 `json:"max_ports_per_client"`
	HeartBeatTimeout  int64 `json:"heart_beat_timeout"`
	UserConnTimeout   int64 `json:"user_conn_timeout"`
}

func GetDefaultServerConf() *ServerCommonConf {
	return &ServerCommonConf{
		BindAddr:          "0.0.0.0",
		BindPort:          7000,
		BindUdpPort:       0,
		KcpBindPort:       0,
		ProxyBindAddr:     "0.0.0.0",
		VhostHttpPort:     0,
		VhostHttpsPort:    0,
		VhostHttpTimeout:  60,
		DashboardAddr:     "0.0.0.0",
		DashboardPort:     0,
		DashboardUser:     "admin",
		DashboardPwd:      "admin",
		AssetsDir:         "",
		LogFile:           "console",
		LogWay:            "console",
		LogLevel:          "info",
		LogMaxDays:        3,
		Token:             "",
		AuthTimeout:       900,
		SubDomainHost:     "",
		TcpMux:            true,
		AllowPorts:        make(map[int]struct{}),
		MaxPoolCount:      5,
		MaxPortsPerClient: 0,
		HeartBeatTimeout:  90,
		UserConnTimeout:   10,
	}
}

func UnmarshalServerConfFromIni(defaultCfg *ServerCommonConf, content string) (cfg *ServerCommonConf, err error) {
	cfg = defaultCfg
	if cfg == nil {
		cfg = GetDefaultServerConf()
	}

	conf, err := ini.Load(strings.NewReader(content))
	if err != nil {
		err = fmt.Errorf("parse ini conf file error: %v", err)
		return nil, err
	}

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

	cfg.Token, _ = conf.Get("common", "token")

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

	if tmpStr, ok = conf.Get("common", "authentication_timeout"); ok {
		v, errRet := strconv.ParseInt(tmpStr, 10, 64)
		if errRet != nil {
			err = fmt.Errorf("Parse conf error: authentication_timeout is incorrect")
			return
		} else {
			cfg.AuthTimeout = v
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

	if tmpStr, ok = conf.Get("common", "heartbeat_timeout"); ok {
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

func (cfg *ServerCommonConf) Check() (err error) {
	return
}

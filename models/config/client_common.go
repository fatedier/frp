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
)

// client common config
type ClientCommonConf struct {
	ServerAddr        string              `json:"server_addr"`
	ServerPort        int                 `json:"server_port"`
	HttpProxy         string              `json:"http_proxy"`
	LogFile           string              `json:"log_file"`
	LogWay            string              `json:"log_way"`
	LogLevel          string              `json:"log_level"`
	LogMaxDays        int64               `json:"log_max_days"`
	Token             string              `json:"token"`
	AdminAddr         string              `json:"admin_addr"`
	AdminPort         int                 `json:"admin_port"`
	AdminUser         string              `json:"admin_user"`
	AdminPwd          string              `json:"admin_pwd"`
	PoolCount         int                 `json:"pool_count"`
	TcpMux            bool                `json:"tcp_mux"`
	User              string              `json:"user"`
	DnsServer         string              `json:"dns_server"`
	LoginFailExit     bool                `json:"login_fail_exit"`
	Start             map[string]struct{} `json:"start"`
	Protocol          string              `json:"protocol"`
	TLSEnable         bool                `json:"tls_enable"`
	HeartBeatInterval int64               `json:"heartbeat_interval"`
	HeartBeatTimeout  int64               `json:"heartbeat_timeout"`
}

func GetDefaultClientConf() *ClientCommonConf {
	return &ClientCommonConf{
		ServerAddr:        "0.0.0.0",
		ServerPort:        7000,
		HttpProxy:         os.Getenv("http_proxy"),
		LogFile:           "console",
		LogWay:            "console",
		LogLevel:          "info",
		LogMaxDays:        3,
		Token:             "",
		AdminAddr:         "127.0.0.1",
		AdminPort:         0,
		AdminUser:         "",
		AdminPwd:          "",
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
	}
}

func UnmarshalClientConfFromIni(defaultCfg *ClientCommonConf, content string) (cfg *ClientCommonConf, err error) {
	cfg = defaultCfg
	if cfg == nil {
		cfg = GetDefaultClientConf()
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

	if tmpStr, ok = conf.Get("common", "token"); ok {
		cfg.Token = tmpStr
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

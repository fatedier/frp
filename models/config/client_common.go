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

var ClientCommonCfg *ClientCommonConf

// client common config
type ClientCommonConf struct {
	ConfigFile        string
	ServerAddr        string
	ServerPort        int
	ServerUdpPort     int // this is specified by login response message from frps
	HttpProxy         string
	LogFile           string
	LogWay            string
	LogLevel          string
	LogMaxDays        int64
	PrivilegeToken    string
	AdminAddr         string
	AdminPort         int
	AdminUser         string
	AdminPwd          string
	PoolCount         int
	TcpMux            bool
	User              string
	LoginFailExit     bool
	Start             map[string]struct{}
	Protocol          string
	HeartBeatInterval int64
	HeartBeatTimeout  int64
}

func GetDeaultClientCommonConf() *ClientCommonConf {
	return &ClientCommonConf{
		ConfigFile:        "./frpc.ini",
		ServerAddr:        "0.0.0.0",
		ServerPort:        7000,
		ServerUdpPort:     0,
		HttpProxy:         "",
		LogFile:           "console",
		LogWay:            "console",
		LogLevel:          "info",
		LogMaxDays:        3,
		PrivilegeToken:    "",
		AdminAddr:         "127.0.0.1",
		AdminPort:         0,
		AdminUser:         "",
		AdminPwd:          "",
		PoolCount:         1,
		TcpMux:            true,
		User:              "",
		LoginFailExit:     true,
		Start:             make(map[string]struct{}),
		Protocol:          "tcp",
		HeartBeatInterval: 30,
		HeartBeatTimeout:  90,
	}
}

func LoadClientCommonConf(conf ini.File) (cfg *ClientCommonConf, err error) {
	var (
		tmpStr string
		ok     bool
		v      int64
	)
	cfg = GetDeaultClientCommonConf()

	tmpStr, ok = conf.Get("common", "server_addr")
	if ok {
		cfg.ServerAddr = tmpStr
	}

	tmpStr, ok = conf.Get("common", "server_port")
	if ok {
		v, err = strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			err = fmt.Errorf("Parse conf error: invalid server_port")
			return
		}
		cfg.ServerPort = int(v)
	}

	tmpStr, ok = conf.Get("common", "http_proxy")
	if ok {
		cfg.HttpProxy = tmpStr
	} else {
		// get http_proxy from env
		cfg.HttpProxy = os.Getenv("http_proxy")
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
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err == nil {
			cfg.LogMaxDays = v
		}
	}

	tmpStr, ok = conf.Get("common", "privilege_token")
	if ok {
		cfg.PrivilegeToken = tmpStr
	}

	tmpStr, ok = conf.Get("common", "admin_addr")
	if ok {
		cfg.AdminAddr = tmpStr
	}

	tmpStr, ok = conf.Get("common", "admin_port")
	if ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err == nil {
			cfg.AdminPort = int(v)
		} else {
			err = fmt.Errorf("Parse conf error: invalid admin_port")
			return
		}
	}

	tmpStr, ok = conf.Get("common", "admin_user")
	if ok {
		cfg.AdminUser = tmpStr
	}

	tmpStr, ok = conf.Get("common", "admin_pwd")
	if ok {
		cfg.AdminPwd = tmpStr
	}

	tmpStr, ok = conf.Get("common", "pool_count")
	if ok {
		v, err = strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			cfg.PoolCount = 1
		} else {
			cfg.PoolCount = int(v)
		}
	}

	tmpStr, ok = conf.Get("common", "tcp_mux")
	if ok && tmpStr == "false" {
		cfg.TcpMux = false
	} else {
		cfg.TcpMux = true
	}

	tmpStr, ok = conf.Get("common", "user")
	if ok {
		cfg.User = tmpStr
	}

	tmpStr, ok = conf.Get("common", "start")
	if ok {
		proxyNames := strings.Split(tmpStr, ",")
		for _, name := range proxyNames {
			cfg.Start[strings.TrimSpace(name)] = struct{}{}
		}
	}

	tmpStr, ok = conf.Get("common", "login_fail_exit")
	if ok && tmpStr == "false" {
		cfg.LoginFailExit = false
	} else {
		cfg.LoginFailExit = true
	}

	tmpStr, ok = conf.Get("common", "protocol")
	if ok {
		// Now it only support tcp and kcp.
		if tmpStr != "kcp" {
			tmpStr = "tcp"
		}
		cfg.Protocol = tmpStr
	}

	tmpStr, ok = conf.Get("common", "heartbeat_timeout")
	if ok {
		v, err = strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			err = fmt.Errorf("Parse conf error: invalid heartbeat_timeout")
			return
		} else {
			cfg.HeartBeatTimeout = v
		}
	}

	tmpStr, ok = conf.Get("common", "heartbeat_interval")
	if ok {
		v, err = strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			err = fmt.Errorf("Parse conf error: invalid heartbeat_interval")
			return
		} else {
			cfg.HeartBeatInterval = v
		}
	}

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

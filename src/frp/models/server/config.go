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

package server

import (
	"fmt"
	"strconv"
	"strings"

	ini "github.com/vaughan0/go-ini"

	"frp/utils/vhost"
)

// common config
var (
	BindAddr         string = "0.0.0.0"
	BindPort         int64  = 7000
	VhostHttpPort    int64  = 0 // if VhostHttpPort equals 0, do not listen a public port for http
	LogFile          string = "console"
	LogWay           string = "console" // console or file
	LogLevel         string = "info"
	LogMaxDays       int64  = 3
	HeartBeatTimeout int64  = 90
	UserConnTimeout  int64  = 10

	VhostMuxer *vhost.HttpMuxer
)

var ProxyServers map[string]*ProxyServer = make(map[string]*ProxyServer)

func LoadConf(confFile string) (err error) {
	var tmpStr string
	var ok bool

	conf, err := ini.LoadFile(confFile)
	if err != nil {
		return err
	}

	// common
	tmpStr, ok = conf.Get("common", "bind_addr")
	if ok {
		BindAddr = tmpStr
	}

	tmpStr, ok = conf.Get("common", "bind_port")
	if ok {
		BindPort, _ = strconv.ParseInt(tmpStr, 10, 64)
	}

	tmpStr, ok = conf.Get("common", "vhost_http_port")
	if ok {
		VhostHttpPort, _ = strconv.ParseInt(tmpStr, 10, 64)
	} else {
		VhostHttpPort = 0
	}

	tmpStr, ok = conf.Get("common", "log_file")
	if ok {
		LogFile = tmpStr
		if LogFile == "console" {
			LogWay = "console"
		} else {
			LogWay = "file"
		}
	}

	tmpStr, ok = conf.Get("common", "log_level")
	if ok {
		LogLevel = tmpStr
	}

	tmpStr, ok = conf.Get("common", "log_max_days")
	if ok {
		LogMaxDays, _ = strconv.ParseInt(tmpStr, 10, 64)
	}

	// servers
	for name, section := range conf {
		if name != "common" {
			proxyServer := &ProxyServer{}
			proxyServer.CustomDomains = make([]string, 0)
			proxyServer.Name = name

			proxyServer.Type, ok = section["type"]
			if ok {
				if proxyServer.Type != "tcp" && proxyServer.Type != "http" {
					return fmt.Errorf("Parse ini file error: proxy [%s] type error", proxyServer.Name)
				}
			} else {
				proxyServer.Type = "tcp"
			}

			proxyServer.AuthToken, ok = section["auth_token"]
			if !ok {
				return fmt.Errorf("Parse ini file error: proxy [%s] no auth_token found", proxyServer.Name)
			}

			// for tcp
			if proxyServer.Type == "tcp" {
				proxyServer.BindAddr, ok = section["bind_addr"]
				if !ok {
					proxyServer.BindAddr = "0.0.0.0"
				}

				portStr, ok := section["listen_port"]
				if ok {
					proxyServer.ListenPort, err = strconv.ParseInt(portStr, 10, 64)
					if err != nil {
						return fmt.Errorf("Parse ini file error: proxy [%s] listen_port error", proxyServer.Name)
					}
				} else {
					return fmt.Errorf("Parse ini file error: proxy [%s] listen_port not found", proxyServer.Name)
				}
			} else if proxyServer.Type == "http" {
				// for http
				domainStr, ok := section["custom_domains"]
				if ok {
					var suffix string
					if VhostHttpPort != 80 {
						suffix = fmt.Sprintf(":%d", VhostHttpPort)
					}
					proxyServer.CustomDomains = strings.Split(domainStr, ",")
					for i, domain := range proxyServer.CustomDomains {
						proxyServer.CustomDomains[i] = strings.ToLower(strings.TrimSpace(domain)) + suffix
					}
				}
			}

			proxyServer.Init()
			ProxyServers[proxyServer.Name] = proxyServer
		}
	}

	if len(ProxyServers) == 0 {
		return fmt.Errorf("Parse ini file error: no proxy config found")
	}

	return nil
}

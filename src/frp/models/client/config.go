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

package client

import (
	"fmt"
	"strconv"

	ini "github.com/vaughan0/go-ini"
)

// common config
var (
	ServerAddr        string = "0.0.0.0"
	ServerPort        int64  = 7000
	LogFile           string = "console"
	LogWay            string = "console"
	LogLevel          string = "info"
	HeartBeatInterval int64  = 20
	HeartBeatTimeout  int64  = 90
)

var ProxyClients map[string]*ProxyClient = make(map[string]*ProxyClient)

func LoadConf(confFile string) (err error) {
	var tmpStr string
	var ok bool

	conf, err := ini.LoadFile(confFile)
	if err != nil {
		return err
	}

	// common
	tmpStr, ok = conf.Get("common", "server_addr")
	if ok {
		ServerAddr = tmpStr
	}

	tmpStr, ok = conf.Get("common", "server_port")
	if ok {
		ServerPort, _ = strconv.ParseInt(tmpStr, 10, 64)
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

	// proxies
	for name, section := range conf {
		if name != "common" {
			proxyClient := &ProxyClient{}
			proxyClient.Name = name

			proxyClient.Passwd, ok = section["passwd"]
			if !ok {
				return fmt.Errorf("Parse ini file error: proxy [%s] no passwd found", proxyClient.Name)
			}

			proxyClient.LocalIp, ok = section["local_ip"]
			if !ok {
				// use 127.0.0.1 as default
				proxyClient.LocalIp = "127.0.0.1"
			}

			portStr, ok := section["local_port"]
			if ok {
				proxyClient.LocalPort, err = strconv.ParseInt(portStr, 10, 64)
				if err != nil {
					return fmt.Errorf("Parse ini file error: proxy [%s] local_port error", proxyClient.Name)
				}
			} else {
				return fmt.Errorf("Parse ini file error: proxy [%s] local_port not found", proxyClient.Name)
			}

			ProxyClients[proxyClient.Name] = proxyClient
		}
	}

	if len(ProxyClients) == 0 {
		return fmt.Errorf("Parse ini file error: no proxy config found")
	}

	return nil
}

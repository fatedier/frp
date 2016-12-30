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
	"os"
	"strconv"
	"strings"

	ini "github.com/vaughan0/go-ini"
)

// common config
var (
	ServerAddr        string = "0.0.0.0"
	ServerPort        int64  = 7000
	HttpProxy         string = ""
	LogFile           string = "console"
	LogWay            string = "console"
	LogLevel          string = "info"
	LogMaxDays        int64  = 3
	PrivilegeToken    string = ""
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

	tmpStr, ok = conf.Get("common", "http_proxy")
	if ok {
		HttpProxy = tmpStr
	} else {
		// get http_proxy from env
		HttpProxy = os.Getenv("http_proxy")
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

	tmpStr, ok = conf.Get("common", "privilege_token")
	if ok {
		PrivilegeToken = tmpStr
	}

	var authToken string
	tmpStr, ok = conf.Get("common", "auth_token")
	if ok {
		authToken = tmpStr
	}

	tmpStr, ok = conf.Get("common", "heartbeat_timeout")
	if ok {
		v, err := strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			return fmt.Errorf("Parse conf error: heartbeat_timeout is incorrect")
		} else {
			HeartBeatTimeout = v
		}
	}

	tmpStr, ok = conf.Get("common", "heartbeat_interval")
	if ok {
		v, err := strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			return fmt.Errorf("Parse conf error: heartbeat_interval is incorrect")
		} else {
			HeartBeatInterval = v
		}
	}

	if HeartBeatInterval <= 0 {
		return fmt.Errorf("Parse conf error: heartbeat_interval is incorrect")
	}

	if HeartBeatTimeout < HeartBeatInterval {
		return fmt.Errorf("Parse conf error: heartbeat_timeout is incorrect, heartbeat_timeout is less than heartbeat_interval")
	}

	// proxies
	for name, section := range conf {
		if name != "common" {
			proxyClient := &ProxyClient{}
			// name
			proxyClient.Name = name

			// auth_token
			proxyClient.AuthToken = authToken

			// local_ip
			proxyClient.LocalIp, ok = section["local_ip"]
			if !ok {
				// use 127.0.0.1 as default
				proxyClient.LocalIp = "127.0.0.1"
			}

			// local_port
			tmpStr, ok = section["local_port"]
			if ok {
				proxyClient.LocalPort, err = strconv.ParseInt(tmpStr, 10, 64)
				if err != nil {
					return fmt.Errorf("Parse conf error: proxy [%s] local_port error", proxyClient.Name)
				}
			} else {
				return fmt.Errorf("Parse conf error: proxy [%s] local_port not found", proxyClient.Name)
			}

			// type
			proxyClient.Type = "tcp"
			tmpStr, ok = section["type"]
			if ok {
				if tmpStr != "tcp" && tmpStr != "http" && tmpStr != "https" && tmpStr != "udp" {
					return fmt.Errorf("Parse conf error: proxy [%s] type error", proxyClient.Name)
				}
				proxyClient.Type = tmpStr
			}

			// use_encryption
			proxyClient.UseEncryption = false
			tmpStr, ok = section["use_encryption"]
			if ok && tmpStr == "true" {
				proxyClient.UseEncryption = true
			}

			// use_gzip
			proxyClient.UseGzip = false
			tmpStr, ok = section["use_gzip"]
			if ok && tmpStr == "true" {
				proxyClient.UseGzip = true
			}

			if proxyClient.Type == "http" {
				// host_header_rewrite
				tmpStr, ok = section["host_header_rewrite"]
				if ok {
					proxyClient.HostHeaderRewrite = tmpStr
				}
				// http_user
				tmpStr, ok = section["http_user"]
				if ok {
					proxyClient.HttpUserName = tmpStr
				}
				// http_pwd
				tmpStr, ok = section["http_pwd"]
				if ok {
					proxyClient.HttpPassWord = tmpStr
				}

			}
			if proxyClient.Type == "http" || proxyClient.Type == "https" {
				// subdomain
				tmpStr, ok = section["subdomain"]
				if ok {
					proxyClient.SubDomain = tmpStr
				}
			}

			// privilege_mode
			proxyClient.PrivilegeMode = false
			tmpStr, ok = section["privilege_mode"]
			if ok && tmpStr == "true" {
				proxyClient.PrivilegeMode = true
			}

			// pool_count
			proxyClient.PoolCount = 0
			tmpStr, ok = section["pool_count"]
			if ok {
				tmpInt, err := strconv.ParseInt(tmpStr, 10, 64)
				if err != nil || tmpInt < 0 {
					return fmt.Errorf("Parse conf error: proxy [%s] pool_count error", proxyClient.Name)
				}
				proxyClient.PoolCount = tmpInt
			}

			// configures used in privilege mode
			if proxyClient.PrivilegeMode == true {
				if PrivilegeToken == "" {
					return fmt.Errorf("Parse conf error: proxy [%s] privilege_token must be set when privilege_mode = true", proxyClient.Name)
				} else {
					proxyClient.PrivilegeToken = PrivilegeToken
				}

				if proxyClient.Type == "tcp" || proxyClient.Type == "udp" {
					// remote_port
					tmpStr, ok = section["remote_port"]
					if ok {
						proxyClient.RemotePort, err = strconv.ParseInt(tmpStr, 10, 64)
						if err != nil {
							return fmt.Errorf("Parse conf error: proxy [%s] remote_port error", proxyClient.Name)
						}
					} else {
						return fmt.Errorf("Parse conf error: proxy [%s] remote_port not found", proxyClient.Name)
					}
				} else if proxyClient.Type == "http" {
					// custom_domains
					domainStr, ok := section["custom_domains"]
					if ok {
						proxyClient.CustomDomains = strings.Split(domainStr, ",")
						if len(proxyClient.CustomDomains) == 0 {
							ok = false
						} else {
							for i, domain := range proxyClient.CustomDomains {
								proxyClient.CustomDomains[i] = strings.ToLower(strings.TrimSpace(domain))
							}
						}
					}

					if !ok && proxyClient.SubDomain == "" {
						return fmt.Errorf("Parse conf error: proxy [%s] custom_domains and subdomain should set at least one of them when type is http", proxyClient.Name)
					}

					// locations
					locations, ok := section["locations"]
					if ok {
						proxyClient.Locations = strings.Split(locations, ",")
					} else {
						proxyClient.Locations = []string{""}
					}
				} else if proxyClient.Type == "https" {
					// custom_domains
					domainStr, ok := section["custom_domains"]
					if ok {
						proxyClient.CustomDomains = strings.Split(domainStr, ",")
						if len(proxyClient.CustomDomains) == 0 {
							ok = false
						} else {
							for i, domain := range proxyClient.CustomDomains {
								proxyClient.CustomDomains[i] = strings.ToLower(strings.TrimSpace(domain))
							}
						}
					}

					if !ok && proxyClient.SubDomain == "" {
						return fmt.Errorf("Parse conf error: proxy [%s] custom_domains and subdomain should set at least one of them when type is https", proxyClient.Name)
					}
				}
			}

			ProxyClients[proxyClient.Name] = proxyClient
		}
	}

	if len(ProxyClients) == 0 {
		return fmt.Errorf("Parse conf error: no proxy config found")
	}

	return nil
}

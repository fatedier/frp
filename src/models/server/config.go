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
	"sync"

	ini "github.com/vaughan0/go-ini"

	"github.com/fatedier/frp/src/models/consts"
	"github.com/fatedier/frp/src/models/metric"
	"github.com/fatedier/frp/src/utils/log"
	"github.com/fatedier/frp/src/utils/vhost"
)

// common config
var (
	ConfigFile        string = "./frps.ini"
	BindAddr          string = "0.0.0.0"
	BindPort          int64  = 7000
	VhostHttpPort     int64  = 0 // if VhostHttpPort equals 0, don't listen a public port for http protocol
	VhostHttpsPort    int64  = 0 // if VhostHttpsPort equals 0, don't listen a public port for https protocol
	DashboardPort     int64  = 0 // if DashboardPort equals 0, dashboard is not available
	DashboardUsername string = "admin"
	DashboardPassword string = "admin"
	AssetsDir         string = ""
	LogFile           string = "console"
	LogWay            string = "console" // console or file
	LogLevel          string = "info"
	LogMaxDays        int64  = 3
	PrivilegeMode     bool   = false
	PrivilegeToken    string = ""
	AuthTimeout       int64  = 900
	SubDomainHost     string = ""

	// if PrivilegeAllowPorts is not nil, tcp proxies which remote port exist in this map can be connected
	PrivilegeAllowPorts [][2]int64
	MaxPoolCount        int64 = 100
	HeartBeatTimeout    int64 = 30
	UserConnTimeout     int64 = 10

	VhostHttpMuxer    *vhost.HttpMuxer
	VhostHttpsMuxer   *vhost.HttpsMuxer
	ProxyServers      map[string]*ProxyServer = make(map[string]*ProxyServer) // all proxy servers info and resources
	ProxyServersMutex sync.RWMutex
)

func LoadConf(confFile string) (err error) {
	err = loadCommonConf(confFile)
	if err != nil {
		return err
	}

	// load all proxy server's configure and initialize
	// and set ProxyServers map
	newProxyServers, err := loadProxyConf(confFile)
	if err != nil {
		return err
	}
	for _, proxyServer := range newProxyServers {
		proxyServer.Init()
	}
	ProxyServersMutex.Lock()
	ProxyServers = newProxyServers
	ProxyServersMutex.Unlock()
	return nil
}

func loadCommonConf(confFile string) error {
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
		v, err := strconv.ParseInt(tmpStr, 10, 64)
		if err == nil {
			BindPort = v
		}
	}

	tmpStr, ok = conf.Get("common", "vhost_http_port")
	if ok {
		VhostHttpPort, _ = strconv.ParseInt(tmpStr, 10, 64)
	} else {
		VhostHttpPort = 0
	}

	tmpStr, ok = conf.Get("common", "vhost_https_port")
	if ok {
		VhostHttpsPort, _ = strconv.ParseInt(tmpStr, 10, 64)
	} else {
		VhostHttpsPort = 0
	}

	tmpStr, ok = conf.Get("common", "dashboard_port")
	if ok {
		DashboardPort, _ = strconv.ParseInt(tmpStr, 10, 64)
	} else {
		DashboardPort = 0
	}

	tmpStr, ok = conf.Get("common", "dashboard_user")
	if ok {
		DashboardUsername = tmpStr
	}

	tmpStr, ok = conf.Get("common", "dashboard_pwd")
	if ok {
		DashboardPassword = tmpStr
	}

	tmpStr, ok = conf.Get("common", "assets_dir")
	if ok {
		AssetsDir = tmpStr
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
		v, err := strconv.ParseInt(tmpStr, 10, 64)
		if err == nil {
			LogMaxDays = v
		}
	}

	tmpStr, ok = conf.Get("common", "privilege_mode")
	if ok {
		if tmpStr == "true" {
			PrivilegeMode = true
		}
	}

	if PrivilegeMode == true {
		tmpStr, ok = conf.Get("common", "privilege_token")
		if ok {
			if tmpStr == "" {
				return fmt.Errorf("Parse conf error: privilege_token can not be null")
			}
			PrivilegeToken = tmpStr
		} else {
			return fmt.Errorf("Parse conf error: privilege_token must be set if privilege_mode is enabled")
		}

		allowPortsStr, ok := conf.Get("common", "privilege_allow_ports")
		// TODO: check if conflicts exist in port ranges
		if ok {
			PrivilegeAllowPorts, err = getPortRanges(allowPortsStr)
			if err != nil {
				return err
			}
		}
	}

	tmpStr, ok = conf.Get("common", "max_pool_count")
	if ok {
		v, err := strconv.ParseInt(tmpStr, 10, 64)
		if err == nil && v >= 0 {
			MaxPoolCount = v
		}
	}
	tmpStr, ok = conf.Get("common", "authentication_timeout")
	if ok {
		v, err := strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			return fmt.Errorf("Parse conf error: authentication_timeout is incorrect")
		} else {
			AuthTimeout = v
		}
	}
	SubDomainHost, ok = conf.Get("common", "subdomain_host")
	if ok {
		SubDomainHost = strings.ToLower(strings.TrimSpace(SubDomainHost))
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
	return nil
}

func loadProxyConf(confFile string) (proxyServers map[string]*ProxyServer, err error) {
	var ok bool
	proxyServers = make(map[string]*ProxyServer)
	conf, err := ini.LoadFile(confFile)
	if err != nil {
		return proxyServers, err
	}
	// servers
	for name, section := range conf {
		if name != "common" {
			proxyServer := NewProxyServer()
			proxyServer.Name = name

			proxyServer.Type, ok = section["type"]
			if ok {
				if proxyServer.Type != "tcp" && proxyServer.Type != "http" && proxyServer.Type != "https" && proxyServer.Type != "udp" {
					return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] type error", proxyServer.Name)
				}
			} else {
				proxyServer.Type = "tcp"
			}

			proxyServer.AuthToken, ok = section["auth_token"]
			if !ok {
				return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] no auth_token found", proxyServer.Name)
			}

			// for tcp and udp
			if proxyServer.Type == "tcp" || proxyServer.Type == "udp" {
				proxyServer.BindAddr, ok = section["bind_addr"]
				if !ok {
					proxyServer.BindAddr = "0.0.0.0"
				}

				portStr, ok := section["listen_port"]
				if ok {
					proxyServer.ListenPort, err = strconv.ParseInt(portStr, 10, 64)
					if err != nil {
						return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] listen_port error", proxyServer.Name)
					}
				} else {
					return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] listen_port not found", proxyServer.Name)
				}
			} else if proxyServer.Type == "http" {
				// for http
				proxyServer.ListenPort = VhostHttpPort

				domainStr, ok := section["custom_domains"]
				if ok {
					proxyServer.CustomDomains = strings.Split(domainStr, ",")
					for i, domain := range proxyServer.CustomDomains {
						domain = strings.ToLower(strings.TrimSpace(domain))
						// custom domain should not belong to subdomain_host
						if SubDomainHost != "" && strings.Contains(domain, SubDomainHost) {
							return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] custom domain should not belong to subdomain_host", proxyServer.Name)
						}
						proxyServer.CustomDomains[i] = domain
					}
				}

				// subdomain
				subdomainStr, ok := section["subdomain"]
				if ok {
					if strings.Contains(subdomainStr, ".") || strings.Contains(subdomainStr, "*") {
						return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] '.' and '*' is not supported in subdomain", proxyServer.Name)
					}

					if SubDomainHost == "" {
						return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] subdomain is not supported because subdomain_host is empty", proxyServer.Name)
					}
					proxyServer.SubDomain = subdomainStr + "." + SubDomainHost
				}

				if len(proxyServer.CustomDomains) == 0 && proxyServer.SubDomain == "" {
					return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] custom_domains and subdomain should set at least one of them when type is http", proxyServer.Name)
				}

				// locations
				locations, ok := section["locations"]
				if ok {
					proxyServer.Locations = strings.Split(locations, ",")
				} else {
					proxyServer.Locations = []string{""}
				}
			} else if proxyServer.Type == "https" {
				// for https
				proxyServer.ListenPort = VhostHttpsPort

				domainStr, ok := section["custom_domains"]
				if ok {
					proxyServer.CustomDomains = strings.Split(domainStr, ",")
					for i, domain := range proxyServer.CustomDomains {
						domain = strings.ToLower(strings.TrimSpace(domain))
						if SubDomainHost != "" && strings.Contains(domain, SubDomainHost) {
							return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] custom domain should not belong to subdomain_host", proxyServer.Name)
						}
						proxyServer.CustomDomains[i] = domain
					}
				}

				// subdomain
				subdomainStr, ok := section["subdomain"]
				if ok {
					if strings.Contains(subdomainStr, ".") || strings.Contains(subdomainStr, "*") {
						return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] '.' and '*' is not supported in subdomain", proxyServer.Name)
					}

					if SubDomainHost == "" {
						return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] subdomain is not supported because subdomain_host is empty", proxyServer.Name)
					}
					proxyServer.SubDomain = subdomainStr + "." + SubDomainHost
				}

				if len(proxyServer.CustomDomains) == 0 && proxyServer.SubDomain == "" {
					return proxyServers, fmt.Errorf("Parse conf error: proxy [%s] custom_domains and subdomain should set at least one of them when type is https", proxyServer.Name)
				}
			}
			proxyServers[proxyServer.Name] = proxyServer
		}
	}

	// set metric statistics of all proxies
	for name, p := range proxyServers {
		metric.SetProxyInfo(name, p.Type, p.BindAddr, p.UseEncryption, p.UseGzip,
			p.PrivilegeMode, p.CustomDomains, p.Locations, p.ListenPort)
	}
	return proxyServers, nil
}

// the function can only reload proxy configures
// common section won't be changed
func ReloadConf(confFile string) (err error) {
	loadProxyServers, err := loadProxyConf(confFile)
	if err != nil {
		return err
	}

	ProxyServersMutex.Lock()
	for name, proxyServer := range loadProxyServers {
		oldProxyServer, ok := ProxyServers[name]
		if ok {
			if !oldProxyServer.Compare(proxyServer) {
				oldProxyServer.Close()
				proxyServer.Init()
				ProxyServers[name] = proxyServer
				log.Info("ProxyName [%s] configure change, restart", name)
			}
		} else {
			proxyServer.Init()
			ProxyServers[name] = proxyServer
			log.Info("ProxyName [%s] is new, init it", name)
		}
	}

	// proxies created by PrivilegeMode won't be deleted
	for name, oldProxyServer := range ProxyServers {
		_, ok := loadProxyServers[name]
		if !ok {
			if !oldProxyServer.PrivilegeMode {
				oldProxyServer.Close()
				delete(ProxyServers, name)
				log.Info("ProxyName [%s] deleted, close it", name)
			} else {
				log.Info("ProxyName [%s] created by PrivilegeMode, won't be closed", name)
			}
		}
	}
	ProxyServersMutex.Unlock()
	return nil
}

func CreateProxy(s *ProxyServer) error {
	ProxyServersMutex.Lock()
	defer ProxyServersMutex.Unlock()
	oldServer, ok := ProxyServers[s.Name]
	if ok {
		if oldServer.Status == consts.Working {
			return fmt.Errorf("this proxy is already working now")
		}
		oldServer.Lock()
		oldServer.Release()
		oldServer.Unlock()
		if oldServer.PrivilegeMode {
			delete(ProxyServers, s.Name)
		}
	}
	ProxyServers[s.Name] = s
	metric.SetProxyInfo(s.Name, s.Type, s.BindAddr, s.UseEncryption, s.UseGzip,
		s.PrivilegeMode, s.CustomDomains, s.Locations, s.ListenPort)
	return nil
}

func DeleteProxy(proxyName string) {
	ProxyServersMutex.Lock()
	defer ProxyServersMutex.Unlock()
	delete(ProxyServers, proxyName)
}

func GetProxyServer(proxyName string) (p *ProxyServer, ok bool) {
	ProxyServersMutex.RLock()
	defer ProxyServersMutex.RUnlock()
	p, ok = ProxyServers[proxyName]
	return
}

func getPortRanges(rangeStr string) (portRanges [][2]int64, err error) {
	// for example: 1000-2000,2001,2002,3000-4000
	log.Debug("rangeStr is %s", rangeStr)
	rangeArray := strings.Split(rangeStr, ",")
	for _, portRangeStr := range rangeArray {
		// 1000-2000 or 2001
		portArray := strings.Split(portRangeStr, "-")
		// length: only 1 or 2 is correct
		rangeType := len(portArray)
		if rangeType == 1 {
			singlePort, err := strconv.ParseInt(portArray[0], 10, 64)
			if err != nil {
				return [][2]int64{}, fmt.Errorf("Parse conf error: privilege_allow_ports is incorrect, %v", err)
			}
			portRanges = append(portRanges, [2]int64{singlePort, singlePort})
		} else if rangeType == 2 {
			min, err := strconv.ParseInt(portArray[0], 10, 64)
			if err != nil {
				return [][2]int64{}, fmt.Errorf("Parse conf error: privilege_allow_ports is incorrect, %v", err)
			}
			max, err := strconv.ParseInt(portArray[1], 10, 64)
			if err != nil {
				return [][2]int64{}, fmt.Errorf("Parse conf error: privilege_allow_ports is incorrect, %v", err)
			}
			if max < min {
				return [][2]int64{}, fmt.Errorf("Parse conf error: privilege_allow_ports range incorrect")
			}
			portRanges = append(portRanges, [2]int64{min, max})
		} else {
			return [][2]int64{}, fmt.Errorf("Parse conf error: privilege_allow_ports is incorrect")
		}
	}
	return portRanges, nil
}

func ContainsPort(portRanges [][2]int64, port int64) bool {
	for _, pr := range portRanges {
		if port >= pr[0] && port <= pr[1] {
			return true
		}
	}
	return false
}

func PortRangesCut(portRanges [][2]int64, port int64) [][2]int64 {
	var tmpRanges [][2]int64
	for _, pr := range portRanges {
		if port >= pr[0] && port <= pr[1] {
			leftRange := [2]int64{pr[0], port - 1}
			rightRange := [2]int64{port + 1, pr[1]}
			if leftRange[0] <= leftRange[1] {
				tmpRanges = append(tmpRanges, leftRange)
			}
			if rightRange[0] <= rightRange[1] {
				tmpRanges = append(tmpRanges, rightRange)
			}
		} else {
			tmpRanges = append(tmpRanges, pr)
		}
	}
	return tmpRanges
}

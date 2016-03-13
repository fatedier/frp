package server

import (
	"fmt"
	"strconv"

	ini "github.com/vaughan0/go-ini"
)

// common config
var (
	BindAddr         string = "0.0.0.0"
	BindPort         int64  = 7000
	LogFile          string = "console"
	LogWay           string = "console" // console or file
	LogLevel         string = "info"
	HeartBeatTimeout int64  = 30
	UserConnTimeout  int64  = 10
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

	// servers
	for name, section := range conf {
		if name != "common" {
			proxyServer := &ProxyServer{}
			proxyServer.Name = name

			proxyServer.Passwd, ok = section["passwd"]
			if !ok {
				return fmt.Errorf("Parse ini file error: proxy [%s] no passwd found", proxyServer.Name)
			}

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

			proxyServer.Init()
			ProxyServers[proxyServer.Name] = proxyServer
		}
	}

	if len(ProxyServers) == 0 {
		return fmt.Errorf("Parse ini file error: no proxy config found")
	}

	return nil
}

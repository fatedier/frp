// Copyright 2018 fatedier, fatedier@gmail.com
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

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/g"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/server"
	"github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/util"
	"github.com/fatedier/frp/utils/version"
)

const (
	CfgFileTypeIni = iota
	CfgFileTypeCmd
)

var (
	cfgFile     string
	showVersion bool

	bindAddr          string
	bindPort          int
	bindUdpPort       int
	kcpBindPort       int
	proxyBindAddr     string
	vhostHttpPort     int
	vhostHttpsPort    int
	vhostHttpTimeout  int64
	dashboardAddr     string
	dashboardPort     int
	dashboardUser     string
	dashboardPwd      string
	assetsDir         string
	logFile           string
	logWay            string
	logLevel          string
	logMaxDays        int64
	token             string
	authTimeout       int64
	subDomainHost     string
	tcpMux            bool
	allowPorts        string
	maxPoolCount      int64
	maxPortsPerClient int64
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "", "c", "", "config file of frps")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "version of frpc")

	rootCmd.PersistentFlags().StringVarP(&bindAddr, "bind_addr", "", "0.0.0.0", "bind address")
	rootCmd.PersistentFlags().IntVarP(&bindPort, "bind_port", "p", 7000, "bind port")
	rootCmd.PersistentFlags().IntVarP(&bindUdpPort, "bind_udp_port", "", 0, "bind udp port")
	rootCmd.PersistentFlags().IntVarP(&kcpBindPort, "kcp_bind_port", "", 0, "kcp bind udp port")
	rootCmd.PersistentFlags().StringVarP(&proxyBindAddr, "proxy_bind_addr", "", "0.0.0.0", "proxy bind address")
	rootCmd.PersistentFlags().IntVarP(&vhostHttpPort, "vhost_http_port", "", 0, "vhost http port")
	rootCmd.PersistentFlags().IntVarP(&vhostHttpsPort, "vhost_https_port", "", 0, "vhost https port")
	rootCmd.PersistentFlags().Int64VarP(&vhostHttpTimeout, "vhost_http_timeout", "", 60, "vhost http response header timeout")
	rootCmd.PersistentFlags().StringVarP(&dashboardAddr, "dashboard_addr", "", "0.0.0.0", "dasboard address")
	rootCmd.PersistentFlags().IntVarP(&dashboardPort, "dashboard_port", "", 0, "dashboard port")
	rootCmd.PersistentFlags().StringVarP(&dashboardUser, "dashboard_user", "", "admin", "dashboard user")
	rootCmd.PersistentFlags().StringVarP(&dashboardPwd, "dashboard_pwd", "", "admin", "dashboard password")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log_file", "", "console", "log file")
	rootCmd.PersistentFlags().StringVarP(&logWay, "log_way", "", "console", "log way")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log_level", "", "info", "log level")
	rootCmd.PersistentFlags().Int64VarP(&logMaxDays, "log_max_days", "", 3, "log_max_days")
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "auth token")
	rootCmd.PersistentFlags().Int64VarP(&authTimeout, "auth_timeout", "", 900, "auth timeout")
	rootCmd.PersistentFlags().StringVarP(&subDomainHost, "subdomain_host", "", "", "subdomain host")
	rootCmd.PersistentFlags().StringVarP(&allowPorts, "allow_ports", "", "", "allow ports")
	rootCmd.PersistentFlags().Int64VarP(&maxPortsPerClient, "max_ports_per_client", "", 0, "max ports per client")
}

var rootCmd = &cobra.Command{
	Use:   "frps",
	Short: "frps is the server of frp (https://github.com/fatedier/frp)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version.Full())
			return nil
		}

		var err error

		if cfgFile != "" {
			err = parseServerCommonCfg(CfgFileTypeIni, cfgFile)
			if err != nil {
				return err
			}

			conf, err := ini.LoadFile(cfgFile)
			if err != nil {
				return err
			}
			for name := range conf {
				err = parseSubServerCommonCfgFromIni(cfgFile, name)
				if err != nil {
					return err
				}
			}
		} else {
			err = parseServerCommonCfg(CfgFileTypeCmd, "")
			if err != nil {
				return err
			}
		}

		err = runServer()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func parseServerCommonCfg(fileType int, filePath string) (err error) {
	if fileType == CfgFileTypeIni {
		err = parseServerCommonCfgFromIni(filePath)
	} else if fileType == CfgFileTypeCmd {
		err = parseServerCommonCfgFromCmd()
	}
	if err != nil {
		return
	}

	g.GlbServerCfg.CfgFile = filePath

	err = g.GlbServerCfg.ServerSectionConf.Check()
	if err != nil {
		return
	}

	config.InitServerCfg(&g.GlbServerCfg.ServerSectionConf)
	return
}

func parseServerCommonCfgFromIni(filePath string) (err error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(b)

	cfg, err := config.UnmarshalServerConfFromIni(&g.GlbServerCfg.ServerSectionConf, content)
	if err != nil {
		return err
	}

	g.GlbServerCfg.ServerSectionConf = *cfg
	return
}

func parseServerCommonCfgFromCmd() (err error) {
	g.GlbServerCfg.BindAddr = bindAddr
	g.GlbServerCfg.BindPort = bindPort
	g.GlbServerCfg.BindUdpPort = bindUdpPort
	g.GlbServerCfg.KcpBindPort = kcpBindPort
	g.GlbServerCfg.ProxyBindAddr = proxyBindAddr
	g.GlbServerCfg.VhostHttpPort = vhostHttpPort
	g.GlbServerCfg.VhostHttpsPort = vhostHttpsPort
	g.GlbServerCfg.VhostHttpTimeout = vhostHttpTimeout
	g.GlbServerCfg.DashboardAddr = dashboardAddr
	g.GlbServerCfg.DashboardPort = dashboardPort
	g.GlbServerCfg.DashboardUser = dashboardUser
	g.GlbServerCfg.DashboardPwd = dashboardPwd
	g.GlbServerCfg.LogFile = logFile
	g.GlbServerCfg.LogWay = logWay
	g.GlbServerCfg.LogLevel = logLevel
	g.GlbServerCfg.LogMaxDays = logMaxDays
	g.GlbServerCfg.AuthTimeout = authTimeout
	g.GlbServerCfg.SubDomainHost = subDomainHost
	g.GlbServerCfg.MaxPortsPerClient = maxPortsPerClient

	cfg := config.GetDefaultSubServerConf()
	cfg.Section = "common"
	cfg.Token = token
	if len(allowPorts) > 0 {
		// e.g. 1000-2000,2001,2002,3000-4000
		ports, errRet := util.ParseRangeNumbers(allowPorts)
		if errRet != nil {
			err = fmt.Errorf("Parse cmd conf error: allow_ports: %v", errRet)
			return
		}

		for _, port := range ports {
			cfg.AllowPorts[int(port)] = struct{}{}
		}
	}
	g.GlbServerSubSectionMap["common"] = cfg

	return
}


func parseSubServerCommonCfgFromIni(filePath string, section string) (err error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(b)

	cfg, err := config.UnmarshalSubServerConfFromIni(content, section)
	if err != nil {
		return err
	}

	g.GlbServerSubSectionMap[section] = cfg

	return
}

func runServer() (err error) {
	log.InitLog(g.GlbServerCfg.LogWay, g.GlbServerCfg.LogFile, g.GlbServerCfg.LogLevel, g.GlbServerCfg.LogMaxDays)
	svr, err := server.NewService()
	if err != nil {
		return err
	}
	log.Info("Start frps success")
	server.ServerService = svr
	svr.Run()
	return
}

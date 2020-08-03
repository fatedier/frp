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
	"os"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/models/auth"
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
	logLevel          string
	logMaxDays        int64
	disableLogColor   bool
	token             string
	subDomainHost     string
	tcpMux            bool
	allowPorts        string
	maxPoolCount      int64
	maxPortsPerClient int64
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file of frps")
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
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log_level", "", "info", "log level")
	rootCmd.PersistentFlags().Int64VarP(&logMaxDays, "log_max_days", "", 3, "log max days")
	rootCmd.PersistentFlags().BoolVarP(&disableLogColor, "disable_log_color", "", false, "disable log color in console")

	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "auth token")
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

		var cfg config.ServerCommonConf
		var err error
		if cfgFile != "" {
			var content string
			content, err = config.GetRenderedConfFromFile(cfgFile)
			if err != nil {
				return err
			}
			cfg, err = parseServerCommonCfg(CfgFileTypeIni, content)
		} else {
			cfg, err = parseServerCommonCfg(CfgFileTypeCmd, "")
		}
		if err != nil {
			return err
		}

		err = runServer(cfg)
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

func parseServerCommonCfg(fileType int, content string) (cfg config.ServerCommonConf, err error) {
	if fileType == CfgFileTypeIni {
		cfg, err = parseServerCommonCfgFromIni(content)
	} else if fileType == CfgFileTypeCmd {
		cfg, err = parseServerCommonCfgFromCmd()
	}
	if err != nil {
		return
	}

	err = cfg.Check()
	if err != nil {
		return
	}
	return
}

func parseServerCommonCfgFromIni(content string) (config.ServerCommonConf, error) {
	cfg, err := config.UnmarshalServerConfFromIni(content)
	if err != nil {
		return config.ServerCommonConf{}, err
	}
	return cfg, nil
}

func parseServerCommonCfgFromCmd() (cfg config.ServerCommonConf, err error) {
	cfg = config.GetDefaultServerConf()

	cfg.BindAddr = bindAddr
	cfg.BindPort = bindPort
	cfg.BindUdpPort = bindUdpPort
	cfg.KcpBindPort = kcpBindPort
	cfg.ProxyBindAddr = proxyBindAddr
	cfg.VhostHttpPort = vhostHttpPort
	cfg.VhostHttpsPort = vhostHttpsPort
	cfg.VhostHttpTimeout = vhostHttpTimeout
	cfg.DashboardAddr = dashboardAddr
	cfg.DashboardPort = dashboardPort
	cfg.DashboardUser = dashboardUser
	cfg.DashboardPwd = dashboardPwd
	cfg.LogFile = logFile
	cfg.LogLevel = logLevel
	cfg.LogMaxDays = logMaxDays
	cfg.SubDomainHost = subDomainHost

	// Only token authentication is supported in cmd mode
	cfg.AuthServerConfig = auth.GetDefaultAuthServerConf()
	cfg.Token = token
	if len(allowPorts) > 0 {
		// e.g. 1000-2000,2001,2002,3000-4000
		ports, errRet := util.ParseRangeNumbers(allowPorts)
		if errRet != nil {
			err = fmt.Errorf("Parse conf error: allow_ports: %v", errRet)
			return
		}

		for _, port := range ports {
			cfg.AllowPorts[int(port)] = struct{}{}
		}
	}
	cfg.MaxPortsPerClient = maxPortsPerClient

	if logFile == "console" {
		cfg.LogWay = "console"
	} else {
		cfg.LogWay = "file"
	}
	cfg.DisableLogColor = disableLogColor
	return
}

func runServer(cfg config.ServerCommonConf) (err error) {
	log.InitLog(cfg.LogWay, cfg.LogFile, cfg.LogLevel, cfg.LogMaxDays, cfg.DisableLogColor)
	svr, err := server.NewService(cfg)
	if err != nil {
		return err
	}
	log.Info("start frps success")
	svr.Run()
	return
}

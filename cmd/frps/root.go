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
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/types"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/frp/server"
)

var (
	cfgFile     string
	showVersion bool

	bindAddr             string
	bindPort             int
	kcpBindPort          int
	proxyBindAddr        string
	vhostHTTPPort        int
	vhostHTTPSPort       int
	vhostHTTPTimeout     int64
	dashboardAddr        string
	dashboardPort        int
	dashboardUser        string
	dashboardPwd         string
	enablePrometheus     bool
	logFile              string
	logLevel             string
	logMaxDays           int64
	disableLogColor      bool
	token                string
	subDomainHost        string
	allowPorts           string
	maxPortsPerClient    int64
	tlsOnly              bool
	dashboardTLSMode     bool
	dashboardTLSCertFile string
	dashboardTLSKeyFile  string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file of frps")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "version of frps")

	rootCmd.PersistentFlags().StringVarP(&bindAddr, "bind_addr", "", "0.0.0.0", "bind address")
	rootCmd.PersistentFlags().IntVarP(&bindPort, "bind_port", "p", 7000, "bind port")
	rootCmd.PersistentFlags().IntVarP(&kcpBindPort, "kcp_bind_port", "", 0, "kcp bind udp port")
	rootCmd.PersistentFlags().StringVarP(&proxyBindAddr, "proxy_bind_addr", "", "0.0.0.0", "proxy bind address")
	rootCmd.PersistentFlags().IntVarP(&vhostHTTPPort, "vhost_http_port", "", 0, "vhost http port")
	rootCmd.PersistentFlags().IntVarP(&vhostHTTPSPort, "vhost_https_port", "", 0, "vhost https port")
	rootCmd.PersistentFlags().Int64VarP(&vhostHTTPTimeout, "vhost_http_timeout", "", 60, "vhost http response header timeout")
	rootCmd.PersistentFlags().StringVarP(&dashboardAddr, "dashboard_addr", "", "0.0.0.0", "dashboard address")
	rootCmd.PersistentFlags().IntVarP(&dashboardPort, "dashboard_port", "", 0, "dashboard port")
	rootCmd.PersistentFlags().StringVarP(&dashboardUser, "dashboard_user", "", "admin", "dashboard user")
	rootCmd.PersistentFlags().StringVarP(&dashboardPwd, "dashboard_pwd", "", "admin", "dashboard password")
	rootCmd.PersistentFlags().BoolVarP(&enablePrometheus, "enable_prometheus", "", false, "enable prometheus dashboard")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log_file", "", "console", "log file")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log_level", "", "info", "log level")
	rootCmd.PersistentFlags().Int64VarP(&logMaxDays, "log_max_days", "", 3, "log max days")
	rootCmd.PersistentFlags().BoolVarP(&disableLogColor, "disable_log_color", "", false, "disable log color in console")

	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "auth token")
	rootCmd.PersistentFlags().StringVarP(&subDomainHost, "subdomain_host", "", "", "subdomain host")
	rootCmd.PersistentFlags().StringVarP(&allowPorts, "allow_ports", "", "", "allow ports")
	rootCmd.PersistentFlags().Int64VarP(&maxPortsPerClient, "max_ports_per_client", "", 0, "max ports per client")
	rootCmd.PersistentFlags().BoolVarP(&tlsOnly, "tls_only", "", false, "frps tls only")
	rootCmd.PersistentFlags().BoolVarP(&dashboardTLSMode, "dashboard_tls_mode", "", false, "dashboard tls mode")
	rootCmd.PersistentFlags().StringVarP(&dashboardTLSCertFile, "dashboard_tls_cert_file", "", "", "dashboard tls cert file")
	rootCmd.PersistentFlags().StringVarP(&dashboardTLSKeyFile, "dashboard_tls_key_file", "", "", "dashboard tls key file")
}

var rootCmd = &cobra.Command{
	Use:   "frps",
	Short: "frps is the server of frp (https://github.com/fatedier/frp)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version.Full())
			return nil
		}

		var (
			svrCfg         *v1.ServerConfig
			isLegacyFormat bool
			err            error
		)
		if cfgFile != "" {
			svrCfg, isLegacyFormat, err = config.LoadServerConfig(cfgFile)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if isLegacyFormat {
				fmt.Printf("WARNING: ini format is deprecated and the support will be removed in the future, " +
					"please use yaml/json/toml format instead!\n")
			}
		} else {
			if svrCfg, err = parseServerConfigFromCmd(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}

		warning, err := validation.ValidateServerConfig(svrCfg)
		if warning != nil {
			fmt.Printf("WARNING: %v\n", warning)
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := runServer(svrCfg); err != nil {
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

func parseServerConfigFromCmd() (*v1.ServerConfig, error) {
	cfg := &v1.ServerConfig{}

	cfg.BindAddr = bindAddr
	cfg.BindPort = bindPort
	cfg.KCPBindPort = kcpBindPort
	cfg.ProxyBindAddr = proxyBindAddr
	cfg.VhostHTTPPort = vhostHTTPPort
	cfg.VhostHTTPSPort = vhostHTTPSPort
	cfg.VhostHTTPTimeout = vhostHTTPTimeout
	cfg.WebServer.Addr = dashboardAddr
	cfg.WebServer.Port = dashboardPort
	cfg.WebServer.User = dashboardUser
	cfg.WebServer.Password = dashboardPwd
	cfg.EnablePrometheus = enablePrometheus
	if dashboardTLSMode {
		cfg.WebServer.TLS = &v1.TLSConfig{
			CertFile: dashboardTLSCertFile,
			KeyFile:  dashboardTLSKeyFile,
		}
	}
	cfg.Log.To = logFile
	cfg.Log.Level = logLevel
	cfg.Log.MaxDays = logMaxDays
	cfg.Log.DisablePrintColor = disableLogColor
	cfg.SubDomainHost = subDomainHost
	cfg.Transport.TLS.Force = tlsOnly
	cfg.MaxPortsPerClient = maxPortsPerClient

	// Only token authentication is supported in cmd mode
	cfg.Auth.Token = token

	if len(allowPorts) > 0 {
		portsRanges, err := types.NewPortsRangeSliceFromString(allowPorts)
		if err != nil {
			return cfg, fmt.Errorf("allow_ports format error: %v", err)
		}
		cfg.AllowPorts = portsRanges
	}

	cfg.Complete()
	return cfg, nil
}

func runServer(cfg *v1.ServerConfig) (err error) {
	log.InitLog(cfg.Log.To, cfg.Log.Level, cfg.Log.MaxDays, cfg.Log.DisablePrintColor)

	if cfgFile != "" {
		log.Info("frps uses config file: %s", cfgFile)
	} else {
		log.Info("frps uses command line arguments for config")
	}

	svr, err := server.NewService(cfg)
	if err != nil {
		return err
	}
	log.Info("frps started successfully")
	svr.Run(context.Background())
	return
}

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

package sub

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/models/auth"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/version"
)

const (
	CfgFileTypeIni = iota
	CfgFileTypeCmd
)

var (
	cfgFile     string
	showVersion bool

	serverAddr      string
	user            string
	protocol        string
	token           string
	logLevel        string
	logFile         string
	logMaxDays      int
	disableLogColor bool

	proxyName         string
	localIp           string
	localPort         int
	remotePort        int
	useEncryption     bool
	useCompression    bool
	customDomains     string
	subDomain         string
	httpUser          string
	httpPwd           string
	locations         string
	hostHeaderRewrite string
	role              string
	sk                string
	multiplexer       string
	serverName        string
	bindAddr          string
	bindPort          int

	kcpDoneCh chan struct{}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./frpc.ini", "config file of frpc")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "version of frpc")

	kcpDoneCh = make(chan struct{})
}

var rootCmd = &cobra.Command{
	Use:   "frpc",
	Short: "frpc is the client of frp (https://github.com/fatedier/frp)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version.Full())
			return nil
		}

		// Do not show command usage here.
		err := runClient(cfgFile)
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

func handleSignal(svr *client.Service) {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	svr.Close()
	time.Sleep(250 * time.Millisecond)
	close(kcpDoneCh)
}

func parseClientCommonCfg(fileType int, content string) (cfg config.ClientCommonConf, err error) {
	if fileType == CfgFileTypeIni {
		cfg, err = parseClientCommonCfgFromIni(content)
	} else if fileType == CfgFileTypeCmd {
		cfg, err = parseClientCommonCfgFromCmd()
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

func parseClientCommonCfgFromIni(content string) (config.ClientCommonConf, error) {
	cfg, err := config.UnmarshalClientConfFromIni(content)
	if err != nil {
		return config.ClientCommonConf{}, err
	}
	return cfg, err
}

func parseClientCommonCfgFromCmd() (cfg config.ClientCommonConf, err error) {
	cfg = config.GetDefaultClientConf()

	strs := strings.Split(serverAddr, ":")
	if len(strs) < 2 {
		err = fmt.Errorf("invalid server_addr")
		return
	}
	if strs[0] != "" {
		cfg.ServerAddr = strs[0]
	}
	cfg.ServerPort, err = strconv.Atoi(strs[1])
	if err != nil {
		err = fmt.Errorf("invalid server_addr")
		return
	}

	cfg.User = user
	cfg.Protocol = protocol
	cfg.LogLevel = logLevel
	cfg.LogFile = logFile
	cfg.LogMaxDays = int64(logMaxDays)
	if logFile == "console" {
		cfg.LogWay = "console"
	} else {
		cfg.LogWay = "file"
	}
	cfg.DisableLogColor = disableLogColor

	// Only token authentication is supported in cmd mode
	cfg.AuthClientConfig = auth.GetDefaultAuthClientConf()
	cfg.Token = token

	return
}

func runClient(cfgFilePath string) (err error) {
	var content string
	content, err = config.GetRenderedConfFromFile(cfgFilePath)
	if err != nil {
		return
	}

	cfg, err := parseClientCommonCfg(CfgFileTypeIni, content)
	if err != nil {
		return
	}

	pxyCfgs, visitorCfgs, err := config.LoadAllConfFromIni(cfg.User, content, cfg.Start)
	if err != nil {
		return err
	}

	err = startService(cfg, pxyCfgs, visitorCfgs, cfgFilePath)
	return
}

func startService(cfg config.ClientCommonConf, pxyCfgs map[string]config.ProxyConf, visitorCfgs map[string]config.VisitorConf, cfgFile string) (err error) {
	log.InitLog(cfg.LogWay, cfg.LogFile, cfg.LogLevel,
		cfg.LogMaxDays, cfg.DisableLogColor)

	if cfg.DnsServer != "" {
		s := cfg.DnsServer
		if !strings.Contains(s, ":") {
			s += ":53"
		}
		// Change default dns server for frpc
		net.DefaultResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return net.Dial("udp", s)
			},
		}
	}
	svr, errRet := client.NewService(cfg, pxyCfgs, visitorCfgs, cfgFile)
	if errRet != nil {
		err = errRet
		return
	}

	// Capture the exit signal if we use kcp.
	if cfg.Protocol == "kcp" {
		go handleSignal(svr)
	}

	err = svr.Run()
	if cfg.Protocol == "kcp" {
		<-kcpDoneCh
	}
	return
}

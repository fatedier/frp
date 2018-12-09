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
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	ini "github.com/vaughan0/go-ini"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/g"
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

	serverAddr string
	user       string
	protocol   string
	token      string
	logLevel   string
	logFile    string
	logMaxDays int

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
	serverName        string
	bindAddr          string
	bindPort          int
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "", "c", "./frpc.ini", "config file of frpc")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "version of frpc")
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
	os.Exit(0)
}

func parseClientCommonCfg(fileType int, filePath string) (err error) {
	if fileType == CfgFileTypeIni {
		err = parseClientCommonCfgFromIni(filePath)
	} else if fileType == CfgFileTypeCmd {
		err = parseClientCommonCfgFromCmd()
	}
	if err != nil {
		return
	}

	g.GlbClientCfg.CfgFile = cfgFile

	err = g.GlbClientCfg.ClientCommonConf.Check()
	if err != nil {
		return
	}
	return
}

func parseClientCommonCfgFromIni(filePath string) (err error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(b)

	cfg, err := config.UnmarshalClientConfFromIni(&g.GlbClientCfg.ClientCommonConf, content)
	if err != nil {
		return err
	}
	g.GlbClientCfg.ClientCommonConf = *cfg
	return
}

func parseClientCommonCfgFromCmd() (err error) {
	strs := strings.Split(serverAddr, ":")
	if len(strs) < 2 {
		err = fmt.Errorf("invalid server_addr")
		return
	}
	if strs[0] != "" {
		g.GlbClientCfg.ServerAddr = strs[0]
	}
	g.GlbClientCfg.ServerPort, err = strconv.Atoi(strs[1])
	if err != nil {
		err = fmt.Errorf("invalid server_addr")
		return
	}

	g.GlbClientCfg.User = user
	g.GlbClientCfg.Protocol = protocol
	g.GlbClientCfg.Token = token
	g.GlbClientCfg.LogLevel = logLevel
	g.GlbClientCfg.LogFile = logFile
	g.GlbClientCfg.LogMaxDays = int64(logMaxDays)
	if logFile == "console" {
		g.GlbClientCfg.LogWay = "console"
	} else {
		g.GlbClientCfg.LogWay = "file"
	}
	return nil
}

func runClient(cfgFilePath string) (err error) {
	err = parseClientCommonCfg(CfgFileTypeIni, cfgFilePath)
	if err != nil {
		return
	}

	conf, err := ini.LoadFile(cfgFilePath)
	if err != nil {
		return err
	}

	pxyCfgs, visitorCfgs, err := config.LoadAllConfFromIni(g.GlbClientCfg.User, conf, g.GlbClientCfg.Start)
	if err != nil {
		return err
	}

	err = startService(pxyCfgs, visitorCfgs)
	return
}

func startService(pxyCfgs map[string]config.ProxyConf, visitorCfgs map[string]config.VisitorConf) (err error) {
	log.InitLog(g.GlbClientCfg.LogWay, g.GlbClientCfg.LogFile, g.GlbClientCfg.LogLevel, g.GlbClientCfg.LogMaxDays)
	if g.GlbClientCfg.DnsServer != "" {
		s := g.GlbClientCfg.DnsServer
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
	svr := client.NewService(pxyCfgs, visitorCfgs)

	// Capture the exit signal if we use kcp.
	if g.GlbClientCfg.Protocol == "kcp" {
		go handleSignal(svr)
	}

	err = svr.Run()
	return
}

// Copyright 2020 guylewin, guy@lewin.co.il
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
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/consts"
)

func init() {
	tcpMuxCmd.PersistentFlags().StringVarP(&serverAddr, "server_addr", "s", "127.0.0.1:7000", "frp server's address")
	tcpMuxCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "user")
	tcpMuxCmd.PersistentFlags().StringVarP(&protocol, "protocol", "p", "tcp", "tcp or kcp or websocket")
	tcpMuxCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "auth token")
	tcpMuxCmd.PersistentFlags().StringVarP(&logLevel, "log_level", "", "info", "log level")
	tcpMuxCmd.PersistentFlags().StringVarP(&logFile, "log_file", "", "console", "console or file path")
	tcpMuxCmd.PersistentFlags().IntVarP(&logMaxDays, "log_max_days", "", 3, "log file reversed days")
	tcpMuxCmd.PersistentFlags().BoolVarP(&disableLogColor, "disable_log_color", "", false, "disable log color in console")

	tcpMuxCmd.PersistentFlags().StringVarP(&proxyName, "proxy_name", "n", "", "proxy name")
	tcpMuxCmd.PersistentFlags().StringVarP(&localIp, "local_ip", "i", "127.0.0.1", "local ip")
	tcpMuxCmd.PersistentFlags().IntVarP(&localPort, "local_port", "l", 0, "local port")
	tcpMuxCmd.PersistentFlags().StringVarP(&customDomains, "custom_domain", "d", "", "custom domain")
	tcpMuxCmd.PersistentFlags().StringVarP(&subDomain, "sd", "", "", "sub domain")
	tcpMuxCmd.PersistentFlags().StringVarP(&multiplexer, "mux", "", "", "multiplexer")
	tcpMuxCmd.PersistentFlags().BoolVarP(&useEncryption, "ue", "", false, "use encryption")
	tcpMuxCmd.PersistentFlags().BoolVarP(&useCompression, "uc", "", false, "use compression")

	rootCmd.AddCommand(tcpMuxCmd)
}

var tcpMuxCmd = &cobra.Command{
	Use:   "tcpmux",
	Short: "Run frpc with a single tcpmux proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientCfg, err := parseClientCommonCfg(CfgFileTypeCmd, "")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cfg := &config.TcpMuxProxyConf{}
		var prefix string
		if user != "" {
			prefix = user + "."
		}
		cfg.ProxyName = prefix + proxyName
		cfg.ProxyType = consts.TcpMuxProxy
		cfg.LocalIp = localIp
		cfg.LocalPort = localPort
		cfg.CustomDomains = strings.Split(customDomains, ",")
		cfg.SubDomain = subDomain
		cfg.Multiplexer = multiplexer
		cfg.UseEncryption = useEncryption
		cfg.UseCompression = useCompression

		err = cfg.CheckForCli()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		proxyConfs := map[string]config.ProxyConf{
			cfg.ProxyName: cfg,
		}
		err = startService(clientCfg, proxyConfs, nil, "")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return nil
	},
}

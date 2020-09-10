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
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/consts"
)

func init() {
	xtcpCmd.PersistentFlags().StringVarP(&serverAddr, "server_addr", "s", "127.0.0.1:7000", "frp server's address")
	xtcpCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "user")
	xtcpCmd.PersistentFlags().StringVarP(&protocol, "protocol", "p", "tcp", "tcp or kcp or websocket")
	xtcpCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "auth token")
	xtcpCmd.PersistentFlags().StringVarP(&logLevel, "log_level", "", "info", "log level")
	xtcpCmd.PersistentFlags().StringVarP(&logFile, "log_file", "", "console", "console or file path")
	xtcpCmd.PersistentFlags().IntVarP(&logMaxDays, "log_max_days", "", 3, "log file reversed days")
	xtcpCmd.PersistentFlags().BoolVarP(&disableLogColor, "disable_log_color", "", false, "disable log color in console")

	xtcpCmd.PersistentFlags().StringVarP(&proxyName, "proxy_name", "n", "", "proxy name")
	xtcpCmd.PersistentFlags().StringVarP(&role, "role", "", "server", "role")
	xtcpCmd.PersistentFlags().StringVarP(&sk, "sk", "", "", "secret key")
	xtcpCmd.PersistentFlags().StringVarP(&serverName, "server_name", "", "", "server name")
	xtcpCmd.PersistentFlags().StringVarP(&localIp, "local_ip", "i", "127.0.0.1", "local ip")
	xtcpCmd.PersistentFlags().IntVarP(&localPort, "local_port", "l", 0, "local port")
	xtcpCmd.PersistentFlags().StringVarP(&bindAddr, "bind_addr", "", "", "bind addr")
	xtcpCmd.PersistentFlags().IntVarP(&bindPort, "bind_port", "", 0, "bind port")
	xtcpCmd.PersistentFlags().BoolVarP(&useEncryption, "ue", "", false, "use encryption")
	xtcpCmd.PersistentFlags().BoolVarP(&useCompression, "uc", "", false, "use compression")

	rootCmd.AddCommand(xtcpCmd)
}

var xtcpCmd = &cobra.Command{
	Use:   "xtcp",
	Short: "Run frpc with a single xtcp proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientCfg, err := parseClientCommonCfg(CfgFileTypeCmd, "")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		proxyConfs := make(map[string]config.ProxyConf)
		visitorConfs := make(map[string]config.VisitorConf)

		var prefix string
		if user != "" {
			prefix = user + "."
		}

		if role == "server" {
			cfg := &config.XtcpProxyConf{}
			cfg.ProxyName = prefix + proxyName
			cfg.ProxyType = consts.XtcpProxy
			cfg.UseEncryption = useEncryption
			cfg.UseCompression = useCompression
			cfg.Role = role
			cfg.Sk = sk
			cfg.LocalIp = localIp
			cfg.LocalPort = localPort
			err = cfg.CheckForCli()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			proxyConfs[cfg.ProxyName] = cfg
		} else if role == "visitor" {
			cfg := &config.XtcpVisitorConf{}
			cfg.ProxyName = prefix + proxyName
			cfg.ProxyType = consts.XtcpProxy
			cfg.UseEncryption = useEncryption
			cfg.UseCompression = useCompression
			cfg.Role = role
			cfg.Sk = sk
			cfg.ServerName = serverName
			cfg.BindAddr = bindAddr
			cfg.BindPort = bindPort
			err = cfg.Check()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			visitorConfs[cfg.ProxyName] = cfg
		} else {
			fmt.Println("invalid role")
			os.Exit(1)
		}

		err = startService(clientCfg, proxyConfs, visitorConfs, "")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return nil
	},
}

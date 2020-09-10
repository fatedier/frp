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
	"strings"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/consts"
)

func init() {
	httpCmd.PersistentFlags().StringVarP(&serverAddr, "server_addr", "s", "127.0.0.1:7000", "frp server's address")
	httpCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "user")
	httpCmd.PersistentFlags().StringVarP(&protocol, "protocol", "p", "tcp", "tcp or kcp or websocket")
	httpCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "auth token")
	httpCmd.PersistentFlags().StringVarP(&logLevel, "log_level", "", "info", "log level")
	httpCmd.PersistentFlags().StringVarP(&logFile, "log_file", "", "console", "console or file path")
	httpCmd.PersistentFlags().IntVarP(&logMaxDays, "log_max_days", "", 3, "log file reversed days")
	httpCmd.PersistentFlags().BoolVarP(&disableLogColor, "disable_log_color", "", false, "disable log color in console")

	httpCmd.PersistentFlags().StringVarP(&proxyName, "proxy_name", "n", "", "proxy name")
	httpCmd.PersistentFlags().StringVarP(&localIp, "local_ip", "i", "127.0.0.1", "local ip")
	httpCmd.PersistentFlags().IntVarP(&localPort, "local_port", "l", 0, "local port")
	httpCmd.PersistentFlags().StringVarP(&customDomains, "custom_domain", "d", "", "custom domain")
	httpCmd.PersistentFlags().StringVarP(&subDomain, "sd", "", "", "sub domain")
	httpCmd.PersistentFlags().StringVarP(&locations, "locations", "", "", "locations")
	httpCmd.PersistentFlags().StringVarP(&httpUser, "http_user", "", "", "http auth user")
	httpCmd.PersistentFlags().StringVarP(&httpPwd, "http_pwd", "", "", "http auth password")
	httpCmd.PersistentFlags().StringVarP(&hostHeaderRewrite, "host_header_rewrite", "", "", "host header rewrite")
	httpCmd.PersistentFlags().BoolVarP(&useEncryption, "ue", "", false, "use encryption")
	httpCmd.PersistentFlags().BoolVarP(&useCompression, "uc", "", false, "use compression")

	rootCmd.AddCommand(httpCmd)
}

var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "Run frpc with a single http proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientCfg, err := parseClientCommonCfg(CfgFileTypeCmd, "")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cfg := &config.HttpProxyConf{}
		var prefix string
		if user != "" {
			prefix = user + "."
		}
		cfg.ProxyName = prefix + proxyName
		cfg.ProxyType = consts.HttpProxy
		cfg.LocalIp = localIp
		cfg.LocalPort = localPort
		cfg.CustomDomains = strings.Split(customDomains, ",")
		cfg.SubDomain = subDomain
		cfg.Locations = strings.Split(locations, ",")
		cfg.HttpUser = httpUser
		cfg.HttpPwd = httpPwd
		cfg.HostHeaderRewrite = hostHeaderRewrite
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

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

	"github.com/fatedier/frp/pkg/config/types"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/consts"
)

func init() {
	RegisterCommonFlags(udpCmd)

	udpCmd.PersistentFlags().StringVarP(&proxyName, "proxy_name", "n", "", "proxy name")
	udpCmd.PersistentFlags().StringVarP(&localIP, "local_ip", "i", "127.0.0.1", "local ip")
	udpCmd.PersistentFlags().IntVarP(&localPort, "local_port", "l", 0, "local port")
	udpCmd.PersistentFlags().IntVarP(&remotePort, "remote_port", "r", 0, "remote port")
	udpCmd.PersistentFlags().BoolVarP(&useEncryption, "ue", "", false, "use encryption")
	udpCmd.PersistentFlags().BoolVarP(&useCompression, "uc", "", false, "use compression")
	udpCmd.PersistentFlags().StringVarP(&bandwidthLimit, "bandwidth_limit", "", "", "bandwidth limit")
	udpCmd.PersistentFlags().StringVarP(&bandwidthLimitMode, "bandwidth_limit_mode", "", types.BandwidthLimitModeClient, "bandwidth limit mode")

	rootCmd.AddCommand(udpCmd)
}

var udpCmd = &cobra.Command{
	Use:   "udp",
	Short: "Run frpc with a single udp proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientCfg, err := parseClientCommonCfgFromCmd()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cfg := &v1.UDPProxyConfig{}
		var prefix string
		if user != "" {
			prefix = user + "."
		}
		cfg.Name = prefix + proxyName
		cfg.Type = consts.UDPProxy
		cfg.LocalIP = localIP
		cfg.LocalPort = localPort
		cfg.RemotePort = remotePort
		cfg.Transport.UseEncryption = useEncryption
		cfg.Transport.UseCompression = useCompression
		cfg.Transport.BandwidthLimit, err = types.NewBandwidthQuantity(bandwidthLimit)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		cfg.Transport.BandwidthLimitMode = bandwidthLimitMode

		if err := validation.ValidateProxyConfigurerForClient(cfg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = startService(clientCfg, []v1.ProxyConfigurer{cfg}, nil, "")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return nil
	},
}

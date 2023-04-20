// Copyright 2023 The frp Authors
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
	"net"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/nathole"
)

var (
	natHoleSTUNServer string
	serverUDPPort     int
)

func init() {
	RegisterCommonFlags(natholeCmd)

	rootCmd.AddCommand(natholeCmd)
	natholeCmd.AddCommand(natholeDiscoveryCmd)

	natholeCmd.PersistentFlags().StringVarP(&natHoleSTUNServer, "nat_hole_stun_server", "", "stun.easyvoip.com:3478", "STUN server address for nathole")
	natholeCmd.PersistentFlags().IntVarP(&serverUDPPort, "server_udp_port", "", 0, "UDP port of frps for nathole")
}

var natholeCmd = &cobra.Command{
	Use:   "nathole",
	Short: "Actions about nathole",
}

var natholeDiscoveryCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover nathole information by frps and stun server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// ignore error here, because we can use command line pameters
		cfg, _, _, _ := config.ParseClientConfig(cfgFile)
		if natHoleSTUNServer != "" {
			cfg.NatHoleSTUNServer = natHoleSTUNServer
		}
		if serverUDPPort != 0 {
			cfg.ServerUDPPort = serverUDPPort
		}

		if err := validateForNatHoleDiscovery(cfg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		serverAddr := ""
		if cfg.ServerUDPPort != 0 {
			serverAddr = net.JoinHostPort(cfg.ServerAddr, strconv.Itoa(cfg.ServerUDPPort))
		}
		addresses, err := nathole.Discover(
			serverAddr,
			[]string{cfg.NatHoleSTUNServer},
			[]byte(cfg.Token),
		)
		if err != nil {
			fmt.Println("discover error:", err)
			os.Exit(1)
		}
		if len(addresses) < 2 {
			fmt.Printf("discover error: can not get enough addresses, need 2, got: %v\n", addresses)
			os.Exit(1)
		}

		natType, behavior, err := nathole.ClassifyNATType(addresses)
		if err != nil {
			fmt.Println("classify nat type error:", err)
			os.Exit(1)
		}
		fmt.Println("Your NAT type is:", natType)
		fmt.Println("Behavior is:", behavior)
		fmt.Println("External address is:", addresses)
		return nil
	},
}

func validateForNatHoleDiscovery(cfg config.ClientCommonConf) error {
	if cfg.NatHoleSTUNServer == "" {
		return fmt.Errorf("nat_hole_stun_server can not be empty")
	}
	return nil
}

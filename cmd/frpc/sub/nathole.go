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

func init() {
	RegisterCommonFlags(natholeCmd)

	rootCmd.AddCommand(natholeCmd)
	natholeCmd.AddCommand(natholeDiscoveryCmd)
}

var natholeCmd = &cobra.Command{
	Use:   "nathole",
	Short: "Actions about nathole",
}

var natholeDiscoveryCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover nathole information by frps and stun server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, _, err := config.ParseClientConfig(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := validateForNatHoleDiscovery(cfg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		addresses, err := nathole.Discover(
			net.JoinHostPort(cfg.ServerAddr, strconv.Itoa(cfg.ServerUDPPort)),
			[]string{cfg.NatHoleSTUNServer},
			[]byte(cfg.Token),
		)
		if err != nil {
			fmt.Println("discover error:", err)
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
	if cfg.ServerUDPPort == 0 {
		return fmt.Errorf("server udp port can not be empty")
	}
	return nil
}

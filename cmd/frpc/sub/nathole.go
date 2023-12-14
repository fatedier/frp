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
	"os"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/nathole"
)

var (
	natHoleSTUNServer string
	natHoleLocalAddr  string
)

func init() {
	rootCmd.AddCommand(natholeCmd)
	natholeCmd.AddCommand(natholeDiscoveryCmd)

	natholeCmd.PersistentFlags().StringVarP(&natHoleSTUNServer, "nat_hole_stun_server", "", "", "STUN server address for nathole")
	natholeCmd.PersistentFlags().StringVarP(&natHoleLocalAddr, "nat_hole_local_addr", "l", "", "local address to connect STUN server")
}

var natholeCmd = &cobra.Command{
	Use:   "nathole",
	Short: "Actions about nathole",
}

var natholeDiscoveryCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover nathole information from stun server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// ignore error here, because we can use command line pameters
		cfg, _, _, _, err := config.LoadClientConfig(cfgFile, strictConfigMode)
		if err != nil {
			cfg = &v1.ClientCommonConfig{}
			cfg.Complete()
		}
		if natHoleSTUNServer != "" {
			cfg.NatHoleSTUNServer = natHoleSTUNServer
		}

		if err := validateForNatHoleDiscovery(cfg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		addrs, localAddr, err := nathole.Discover([]string{cfg.NatHoleSTUNServer}, natHoleLocalAddr)
		if err != nil {
			fmt.Println("discover error:", err)
			os.Exit(1)
		}
		if len(addrs) < 2 {
			fmt.Printf("discover error: can not get enough addresses, need 2, got: %v\n", addrs)
			os.Exit(1)
		}

		localIPs, _ := nathole.ListLocalIPsForNatHole(10)

		natFeature, err := nathole.ClassifyNATFeature(addrs, localIPs)
		if err != nil {
			fmt.Println("classify nat feature error:", err)
			os.Exit(1)
		}
		fmt.Println("STUN server:", cfg.NatHoleSTUNServer)
		fmt.Println("Your NAT type is:", natFeature.NatType)
		fmt.Println("Behavior is:", natFeature.Behavior)
		fmt.Println("External address is:", addrs)
		fmt.Println("Local address is:", localAddr.String())
		fmt.Println("Public Network:", natFeature.PublicNetwork)
		return nil
	},
}

func validateForNatHoleDiscovery(cfg *v1.ClientCommonConfig) error {
	if cfg.NatHoleSTUNServer == "" {
		return fmt.Errorf("nat_hole_stun_server can not be empty")
	}
	return nil
}

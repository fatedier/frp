// Copyright 2021 The frp Authors
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
)

func init() {
	rootCmd.AddCommand(verifyCmd)
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify that the configures is valid",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, _, err := config.ParseClientConfig(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("frpc: the configuration file %s syntax is ok\n", cfgFile)
		return nil
	},
}

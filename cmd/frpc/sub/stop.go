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
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/pkg/config"
)

func init() {
	rootCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running frpc",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, _, err := config.ParseClientConfig(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = stopClient(cfg)
		if err != nil {
			fmt.Printf("frpc stop error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("stop success\n")
		return nil
	},
}

func stopClient(clientCfg config.ClientCommonConf) error {
	if clientCfg.AdminPort == 0 {
		return fmt.Errorf("admin_port shoud be set if you want to use stop feature")
	}

	req, err := http.NewRequest("POST", "http://"+
		clientCfg.AdminAddr+":"+fmt.Sprintf("%d", clientCfg.AdminPort)+"/api/stop", nil)
	if err != nil {
		return err
	}

	authStr := "Basic " + base64.StdEncoding.EncodeToString([]byte(clientCfg.AdminUser+":"+
		clientCfg.AdminPwd))

	req.Header.Add("Authorization", authStr)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("code [%d], %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Overview of all proxies status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, _, _, err := config.LoadClientConfig(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err = status(cfg); err != nil {
			fmt.Printf("frpc get status error: %v\n", err)
			os.Exit(1)
		}
		return nil
	},
}

func status(clientCfg *v1.ClientCommonConfig) error {
	if clientCfg.WebServer.Port == 0 {
		return fmt.Errorf("the port of web server shoud be set if you want to get proxy status")
	}

	req, err := http.NewRequest("GET", "http://"+
		clientCfg.WebServer.Addr+":"+fmt.Sprintf("%d", clientCfg.WebServer.Port)+"/api/status", nil)
	if err != nil {
		return err
	}

	authStr := "Basic " + base64.StdEncoding.EncodeToString(
		[]byte(clientCfg.WebServer.User+":"+clientCfg.WebServer.Password))

	req.Header.Add("Authorization", authStr)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("admin api status code [%d]", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	res := make(client.StatusResp)
	err = json.Unmarshal(body, &res)
	if err != nil {
		return fmt.Errorf("unmarshal http response error: %s", strings.TrimSpace(string(body)))
	}

	fmt.Println("Proxy Status...")
	types := []string{"tcp", "udp", "tcpmux", "http", "https", "stcp", "sudp", "xtcp"}
	for _, pxyType := range types {
		arrs := res[pxyType]
		if len(arrs) == 0 {
			continue
		}

		fmt.Println(strings.ToUpper(pxyType))
		tbl := table.New("Name", "Status", "LocalAddr", "Plugin", "RemoteAddr", "Error")
		for _, ps := range arrs {
			tbl.AddRow(ps.Name, ps.Status, ps.LocalAddr, ps.Plugin, ps.RemoteAddr, ps.Err)
		}
		tbl.Print()
		fmt.Println("")
	}
	return nil
}

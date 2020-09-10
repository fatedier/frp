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
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/models/config"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Overview of all proxies status",
	RunE: func(cmd *cobra.Command, args []string) error {
		iniContent, err := config.GetRenderedConfFromFile(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		clientCfg, err := parseClientCommonCfg(CfgFileTypeIni, iniContent)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = status(clientCfg)
		if err != nil {
			fmt.Printf("frpc get status error: %v\n", err)
			os.Exit(1)
		}
		return nil
	},
}

func status(clientCfg config.ClientCommonConf) error {
	if clientCfg.AdminPort == 0 {
		return fmt.Errorf("admin_port shoud be set if you want to get proxy status")
	}

	req, err := http.NewRequest("GET", "http://"+
		clientCfg.AdminAddr+":"+fmt.Sprintf("%d", clientCfg.AdminPort)+"/api/status", nil)
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

	if resp.StatusCode != 200 {
		return fmt.Errorf("admin api status code [%d]", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	res := &client.StatusResp{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return fmt.Errorf("unmarshal http response error: %s", strings.TrimSpace(string(body)))
	}

	fmt.Println("Proxy Status...")
	if len(res.Tcp) > 0 {
		fmt.Printf("TCP")
		tbl := table.New("Name", "Status", "LocalAddr", "Plugin", "RemoteAddr", "Error")
		for _, ps := range res.Tcp {
			tbl.AddRow(ps.Name, ps.Status, ps.LocalAddr, ps.Plugin, ps.RemoteAddr, ps.Err)
		}
		tbl.Print()
		fmt.Println("")
	}
	if len(res.Udp) > 0 {
		fmt.Printf("UDP")
		tbl := table.New("Name", "Status", "LocalAddr", "Plugin", "RemoteAddr", "Error")
		for _, ps := range res.Udp {
			tbl.AddRow(ps.Name, ps.Status, ps.LocalAddr, ps.Plugin, ps.RemoteAddr, ps.Err)
		}
		tbl.Print()
		fmt.Println("")
	}
	if len(res.Http) > 0 {
		fmt.Printf("HTTP")
		tbl := table.New("Name", "Status", "LocalAddr", "Plugin", "RemoteAddr", "Error")
		for _, ps := range res.Http {
			tbl.AddRow(ps.Name, ps.Status, ps.LocalAddr, ps.Plugin, ps.RemoteAddr, ps.Err)
		}
		tbl.Print()
		fmt.Println("")
	}
	if len(res.Https) > 0 {
		fmt.Printf("HTTPS")
		tbl := table.New("Name", "Status", "LocalAddr", "Plugin", "RemoteAddr", "Error")
		for _, ps := range res.Https {
			tbl.AddRow(ps.Name, ps.Status, ps.LocalAddr, ps.Plugin, ps.RemoteAddr, ps.Err)
		}
		tbl.Print()
		fmt.Println("")
	}
	if len(res.Stcp) > 0 {
		fmt.Printf("STCP")
		tbl := table.New("Name", "Status", "LocalAddr", "Plugin", "RemoteAddr", "Error")
		for _, ps := range res.Stcp {
			tbl.AddRow(ps.Name, ps.Status, ps.LocalAddr, ps.Plugin, ps.RemoteAddr, ps.Err)
		}
		tbl.Print()
		fmt.Println("")
	}
	if len(res.Xtcp) > 0 {
		fmt.Printf("XTCP")
		tbl := table.New("Name", "Status", "LocalAddr", "Plugin", "RemoteAddr", "Error")
		for _, ps := range res.Xtcp {
			tbl.AddRow(ps.Name, ps.Status, ps.LocalAddr, ps.Plugin, ps.RemoteAddr, ps.Err)
		}
		tbl.Print()
		fmt.Println("")
	}

	return nil
}

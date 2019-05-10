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

package main

import (
	"fmt"
	"os"
	"path"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"

	"github.com/fatedier/frp/g"
	"github.com/fatedier/frp/models/config"
)

var (
	srvName  string
	srvDName string
	srvDesc  string
	srv      service.Service
)

func init() {
	rootCmd.PersistentFlags().StringVar(&srvName, "name", "frps", "Service name")
	rootCmd.PersistentFlags().StringVar(&srvDName, "display_name", "frps", "Service display name")
	rootCmd.PersistentFlags().StringVar(&srvDesc, "description", "frps service", "Service description")

	if !service.Interactive() {
		srvCmd.RunE = func(cmd *cobra.Command, args []string) error {
			return srv.Run()
		}
	}

	srvCmd.AddCommand(installSrvCmd, uninstallSrvCmd, startSrvCmd, stopSrvCmd, restartSrvCmd)
	rootCmd.AddCommand(srvCmd)
}

var srvCmd = &cobra.Command{
	Use:     "service",
	Aliases: []string{"srv"},
	Short:   "Control frps system service",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		srv, err = service.New(&serviceFRP{}, &service.Config{
			Name:        srvName,
			DisplayName: srvDName,
			Description: srvDesc,
		})
		return err
	},
}

var installSrvCmd = &cobra.Command{
	Use:   "install",
	Short: "install frps service",
	RunE:  srvAction("install"),
}

var uninstallSrvCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "uninstall frps service",
	RunE:  srvAction("uninstall"),
}

var startSrvCmd = &cobra.Command{
	Use:   "start",
	Short: "start frps service",
	RunE:  srvAction("start"),
}

var stopSrvCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop frps service",
	RunE:  srvAction("stop"),
}

var restartSrvCmd = &cobra.Command{
	Use:   "restart",
	Short: "restart frps service",
	RunE:  srvAction("restart"),
}

type serviceFRP struct{}

func (sf serviceFRP) Start(s service.Service) error {
	var (
		content string
		err     error
	)
	envCfgFile := path.Join(os.Getenv("FRP_HOME"), "frps.ini")
	content, err = config.GetRenderedConfFromFile(envCfgFile)
	if err != nil {
		return err
	}
	g.GlbServerCfg.CfgFile = cfgFile
	err = parseServerCommonCfg(CfgFileTypeIni, content)
	if err != nil {
		return err
	}
	go func(s service.Service) {
		err = runServer()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}(s)
	return nil
}

func (sf serviceFRP) Stop(s service.Service) error {
	os.Exit(0)
	return nil
}

func srvAction(act string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := service.Control(srv, act); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return nil
	}
}

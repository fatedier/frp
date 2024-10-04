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
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/frp/server"
)

var (
	cfgFile          string
	showVersion      bool
	strictConfigMode bool

	serverCfg v1.ServerConfig
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file of frps")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "version of frps")
	rootCmd.PersistentFlags().BoolVarP(&strictConfigMode, "strict_config", "", true, "strict config parsing mode, unknown fields will cause errors")

	config.RegisterServerConfigFlags(rootCmd, &serverCfg)
}

var rootCmd = &cobra.Command{
	Use:   "frps",
	Short: "frps is the server of frp (https://github.com/fatedier/frp)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version.Full())
			return nil
		}

		var (
			svrCfg         *v1.ServerConfig
			isLegacyFormat bool
			err            error
		)
		if cfgFile != "" {
			svrCfg, isLegacyFormat, err = config.LoadServerConfig(cfgFile, strictConfigMode)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if isLegacyFormat {
				fmt.Printf("警告：INI文件格式在未来将会不受支持并且移除（反正目前HayFrps支持就行，这行别管）!\n")
			}
		} else {
			serverCfg.Complete()
			svrCfg = &serverCfg
		}

		warning, err := validation.ValidateServerConfig(svrCfg)
		if warning != nil {
			fmt.Printf("警告: %v\n", warning)
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := runServer(svrCfg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return nil
	},
}

func Execute() {
	rootCmd.SetGlobalNormalizationFunc(config.WordSepNormalizeFunc)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServer(cfg *v1.ServerConfig) (err error) {
	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)
	log.Infof("[HayFrp] 欢迎使用全新的HayFrp服务端!")
	log.Infof("[HayFrp] 服务端版本：" + version.Full() + "!")
	log.Infof("[HayFrp] 本新版本在客户端链接时将介入HayFrp终端!")
	log.Infof("[HayFrp] 各种链接协议已升级到现代协议！")

	// 发起 GET 请求获取 API 返回的内容(节点名称)
	resp, err := http.Get("https://api.hayfrp.org/NodeAPI?type=GetNodeName&token=" + cfg.ApiToken)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 读取 API 返回的内容
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// 将 API 返回的内容添加到 loginMsg.RunId 后面
	log.Infof("[HayFrp] 欢迎节点" + string(body) + "上线!")
	log.Infof("[HayFrp] 自检所有必要项中，请稍等......")
	log.Infof("[HayFrp] HayFrp API启用状态: %s", strconv.FormatBool(cfg.EnableApi))
	log.Infof("[HayFrp] HayFrp API URL: %s", cfg.ApiBaseUrl)
	log.Infof("[HayFrp] HayFrp API Node Key: %s", cfg.ApiToken)

	if cfgFile != "" {
		log.Infof("[HayFrp] HayFrp服务端已引用配置文件: %s", cfgFile)
	} else {
		log.Infof("[HayFrp] HayFrp服务端已引用命令行参数")
	}

	svr, err := server.NewService(cfg)
	if err != nil {
		return err
	}
	log.Infof("[HayFrp] HayFrp服务端已成功启动！")
	go checkonline(cfg)
	svr.Run(context.Background())
	return
}

func checkonline(cfg *v1.ServerConfig) {
	time.Sleep(2 * time.Second) // 延时2秒，确保延时函数有足够的时间运行
	log.Infof("[HayFrp] 检测到所有端口已成功启动，请稍等......")
	log.Infof("[HayFrp] 即将请求HayFrp API授权本节点调用其他节点检查本节点状态......")
	log.Infof("[HayFrp] 检测节点在线状态中，这将会更新云端状态......")
	// 发起 GET 请求获取 API 返回的内容(节点状态)
	resp, err := http.Get("https://api.hayfrp.org/NodeAPI?type=checkonline&token=" + cfg.ApiToken)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// 读取 API 返回的内容
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	// 将 API 返回的内容添加到 loginMsg.RunId 后面
	log.Infof("[HayFrp] " + string(body))
}

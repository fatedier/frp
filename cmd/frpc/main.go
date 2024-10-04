// Copyright 2016 fatedier, fatedier@gmail.com
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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	_ "github.com/fatedier/frp/assets/frpc"
	"github.com/fatedier/frp/cmd/frpc/sub"
	"github.com/fatedier/frp/pkg/util/system"
	"github.com/fatedier/frp/pkg/util/version"
)

type Config struct {
	Type    string `json:"type"`
	Format  string `json:"format"`
	Csrf    string `json:"csrf"`
	ProxyID string `json:"proxy"`
}

type Response struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func main() {
	fmt.Println(`
 __    __                      ________                   
|  \  |  \                    |        \                  
| $$  | $$  ______   __    __ | $$$$$$$$______    ______  
| $$__| $$ |      \ |  \  |  \| $$__   /      \  /      \ 
| $$    $$  \$$$$$$\| $$  | $$| $$  \ |  $$$$$$\|  $$$$$$\
| $$$$$$$$ /      $$| $$  | $$| $$$$$ | $$   \$$| $$  | $$
| $$  | $$|  $$$$$$$| $$__/ $$| $$    | $$      | $$__/ $$
| $$  | $$ \$$    $$ \$$    $$| $$    | $$      | $$    $$
 \$$   \$$  \$$$$$$$ _\$$$$$$$ \$$     \$$      | $$$$$$$ 
                    |  \__| $$                  | $$      
                     \$$    $$                  | $$      
                      \$$$$$$                    \$$               ——— HayFrp公益项目运营&开发组
					  
	`)
	fmt.Println("欢迎使用HayFrp！")
	fmt.Println("HayFrp程序发行版本：" + version.Full())
	// 获取命令行参数
	args := os.Args[1:]
	// 检查参数数量
	for _, arg := range args {
		if arg == "-m" {
			fmt.Println("[HayFrpOH] 捕捉到配置命令，模式已自动切换为HayFrpOH")

			// 解析参数
			var id, csrf string
			var isS bool
			for i, arg := range args {
				switch arg {
				case "-m":
					if i+1 < len(args) {
						csrf = args[i+1]
					}
				case "-h":
					isS = true
				}
			}

			// 检查是否提供了id和csrf
			if csrf == "" {
				fmt.Println("[HayFrpOH] 请提供有效的CSRF")
				return
			}

			// 构建请求体
			config := Config{
				Type:    "config",
				Csrf:    csrf,
				ProxyID: id, // 如果id不为空，则添加到请求体中
			}

			// 如果是-h操作，则设置format为toml
			if isS {
				config.Format = "toml"
				fmt.Println("[HayFrpOH] HayFrpOH设定模式：发起端.")
			} else {
				fmt.Println("[HayFrpOH] HayFrpOH设定模式：参与端.")
			}

			jsonData, err := json.Marshal(config)
			if err != nil {
				fmt.Println("[HayFrpOH] JSON编码错误:", err)
				return
			}

			fmt.Println("[HayFrpOH] 拉取配置文件......")
			// 发送POST请求
			client := &http.Client{}
			req, err := http.NewRequest("POST", "https://api.hayfrp.org/p2p", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Println("[HayFrpOH] 创建请求错误:", err)
				return
			}

			// 设置请求头
			req.Header.Set("waf", "off")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("[HayFrpOH] 请求错误，请尝试使用加速器:", err)
				return
			}
			defer resp.Body.Close()

			fmt.Println("[HayFrpOH] 拉取成功.")
			fmt.Println("[HayFrpOH] 解析配置文件......")
			fmt.Println("[HayFrpOH] 成功，即将启动FRP进行X/S TCP")
			// 读取响应
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("[HayFrpOH] 读取响应错误:", err)
				return
			}

			// 解析响应的JSON内容
			var response Response
			err = json.Unmarshal(body, &response)
			if err != nil {
				fmt.Println("[HayFrpOH] 解析响应错误:", err)
				return
			}

			// 检查响应状态
			if response.Status != 200 {
				fmt.Println("[HayFrpOH] 请求失败:", response.Message)
				return
			}

			// 解析data字段
			data := response.Data.(map[string]interface{})
			// 获取当前文件的绝对路径
			exePath, err := os.Executable()
			if err != nil {
				fmt.Println("Error getting executable path:", err)
				return
			}
			exeDir := filepath.Dir(exePath)
			programName := filepath.Base(os.Args[0])
			goPath := filepath.Join(exeDir, programName)
			os.Args = []string{}
			// 构造启动命令
			cmd := exec.Command(goPath, data["proxy_type"].(string), "visitor", "-s", data["hostname"].(string), "-P", data["port"].(string), "-t", "ConnectHayFrpTokenWelcomToUseOurCloudServiceDonttellyourtokentotheotherpeoplesbecauseyouwilllostyouraccount", "-u", data["token"].(string), "--sk", data["sk"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "--server-name", data["proxy_name"].(string), "-n", data["proxy_name"].(string)+"_visitor", "--bind-addr", "127.0.0.1", "--bind-port", data["local_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
			if runtime.GOOS != "windows" {
				cmd = exec.Command(goPath, data["proxy_type"].(string), "visitor", "-s", data["hostname"].(string), "-P", data["port"].(string), "-t", "ConnectHayFrpTokenWelcomToUseOurCloudServiceDonttellyourtokentotheotherpeoplesbecauseyouwilllostyouraccount", "-u", data["token"].(string), "--sk", data["sk"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "--server-name", data["proxy_name"].(string), "-n", data["proxy_name"].(string)+"_visitor", "--bind-addr", "127.0.0.1", "--bind-port", data["local_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
			}

			// 检查是否需要添加-s参数
			if isS {
				// 构造启动命令
				cmd = exec.Command(goPath, data["proxy_type"].(string), "-s", data["hostname"].(string), "-P", data["port"].(string), "-t", "ConnectHayFrpTokenWelcomToUseOurCloudServiceDonttellyourtokentotheotherpeoplesbecauseyouwilllostyouraccount", "-u", data["token"].(string), "--sk", data["sk"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "--proxy-name", data["proxy_name"].(string), "-i", data["local_ip"].(string), "-l", data["local_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
				if runtime.GOOS != "windows" {
					cmd = exec.Command(goPath, data["proxy_type"].(string), "-s", data["hostname"].(string), "-P", data["port"].(string), "-t", "ConnectHayFrpTokenWelcomToUseOurCloudServiceDonttellyourtokentotheotherpeoplesbecauseyouwilllostyouraccount", "-u", data["token"].(string), "--sk", data["sk"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "--proxy-name", data["proxy_name"].(string), "-i", data["local_ip"].(string), "-l", data["local_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
				}
			}

			// 获取命令的输出和错误信息
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				fmt.Println("[HayFrpOH] 获取命令输出错误:", err)
				return
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				fmt.Println("[HayFrpOH] 获取命令错误输出错误:", err)
				return
			}

			// 启动命令
			err = cmd.Start()
			if err != nil {
				fmt.Println("[HayFrpOH] 启动命令错误:", err)
				return
			}

			// 读取输出和错误信息
			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := stdout.Read(buf)
					if n > 0 {
						fmt.Print(string(buf[:n]))
					}
					if err != nil {
						break
					}
				}
			}()

			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := stderr.Read(buf)
					if n > 0 {
						fmt.Print(string(buf[:n]))
					}
					if err != nil {
						break
					}
				}
			}()

			// 等待命令完成
			err = cmd.Wait()
			if err != nil {
				fmt.Println("[HayFrpOH] 命令执行错误:", err)
				return
			}

			fmt.Println("[HayFrpOH] 任务被结束.")
			os.Exit(0)
		}

		if arg == "-esirun" {
			fmt.Println("[HayFrpEasyRun] 捕捉到配置命令，模式已自动切换为快速启动.")

			// 解析参数
			var id, csrf string
			for i, arg := range args {
				switch arg {
				case "-esirun":
					if i+1 < len(args) {
						csrf = args[i+1]
					}
				}
			}

			// 检查是否提供了id和csrf
			if csrf == "" {
				fmt.Println("[HayFrpEasyRun] 请提供有效的CSRF")
				return
			}

			// 构建请求体
			config := Config{
				Type:    "config",
				Csrf:    csrf,
				ProxyID: id, // 如果id不为空，则添加到请求体中
			}

			jsonData, err := json.Marshal(config)
			if err != nil {
				fmt.Println("[HayFrpEasyRun] JSON编码错误:", err)
				return
			}

			fmt.Println("[HayFrpEasyRun] 拉取配置文件......")
			// 发送POST请求
			client := &http.Client{}
			req, err := http.NewRequest("POST", "https://api.hayfrp.org/esirun", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Println("[HayFrpEasyRun] 创建请求错误:", err)
				return
			}

			// 设置请求头
			req.Header.Set("waf", "off")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("[HayFrpEasyRun] 请求错误，请尝试使用加速器:", err)
				return
			}
			defer resp.Body.Close()

			fmt.Println("[HayFrpEasyRun] 拉取成功.")
			fmt.Println("[HayFrpEasyRun] 解析配置文件......")
			fmt.Println("[HayFrpEasyRun] 成功，即将启动FRP")
			// 读取响应
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("[HayFrpEasyRun] 读取响应错误:", err)
				return
			}

			// 解析响应的JSON内容
			var response Response
			err = json.Unmarshal(body, &response)
			if err != nil {
				fmt.Println("[HayFrpEasyRun] 解析响应错误:", err)
				return
			}

			// 检查响应状态
			if response.Status != 200 {
				fmt.Println("[HayFrpEasyRun] 请求失败:", response.Message)
				return
			}

			// 解析data字段
			data := response.Data.(map[string]interface{})
			// 获取当前文件的绝对路径
			exePath, err := os.Executable()
			if err != nil {
				fmt.Println("Error getting executable path:", err)
				return
			}
			exeDir := filepath.Dir(exePath)
			programName := filepath.Base(os.Args[0])
			goPath := filepath.Join(exeDir, programName)
			os.Args = []string{}
			// 构造启动命令
			cmd := exec.Command(goPath, data["proxy_type"].(string), "-s", data["hostname"].(string), "-P", data["port"].(string), "-t", "ConnectHayFrpTokenWelcomToUseOurCloudServiceDonttellyourtokentotheotherpeoplesbecauseyouwilllostyouraccount", "-u", data["token"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "--proxy-name", data["proxy_name"].(string), "-i", data["local_ip"].(string), "-l", data["local_port"].(string), "-r", data["remote_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
			if runtime.GOOS != "windows" {
				cmd = exec.Command(goPath, data["proxy_type"].(string), "-s", data["hostname"].(string), "-P", data["port"].(string), "-t", "ConnectHayFrpTokenWelcomToUseOurCloudServiceDonttellyourtokentotheotherpeoplesbecauseyouwilllostyouraccount", "-u", data["token"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "--proxy-name", data["proxy_name"].(string), "-i", data["local_ip"].(string), "-l", data["local_port"].(string), "-r", data["remote_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
			}

			// 获取命令的输出和错误信息
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				fmt.Println("[HayFrpEasyRun] 获取命令输出错误:", err)
				return
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				fmt.Println("[HayFrpEasyRun] 获取命令错误输出错误:", err)
				return
			}

			// 启动命令
			err = cmd.Start()
			if err != nil {
				fmt.Println("[HayFrpEasyRun] 启动命令错误:", err)
				return
			}

			// 读取输出和错误信息
			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := stdout.Read(buf)
					if n > 0 {
						fmt.Print(string(buf[:n]))
					}
					if err != nil {
						break
					}
				}
			}()

			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := stderr.Read(buf)
					if n > 0 {
						fmt.Print(string(buf[:n]))
					}
					if err != nil {
						break
					}
				}
			}()

			// 等待命令完成
			err = cmd.Wait()
			if err != nil {
				fmt.Println("[HayFrpEasyRun] 命令执行错误:", err)
				return
			}

			fmt.Println("[HayFrpEasyRun] 任务被结束.")
			os.Exit(0)
		}
	}

	system.EnableCompatibilityMode()
	sub.Execute()

}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	_ "github.com/fatedier/frp/assets/frpc"
	"github.com/fatedier/frp/cmd/frpc/sub"
	"github.com/fatedier/frp/pkg/util/system"
	"github.com/fatedier/frp/pkg/util/version"
)

type Config struct {
	Type     string `json:"type"`
	Format   string `json:"format"`
	Csrf     string `json:"csrf"`
	ProxyID  string `json:"proxy"`
	Device   string `json:"device"`
	System   string `json:"system"`
	Hostname string `json:"hostname"`
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
			// 检测是否自定义API地址
			api := "https://api.hayfrp.com/" // 默认API地址
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

				case "-i":
					if i+1 < len(args) {
						api = args[i+1]
					}
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
			var req *http.Request
			req, err = http.NewRequest("POST", api+"p2p", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Println("[HayFrpOH] 创建请求错误:", err)
				return
			}

			// 设置请求头
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "HayFrpClient/114514")
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
			cmd := exec.Command(goPath, data["proxy_type"].(string), "visitor", "-s", data["ip"].(string), "-P", data["port"].(string), "-t", data["key"].(string), "-u", data["token"].(string), "--sk", data["sk"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "--server-name", data["proxy_name"].(string), "-n", data["proxy_name"].(string)+"_visitor", "--bind-addr", "127.0.0.1", "--bind-port", data["remote_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
			if runtime.GOOS != "windows" {
				cmd = exec.Command(goPath, data["proxy_type"].(string), "visitor", "-s", data["ip"].(string), "-P", data["port"].(string), "-t", data["key"].(string), "-u", data["token"].(string), "--sk", data["sk"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "--server-name", data["proxy_name"].(string), "-n", data["proxy_name"].(string)+"_visitor", "--bind-addr", "127.0.0.1", "--bind-port", data["remote_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
			}

			// 检查是否需要添加-s参数
			if isS {
				// 构造启动命令
				cmd = exec.Command(goPath, data["proxy_type"].(string), "-s", data["ip"].(string), "-P", data["port"].(string), "-t", data["key"].(string), "-u", data["token"].(string), "--sk", data["sk"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "--proxy-name", data["proxy_name"].(string), "-i", data["local_ip"].(string), "-l", data["local_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
				if runtime.GOOS != "windows" {
					cmd = exec.Command(goPath, data["proxy_type"].(string), "-s", data["ip"].(string), "-P", data["port"].(string), "-t", data["key"].(string), "-u", data["token"].(string), "--sk", data["sk"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "--proxy-name", data["proxy_name"].(string), "-i", data["local_ip"].(string), "-l", data["local_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
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
			// 检测是否自定义API地址
			api := "https://api.hayfrp.com/"
			// 解析参数
			var id, csrf string
			for i, arg := range args {
				switch arg {
				case "-esirun":
					if i+1 < len(args) {
						csrf = args[i+1]
					}

				case "-i":
					if i+1 < len(args) {
						api = args[i+1]
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
			var req *http.Request
			req, err = http.NewRequest("POST", api+"esirun", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Println("[HayFrpEasyRun] 创建请求错误:", err)
				return
			}

			// 设置请求头
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "HayFrpClient/114514")
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

			cmd := exec.Command(goPath, data["proxy_type"].(string), "-s", data["ip"].(string), "-P", data["port"].(string), "-t", data["key"].(string), "-u", data["token"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "-n", data["proxy_name"].(string), "-i", data["local_ip"].(string), "-l", data["local_port"].(string), "-r", data["remote_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
			if data["proxy_type"].(string) == "http" || data["proxy_type"].(string) == "https" {
				cmd = exec.Command(goPath, data["proxy_type"].(string), "-s", data["ip"].(string), "-P", data["port"].(string), "-t", data["key"].(string), "-u", data["token"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "-n", data["proxy_name"].(string), "-i", data["local_ip"].(string), "-l", data["local_port"].(string), "-d", data["domain"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
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

		// 云控部分

		if arg == "-rwmc" {
			// 检测是否自定义API地址
			api := "https://api.hayfrp.com/"
			// 解析参数
			var device string
			for i, arg := range args {
				switch arg {
				case "-rwmc":
					if i+1 < len(args) {
						device = args[i+1]
					}

				case "-i":
					if i+1 < len(args) {
						api = args[i+1]
					}
				}
			}

			// 检查是否提供了id和csrf
			if device == "" {
				fmt.Println("[HayFrpRemoteControl] 未识别到设备标识符，正在请求云端获取设备标识符......")
				// 构建请求体
				config := Config{
					Type: "active",
				}

				jsonData, err := json.Marshal(config)
				if err != nil {
					fmt.Println("[HayFrpRemoteControl] JSON编码错误:", err)
					return
				}
				// 发送POST请求
				client := &http.Client{}
				var req *http.Request
				req, err = http.NewRequest("POST", api+"rwmc", bytes.NewBuffer(jsonData))
				if err != nil {
					fmt.Println("[HayFrpRemoteControl] 创建请求错误:", err)
					return
				}

				// 设置请求头
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("User-Agent", "HayFrpClient/114514")
				req.Header.Set("waf", "off")

				resp, err := client.Do(req)
				if err != nil {
					fmt.Println("[HayFrpRemoteControl] 请求错误，请尝试使用加速器:", err)
					return
				}
				defer resp.Body.Close()

				fmt.Println("[HayFrpRemoteControl] 获取标识符成功.")
				fmt.Println("[HayFrpRemoteControl] 构建设备请求......")
				// 读取响应
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Println("[HayFrpRemoteControl] 读取响应错误:", err)
					return
				}

				// 解析响应的JSON内容
				var data map[string]interface{}
				err = json.Unmarshal([]byte(body), &data)
				if err != nil {
					fmt.Println("[HayFrpRemoteControl] 解析响应错误:", err)
					return
				}

				device = data["device"].(string)
			}

			fmt.Println("[HayFrpRemoteControl] 已激活云控，您可以在控制台控制本设备.")
			hostname, err := os.Hostname()
			if err != nil {
				fmt.Println("Error getting hostname:", err)
				return
			}
			for {
				// 构建请求体
				config := Config{
					Type:     "active",
					Device:   device,
					Hostname: hostname,
					System:   runtime.GOOS,
				}

				jsonData, err := json.Marshal(config)
				if err != nil {
					fmt.Println("[HayFrpRemoteControl] JSON编码错误:", err)
					return
				}

				// 发送POST请求
				client := &http.Client{}
				var req *http.Request
				req, err = http.NewRequest("POST", api+"rwmc", bytes.NewBuffer(jsonData))
				if err != nil {
					fmt.Println("[HayFrpRemoteControl] 创建请求错误:", err)
					return
				}

				// 设置请求头
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("User-Agent", "HayFrpClient/114514")
				req.Header.Set("waf", "off")

				resp, err := client.Do(req)
				if err != nil {
					fmt.Println("[HayFrpRemoteControl] 请求错误，请尝试使用加速器:", err)
					return
				}
				defer resp.Body.Close()
				// 读取响应
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Println("[HayFrpRemoteControl] 读取响应错误:", err)
					return
				}

				// 解析响应的JSON内容
				var data map[string]interface{}
				err = json.Unmarshal(body, &data)
				if err != nil {
					fmt.Println("[HayFrpRemoteControl] 解析响应错误:", err)
					return
				}

				if data["status"].(float64) != 200 {
					fmt.Println("[HayFrpRemoteControl]", data["message"])
					if data["action"] == "runcmd" {
						fmt.Println("[HayFrpRemoteControl] 执行命令:", data["cmd"])
						cmd := exec.Command("cmd", "/C", data["cmd"].(string))
						if runtime.GOOS != "windows" {
							cmd = exec.Command("sh", "-c", data["cmd"].(string))
						}
						go func() {
							// 获取标准输出和标准错误输出
							stdout, err := cmd.StdoutPipe()
							if err != nil {
								log.Fatal(err)
							}
							stderr, err := cmd.StderrPipe()
							if err != nil {
								log.Fatal(err)
							}

							// 启动命令
							if err := cmd.Start(); err != nil {
								log.Fatal(err)
							}

							// 将标准输出和标准错误输出复制到标准输出
							go io.Copy(os.Stdout, stdout)
							go io.Copy(os.Stdout, stderr)

							// 等待命令完成
							if err := cmd.Wait(); err != nil {
								fmt.Println("[HayFrpRemoteControl] 命令执行出错:", err)
							} else {
								fmt.Println("[HayFrpRemoteControl] 命令执行成功.")
							}
						}()

					}

					if data["action"] == "close" {
						fmt.Println("[HayFrpRemoteControl] 控制台已关闭连接，程序即将退出.")
						os.Exit(0)
					}

					if data["action"] == "run" {
						fmt.Println("[HayFrpRemoteControl] 收到启动隧道操作，即将使用EsiRun启动隧道.")
						// 构建请求体
						go func() {
							config := Config{
								Type: "config",
								Csrf: data["cmd"].(string),
							}

							jsonData, err := json.Marshal(config)
							if err != nil {
								fmt.Println("[HayFrpEasyRun] JSON编码错误:", err)
								return
							}

							fmt.Println("[HayFrpEasyRun] 拉取配置文件......")
							// 发送POST请求
							client := &http.Client{}
							var req *http.Request
							req, err = http.NewRequest("POST", api+"esirun", bytes.NewBuffer(jsonData))
							if err != nil {
								fmt.Println("[HayFrpEasyRun] 创建请求错误:", err)
								return
							}

							// 设置请求头
							req.Header.Set("Content-Type", "application/json")
							req.Header.Set("User-Agent", "HayFrpClient/114514")
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

							// 构造启动命令
							var cmd *exec.Cmd
							if data["proxy_type"].(string) == "http" || data["proxy_type"].(string) == "https" {
								cmd = exec.Command(goPath, data["proxy_type"].(string), "-s", data["ip"].(string), "-P", data["port"].(string), "-t", data["key"].(string), "-u", data["token"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "-n", data["proxy_name"].(string), "-i", data["local_ip"].(string), "-l", data["local_port"].(string), "-d", data["domain"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
							} else {
								cmd = exec.Command(goPath, data["proxy_type"].(string), "-s", data["ip"].(string), "-P", data["port"].(string), "-t", data["key"].(string), "-u", data["token"].(string), "--tls-enable", "false", "-p", "tcp", "--dns-server", "223.5.5.5", "-n", data["proxy_name"].(string), "-i", data["local_ip"].(string), "-l", data["local_port"].(string), "-r", data["remote_port"].(string), "--uc", data["use_compression"].(string), "--ue", data["use_encryption"].(string))
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
								cmd.Process.Kill()
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
						}()
					}
				}

				time.Sleep(time.Second * 3)
			}
		}
	}

	system.EnableCompatibilityMode()
	sub.Execute()

}

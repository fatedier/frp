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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	docopt "github.com/docopt/docopt-go"

	"github.com/fatedier/frp/src/assets"
	"github.com/fatedier/frp/src/models/server"
	"github.com/fatedier/frp/src/utils/conn"
	"github.com/fatedier/frp/src/utils/log"
	"github.com/fatedier/frp/src/utils/version"
	"github.com/fatedier/frp/src/utils/vhost"
)

var usage string = `frps is the server of frp

Usage: 
    frps [-c config_file] [-L log_file] [--log-level=<log_level>] [--addr=<bind_addr>]
    frps [-c config_file] --reload
    frps -h | --help
    frps -v | --version

Options:
    -c config_file            set config file
    -L log_file               set output log file, including console
    --log-level=<log_level>   set log level: debug, info, warn, error
    --addr=<bind_addr>        listen addr for client, example: 0.0.0.0:7000
    --reload                  reload ini file and configures in common section won't be changed
    -h --help                 show this screen
    -v --version              show version
`

func main() {
	// the configures parsed from file will be replaced by those from command line if exist
	args, err := docopt.Parse(usage, nil, true, version.Full(), false)

	if args["-c"] != nil {
		server.ConfigFile = args["-c"].(string)
	}
	err = server.LoadConf(server.ConfigFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	// reload check
	if args["--reload"] != nil {
		if args["--reload"].(bool) {
			req, err := http.NewRequest("GET", "http://"+server.BindAddr+":"+fmt.Sprintf("%d", server.DashboardPort)+"/api/reload", nil)
			if err != nil {
				fmt.Printf("frps reload error: %v\n", err)
				os.Exit(1)
			}

			authStr := "Basic " + base64.StdEncoding.EncodeToString([]byte(server.DashboardUsername+":"+server.DashboardPassword))

			req.Header.Add("Authorization", authStr)
			defaultClient := &http.Client{}
			resp, err := defaultClient.Do(req)

			if err != nil {
				fmt.Printf("frps reload error: %v\n", err)
				os.Exit(1)
			} else {
				defer resp.Body.Close()
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Printf("frps reload error: %v\n", err)
					os.Exit(1)
				}
				res := &server.GeneralResponse{}
				err = json.Unmarshal(body, &res)
				if err != nil {
					fmt.Printf("http response error: %v\n", err)
					os.Exit(1)
				} else if res.Code != 0 {
					fmt.Printf("reload error: %s\n", res.Msg)
					os.Exit(1)
				}
				fmt.Printf("reload success\n")
				os.Exit(0)
			}
		}
	}

	if args["-L"] != nil {
		if args["-L"].(string) == "console" {
			server.LogWay = "console"
		} else {
			server.LogWay = "file"
			server.LogFile = args["-L"].(string)
		}
	}

	if args["--log-level"] != nil {
		server.LogLevel = args["--log-level"].(string)
	}

	if args["--addr"] != nil {
		addr := strings.Split(args["--addr"].(string), ":")
		if len(addr) != 2 {
			fmt.Println("--addr format error: example 0.0.0.0:7000")
			os.Exit(1)
		}
		bindPort, err := strconv.ParseInt(addr[1], 10, 64)
		if err != nil {
			fmt.Println("--addr format error, example 0.0.0.0:7000")
			os.Exit(1)
		}
		server.BindAddr = addr[0]
		server.BindPort = bindPort
	}

	if args["-v"] != nil {
		if args["-v"].(bool) {
			fmt.Println(version.Full())
			os.Exit(0)
		}
	}

	log.InitLog(server.LogWay, server.LogFile, server.LogLevel, server.LogMaxDays)

	// init assets
	err = assets.Load(server.AssetsDir)
	if err != nil {
		log.Error("Load assets error: %v", err)
		os.Exit(1)
	}

	l, err := conn.Listen(server.BindAddr, server.BindPort)
	if err != nil {
		log.Error("Create server listener error, %v", err)
		os.Exit(1)
	}

	// create vhost if VhostHttpPort != 0
	if server.VhostHttpPort != 0 {
		vhostListener, err := conn.Listen(server.BindAddr, server.VhostHttpPort)
		if err != nil {
			log.Error("Create vhost http listener error, %v", err)
			os.Exit(1)
		}
		server.VhostHttpMuxer, err = vhost.NewHttpMuxer(vhostListener, 30*time.Second)
		if err != nil {
			log.Error("Create vhost httpMuxer error, %v", err)
		}
	}

	// create vhost if VhostHttpPort != 0
	if server.VhostHttpsPort != 0 {
		vhostListener, err := conn.Listen(server.BindAddr, server.VhostHttpsPort)
		if err != nil {
			log.Error("Create vhost https listener error, %v", err)
			os.Exit(1)
		}
		server.VhostHttpsMuxer, err = vhost.NewHttpsMuxer(vhostListener, 30*time.Second)
		if err != nil {
			log.Error("Create vhost httpsMuxer error, %v", err)
		}
	}

	// create dashboard web server if DashboardPort is set, so it won't be 0
	if server.DashboardPort != 0 {
		err := server.RunDashboardServer(server.BindAddr, server.DashboardPort)
		if err != nil {
			log.Error("Create dashboard web server error, %v", err)
			os.Exit(1)
		}
	}

	log.Info("Start frps success")
	if server.PrivilegeMode == true {
		log.Info("PrivilegeMode is enabled, you should pay more attention to security issues")
	}
	ProcessControlConn(l)
}

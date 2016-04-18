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
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	docopt "github.com/docopt/docopt-go"

	"frp/models/server"
	"frp/utils/conn"
	"frp/utils/log"
	"frp/utils/version"
	"frp/utils/vhost"
)

var (
	configFile string = "./frps.ini"
)

var usage string = `frps is the server of frp

Usage: 
	frps [-c config_file] [-L log_file] [--log-level=<log_level>] [--addr=<bind_addr>]
	frps -h | --help | --version

Options:
	-c config_file            set config file
	-L log_file               set output log file, including console
	--log-level=<log_level>   set log level: debug, info, warn, error
	--addr=<bind_addr>        listen addr for client, example: 0.0.0.0:7000
	-h --help                 show this screen
	--version                 show version
`

func main() {
	// the configures parsed from file will be replaced by those from command line if exist
	args, err := docopt.Parse(usage, nil, true, version.Full(), false)

	if args["-c"] != nil {
		configFile = args["-c"].(string)
	}
	err = server.LoadConf(configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
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

	log.InitLog(server.LogWay, server.LogFile, server.LogLevel)

	l, err := conn.Listen(server.BindAddr, server.BindPort)
	if err != nil {
		log.Error("Create server listener error, %v", err)
		os.Exit(1)
	}

	if server.VhostHttpPort != 0 {
		vhostListener, err := conn.Listen(server.BindAddr, server.VhostHttpPort)
		if err != nil {
			log.Error("Create vhost http listener error, %v", err)
			os.Exit(1)
		}
		server.VhostMuxer, err = vhost.NewHttpMuxer(vhostListener, 30*time.Second)
		if err != nil {
			log.Error("Create vhost httpMuxer error, %v", err)
		}
	}

	log.Info("Start frps success")
	ProcessControlConn(l)
}

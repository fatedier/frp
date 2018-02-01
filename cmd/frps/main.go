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

	docopt "github.com/docopt/docopt-go"
	ini "github.com/vaughan0/go-ini"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/server"
	"github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/version"
)

var usage string = `frps is the server of frp

Usage: 
    frps [-c config_file] [-L log_file] [--log-level=<log_level>] [--addr=<bind_addr>]
    frps -h | --help
    frps -v | --version

Options:
    -c config_file            set config file
    -L log_file               set output log file, including console
    --log-level=<log_level>   set log level: debug, info, warn, error
    --addr=<bind_addr>        listen addr for client, example: 0.0.0.0:7000
    -h --help                 show this screen
    -v --version              show version
`

func main() {
	var err error
	confFile := "./frps.ini"
	// the configures parsed from file will be replaced by those from command line if exist
	args, err := docopt.Parse(usage, nil, true, version.Full(), false)

	if args["-c"] != nil {
		confFile = args["-c"].(string)
	}

	conf, err := ini.LoadFile(confFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	config.ServerCommonCfg, err = config.LoadServerCommonConf(conf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if args["-L"] != nil {
		if args["-L"].(string) == "console" {
			config.ServerCommonCfg.LogWay = "console"
		} else {
			config.ServerCommonCfg.LogWay = "file"
			config.ServerCommonCfg.LogFile = args["-L"].(string)
		}
	}

	if args["--log-level"] != nil {
		config.ServerCommonCfg.LogLevel = args["--log-level"].(string)
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
		config.ServerCommonCfg.BindAddr = addr[0]
		config.ServerCommonCfg.BindPort = int(bindPort)
	}

	if args["-v"] != nil {
		if args["-v"].(bool) {
			fmt.Println(version.Full())
			os.Exit(0)
		}
	}

	log.InitLog(config.ServerCommonCfg.LogWay, config.ServerCommonCfg.LogFile,
		config.ServerCommonCfg.LogLevel, config.ServerCommonCfg.LogMaxDays)

	svr, err := server.NewService()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log.Info("Start frps success")
	if config.ServerCommonCfg.PrivilegeMode == true {
		log.Info("PrivilegeMode is enabled, you should pay more attention to security issues")
	}
	server.ServerService = svr
	svr.Run()
}

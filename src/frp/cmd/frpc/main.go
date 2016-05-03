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
	"sync"

	docopt "github.com/docopt/docopt-go"

	"frp/models/client"
	"frp/utils/log"
	"frp/utils/version"
)

var (
	configFile string = "./frpc.ini"
)

var usage string = `frpc is the client of frp

Usage: 
    frpc [-c config_file] [-L log_file] [--log-level=<log_level>] [--server-addr=<server_addr>]
    frpc -h | --help | --version

Options:
    -c config_file              set config file
    -L log_file                 set output log file, including console
    --log-level=<log_level>     set log level: debug, info, warn, error
    --server-addr=<server_addr> addr which frps is listening for, example: 0.0.0.0:7000
    -h --help                   show this screen
    --version                   show version
`

func main() {
	// the configures parsed from file will be replaced by those from command line if exist
	args, err := docopt.Parse(usage, nil, true, version.Full(), false)

	if args["-c"] != nil {
		configFile = args["-c"].(string)
	}
	err = client.LoadConf(configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	if args["-L"] != nil {
		if args["-L"].(string) == "console" {
			client.LogWay = "console"
		} else {
			client.LogWay = "file"
			client.LogFile = args["-L"].(string)
		}
	}

	if args["--log-level"] != nil {
		client.LogLevel = args["--log-level"].(string)
	}

	if args["--server-addr"] != nil {
		addr := strings.Split(args["--server-addr"].(string), ":")
		if len(addr) != 2 {
			fmt.Println("--server-addr format error: example 0.0.0.0:7000")
			os.Exit(1)
		}
		serverPort, err := strconv.ParseInt(addr[1], 10, 64)
		if err != nil {
			fmt.Println("--server-addr format error, example 0.0.0.0:7000")
			os.Exit(1)
		}
		client.ServerAddr = addr[0]
		client.ServerPort = serverPort
	}

	log.InitLog(client.LogWay, client.LogFile, client.LogLevel, client.LogMaxDays)

	// wait until all control goroutine exit
	var wait sync.WaitGroup
	wait.Add(len(client.ProxyClients))

	for _, client := range client.ProxyClients {
		go ControlProcess(client, &wait)
	}

	log.Info("Start frpc success")

	wait.Wait()
	log.Warn("All proxy exit!")
}

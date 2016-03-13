package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	docopt "github.com/docopt/docopt-go"

	"frp/models/server"
	"frp/utils/conn"
	"frp/utils/log"
	"frp/utils/version"
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
		log.Error("Create listener error, %v", err)
		os.Exit(-1)
	}

	log.Info("Start frps success")
	ProcessControlConn(l)
}

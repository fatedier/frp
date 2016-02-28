package main

import (
	"os"

	"frp/models/server"
	"frp/utils/conn"
	"frp/utils/log"
)

func main() {
	err := server.LoadConf("./frps.ini")
	if err != nil {
		os.Exit(-1)
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

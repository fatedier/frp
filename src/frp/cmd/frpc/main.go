package main

import (
	"os"
	"sync"

	"frp/models/client"
	"frp/utils/log"
)

func main() {
	err := client.LoadConf("./frpc.ini")
	if err != nil {
		os.Exit(-1)
	}

	log.InitLog(client.LogWay, client.LogFile, client.LogLevel)

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

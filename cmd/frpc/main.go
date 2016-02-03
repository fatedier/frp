package main

import (
	"os"
	"sync"
	
	"github.com/fatedier/frp/pkg/utils/log"
)

func main() {
	err := LoadConf("./frpc.ini")
	if err != nil {
		os.Exit(-1)
	}

	log.InitLog(LogWay, LogFile, LogLevel)

	// wait until all control goroutine exit
	var wait sync.WaitGroup
	wait.Add(len(ProxyClients))

	for _, client := range ProxyClients {
		go ControlProcess(client, &wait)
	}

	log.Info("Start frpc success")

	wait.Wait()
	log.Warn("All proxy exit!")
}

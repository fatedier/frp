package proxy

import (
	"os"

	"github.com/fatedier/frp/utils/log"
)

var cnfg Config

func init() {
	// 加载配置文件
	err := cnfg.GetConfig("../config/config.json")
	if err != nil {
		log.Error("can not load config file:%v\n", err)
		os.Exit(-1)
	}

	setLog()
}

func setLog() {
	log.SetLogLevel("debug")
}

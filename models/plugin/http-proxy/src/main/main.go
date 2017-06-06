package main

import (
	"proxy"

	"github.com/fatedier/frp/utils/log"
)

func main() {
	ps := proxy.NewProxyServer()

	log.Info("begin proxy")
	log.Error("proxy exit %v", ps.ListenAndServe())
}

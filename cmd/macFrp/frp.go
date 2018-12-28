package main

/*
 */
import "C"

import (
	"github.com/fatedier/frp/cmd/frpc/sub"
	"github.com/fatedier/frp/cmd/frps/frps"
	"github.com/fatedier/frp/utils/version"
)

//export StopFrpc
func StopFrpc() C.int {
	if err := sub.StopFrp(); err != nil {
		println(err.Error())
		return C.int(0)
	}
	return C.int(1)
}

//export IsFrpcRunning
func IsFrpcRunning() bool {
	return sub.IsFrpRunning()
}

//export StopFrps
func StopFrps() C.int {
	if err := frps.StopFrps(); err != nil {
		println(err.Error())
		return C.int(0)
	}
	return C.int(1)
}

//export IsFrpsRunning
func IsFrpsRunning() bool {
	return frps.IsFrpsRunning()
}

//export Version
func Version() string {
	return version.Full()
}

func main() {
	// frpsPath := os.Args[2]
	// RunFrps(C.CString(frpsPath))
	// TestFrps(C.CString(frpsPath))
	c := make(chan bool)
	<-c
}

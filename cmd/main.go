package main

import (
	"runtime"
	"strings"

	_ "net/http/pprof"

	"github.com/fatedier/frp/cmd/frp"
)

func main() {
	_, path, _, _ := runtime.Caller(0)
	path = path[0:strings.LastIndex(path, "/")]
	println(path)
	frp.RunFrps(path + "/frps.ini")

	ch := make(chan bool)
	// go func() {
	// 	time.Sleep(time.Second * 5)
	// 	frp.RunFrpc(path + "/frpc.ini")
	// 	ch <- true
	// }()
	// <-ch

	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:10000", nil))
	// }()
	// // frp.RunFrps("./frps.ini")
	// log.Println("frps is running: ", frp.IsFrpsRunning())

	// //frp.StopFrps()
	// //log.Println("frps is running: ", frp.IsFrpsRunning())
	// ch = make(chan bool)
	// go func() {
	// 	time.Sleep(time.Second * 5)
	// 	log.Println("frpc is running: ", frp.IsFrpcRunning())
	// 	frp.StopFrpc()
	// 	log.Println("frpc is running: ", frp.IsFrpcRunning())
	// 	ch <- true
	// }()
	<-ch
}

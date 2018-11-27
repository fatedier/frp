package main

import (
	"log"
	"time"

	"github.com/fatedier/frp/cmd/frp"
)

func main() {
	frp.RunFrps("./frps.ini")

	ch := make(chan bool)
	go func() {
		time.Sleep(time.Second * 5)
		frp.StopFrps()
		ch <- true
	}()
	<-ch

	// frp.RunFrps("./frps.ini")
	log.Println(frp.IsFrpsRunning())
	ch = make(chan bool)
	<-ch
}

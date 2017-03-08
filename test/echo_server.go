package main

import (
	"bufio"
	"fmt"
	"io"

	"github.com/fatedier/frp/utils/net"
)

var (
	PORT int64 = 10701
)

func main() {
	l, err := net.ListenTcp("127.0.0.1", PORT)
	if err != nil {
		fmt.Printf("echo server listen error: %v\n", err)
		return
	}

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Printf("echo server accept error: %v\n", err)
			return
		}

		go echoWorker(c)
	}
}

func echoWorker(c net.Conn) {
	br := bufio.NewReader(c)
	for {
		buf, err := br.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("echo server read error: %v\n", err)
			return
		}

		c.Write([]byte(buf + "\n"))
	}
}

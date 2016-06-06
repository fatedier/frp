package main

import (
	"fmt"
	"io"

	"frp/utils/conn"
)

var (
	PORT int64 = 10701
)

func main() {
	l, err := conn.Listen("0.0.0.0", PORT)
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

func echoWorker(c *conn.Conn) {
	for {
		buff, err := c.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("echo server read error: %v\n", err)
			return
		}

		c.Write(buff)
	}
}

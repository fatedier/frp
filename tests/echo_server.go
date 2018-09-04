package tests

import (
	"fmt"
	"io"
	"net"
	"os"
	"syscall"

	frpNet "github.com/fatedier/frp/utils/net"
)

func StartTcpEchoServer() {
	l, err := frpNet.ListenTcp("127.0.0.1", TEST_TCP_PORT)
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

func StartUdpEchoServer() {
	l, err := frpNet.ListenUDP("127.0.0.1", TEST_UDP_PORT)
	if err != nil {
		fmt.Printf("udp echo server listen error: %v\n", err)
		return
	}

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Printf("udp echo server accept error: %v\n", err)
			return
		}

		go echoWorker(c)
	}
}

func StartUnixDomainServer() {
	unixPath := TEST_UNIX_DOMAIN_ADDR
	os.Remove(unixPath)
	syscall.Umask(0)
	l, err := net.Listen("unix", unixPath)
	if err != nil {
		fmt.Printf("unix domain server listen error: %v\n", err)
		return
	}

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Printf("unix domain server accept error: %v\n", err)
			return
		}

		go echoWorker(c)
	}
}

func echoWorker(c net.Conn) {
	buf := make([]byte, 2048)

	for {
		n, err := c.Read(buf)
		if err != nil {
			if err == io.EOF {
				c.Close()
				break
			} else {
				fmt.Printf("echo server read error: %v\n", err)
				return
			}
		}

		c.Write(buf[:n])
	}
}

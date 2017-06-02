package tests

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"

	frpNet "github.com/fatedier/frp/utils/net"
)

func StartEchoServer() {
	l, err := frpNet.ListenTcp("127.0.0.1", 10701)
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
	l, err := frpNet.ListenUDP("127.0.0.1", 10703)
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
	unixPath := "/tmp/frp_echo_server.sock"
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

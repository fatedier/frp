package mock

import (
	"fmt"
	"io"
	"net"
	"os"
	"syscall"

	frpNet "github.com/fatedier/frp/utils/net"
)

func StartTcpEchoServer(port int) {
	l, err := frpNet.ListenTcp("127.0.0.1", port)
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

func StartTcpEchoServer2(port int) {
	l, err := frpNet.ListenTcp("127.0.0.1", port)
	if err != nil {
		fmt.Printf("echo server2 listen error: %v\n", err)
		return
	}

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Printf("echo server2 accept error: %v\n", err)
			return
		}

		go echoWorker2(c)
	}
}

func StartUdpEchoServer(port int) {
	l, err := frpNet.ListenUDP("127.0.0.1", port)
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

func StartUnixDomainServer(unixPath string) {
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

func echoWorker2(c net.Conn) {
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

		var w []byte
		w = append(w, buf[:n]...)
		w = append(w, buf[:n]...)
		c.Write(w)
	}
}

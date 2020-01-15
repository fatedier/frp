package mock

import (
	"fmt"
	"io"
	"net"
	"os"
	"syscall"

	frpNet "github.com/fatedier/frp/utils/net"
)

type EchoServer struct {
	l net.Listener

	port        int
	repeatedNum int
	specifyStr  string
}

func NewEchoServer(port int, repeatedNum int, specifyStr string) *EchoServer {
	if repeatedNum <= 0 {
		repeatedNum = 1
	}
	return &EchoServer{
		port:        port,
		repeatedNum: repeatedNum,
		specifyStr:  specifyStr,
	}
}

func (es *EchoServer) Start() error {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", es.port))
	if err != nil {
		fmt.Printf("echo server listen error: %v\n", err)
		return err
	}
	es.l = l

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}

			go echoWorker(c, es.repeatedNum, es.specifyStr)
		}
	}()
	return nil
}

func (es *EchoServer) Stop() {
	es.l.Close()
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

		go echoWorker(c, 1, "")
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

		go echoWorker(c, 1, "")
	}
}

func echoWorker(c net.Conn, repeatedNum int, specifyStr string) {
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

		if specifyStr != "" {
			c.Write([]byte(specifyStr))
		} else {
			var w []byte
			for i := 0; i < repeatedNum; i++ {
				w = append(w, buf[:n]...)
			}
			c.Write(w)
		}
	}
}

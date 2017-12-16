package test

import (
	"fmt"
	"net"
	"time"

	frpNet "github.com/fatedier/frp/utils/net"
)

func sendTcpMsg(addr string, msg string) (res string, err error) {
	c, err := frpNet.ConnectTcpServer(addr)
	defer c.Close()
	if err != nil {
		err = fmt.Errorf("connect to tcp server error: %v", err)
		return
	}

	timer := time.Now().Add(5 * time.Second)
	c.SetDeadline(timer)
	c.Write([]byte(msg))

	buf := make([]byte, 2048)
	n, errRet := c.Read(buf)
	if errRet != nil {
		err = fmt.Errorf("read from tcp server error: %v", errRet)
		return
	}
	return string(buf[:n]), nil
}

func sendUdpMsg(addr string, msg string) (res string, err error) {
	udpAddr, errRet := net.ResolveUDPAddr("udp", addr)
	if errRet != nil {
		err = fmt.Errorf("resolve udp addr error: %v", err)
		return
	}
	conn, errRet := net.DialUDP("udp", nil, udpAddr)
	if errRet != nil {
		err = fmt.Errorf("dial udp server error: %v", err)
		return
	}
	defer conn.Close()
	_, err = conn.Write([]byte(msg))
	if err != nil {
		err = fmt.Errorf("write to udp server error: %v", err)
		return
	}

	buf := make([]byte, 2048)
	n, errRet := conn.Read(buf)
	if errRet != nil {
		err = fmt.Errorf("read from udp server error: %v", err)
		return
	}
	return string(buf[:n]), nil
}

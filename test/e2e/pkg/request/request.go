package request

import (
	"fmt"
	"net"
	"time"
)

func SendTCPRequest(port int, content []byte, timeout time.Duration) (res string, err error) {
	c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		err = fmt.Errorf("connect to tcp server error: %v", err)
		return
	}
	defer c.Close()

	c.SetDeadline(time.Now().Add(timeout))
	return sendTCPRequestByConn(c, content)
}

func sendTCPRequestByConn(c net.Conn, content []byte) (res string, err error) {
	c.Write(content)

	buf := make([]byte, 2048)
	n, errRet := c.Read(buf)
	if errRet != nil {
		err = fmt.Errorf("read from tcp error: %v", errRet)
		return
	}
	return string(buf[:n]), nil
}

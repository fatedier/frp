package tests

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	frpNet "github.com/fatedier/frp/utils/net"
)

func sendTcpMsg(addr string, msg string) (res string, err error) {
	c, err := frpNet.ConnectTcpServer(addr)
	if err != nil {
		err = fmt.Errorf("connect to tcp server error: %v", err)
		return
	}
	defer c.Close()

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

func sendHttpMsg(method, url string, host string, header map[string]string) (code int, body string, err error) {
	req, errRet := http.NewRequest(method, url, nil)
	if errRet != nil {
		err = errRet
		return
	}

	if host != "" {
		req.Host = host
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, errRet := http.DefaultClient.Do(req)
	if errRet != nil {
		err = errRet
		return
	}
	code = resp.StatusCode
	buf, errRet := ioutil.ReadAll(resp.Body)
	if errRet != nil {
		err = errRet
		return
	}
	body = string(buf)
	return
}

func basicAuth(username, passwd string) string {
	auth := username + ":" + passwd
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

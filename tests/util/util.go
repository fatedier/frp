package util

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fatedier/frp/client"
)

func GetProxyStatus(statusAddr string, user string, passwd string, name string) (status *client.ProxyStatusResp, err error) {
	req, err := http.NewRequest("GET", "http://"+statusAddr+"/api/status", nil)
	if err != nil {
		return status, err
	}

	authStr := "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+passwd))
	req.Header.Add("Authorization", authStr)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return status, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return status, fmt.Errorf("admin api status code [%d]", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return status, err
	}
	allStatus := &client.StatusResp{}
	err = json.Unmarshal(body, &allStatus)
	if err != nil {
		return status, fmt.Errorf("unmarshal http response error: %s", strings.TrimSpace(string(body)))
	}
	for _, s := range allStatus.Tcp {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.Udp {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.Http {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.Https {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.Stcp {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.Xtcp {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.Sudp {
		if s.Name == name {
			return &s, nil
		}
	}

	return status, errors.New("no proxy status found")
}

func ReloadConf(reloadAddr string, user string, passwd string) error {
	req, err := http.NewRequest("GET", "http://"+reloadAddr+"/api/reload", nil)
	if err != nil {
		return err
	}

	authStr := "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+passwd))
	req.Header.Add("Authorization", authStr)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("admin api status code [%d]", resp.StatusCode)
	}
	io.Copy(ioutil.Discard, resp.Body)
	return nil
}

func SendTcpMsg(addr string, msg string) (res string, err error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		err = fmt.Errorf("connect to tcp server error: %v", err)
		return
	}
	defer c.Close()
	return SendTcpMsgByConn(c, msg)
}

func SendTcpMsgByConn(c net.Conn, msg string) (res string, err error) {
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

func SendUdpMsg(addr string, msg string) (res string, err error) {
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

func SendHttpMsg(method, urlStr string, host string, headers map[string]string, proxy string) (code int, body string, header http.Header, err error) {
	req, errRet := http.NewRequest(method, urlStr, nil)
	if errRet != nil {
		err = errRet
		return
	}

	if host != "" {
		req.Host = host
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if len(proxy) != 0 {
		tr.Proxy = func(req *http.Request) (*url.URL, error) {
			return url.Parse(proxy)
		}
	}
	client := http.Client{
		Transport: tr,
	}

	resp, errRet := client.Do(req)
	if errRet != nil {
		err = errRet
		return
	}
	code = resp.StatusCode
	header = resp.Header
	buf, errRet := ioutil.ReadAll(resp.Body)
	if errRet != nil {
		err = errRet
		return
	}
	body = string(buf)
	return
}

func BasicAuth(username, passwd string) string {
	auth := username + ":" + passwd
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

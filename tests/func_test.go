package tests

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/stretchr/testify/assert"
)

var (
	TEST_STR                    = "frp is a fast reverse proxy to help you expose a local server behind a NAT or firewall to the internet."
	TEST_TCP_PORT        int64  = 10701
	TEST_TCP_FRP_PORT    int64  = 10801
	TEST_TCP_EC_FRP_PORT int64  = 10901
	TEST_TCP_ECHO_STR    string = "tcp type:" + TEST_STR

	TEST_UDP_PORT     int64  = 10702
	TEST_UDP_FRP_PORT int64  = 10802
	TEST_UDP_ECHO_STR string = "udp type:" + TEST_STR

	TEST_UNIX_DOMAIN_ADDR     string = "/tmp/frp_echo_server.sock"
	TEST_UNIX_DOMAIN_FRP_PORT int64  = 10803
	TEST_UNIX_DOMAIN_STR      string = "unix domain type:" + TEST_STR

	TEST_HTTP_PORT      int64  = 10704
	TEST_HTTP_FRP_PORT  int64  = 10804
	TEST_HTTP_WEB01_STR string = "http web01:" + TEST_STR
)

func init() {
	go StartTcpEchoServer()
	go StartUdpEchoServer()
	go StartUnixDomainServer()
	go StartHttpServer()
	time.Sleep(500 * time.Millisecond)
}

func TestTcpServer(t *testing.T) {
	assert := assert.New(t)
	// Normal
	addr := fmt.Sprintf("127.0.0.1:%d", TEST_TCP_FRP_PORT)
	res, err := sendTcpMsg(addr, TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(TEST_TCP_ECHO_STR, res)

	// Encrytion and compression
	addr = fmt.Sprintf("127.0.0.1:%d", TEST_TCP_EC_FRP_PORT)
	res, err = sendTcpMsg(addr, TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(TEST_TCP_ECHO_STR, res)
}

func TestUdpEchoServer(t *testing.T) {
	assert := assert.New(t)
	// Normal
	addr := fmt.Sprintf("127.0.0.1:%d", TEST_UDP_FRP_PORT)
	res, err := sendUdpMsg(addr, TEST_UDP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(TEST_UDP_ECHO_STR, res)

func TestUnixDomainServer(t *testing.T) {
	assert := assert.New(t)
	// Normal
	addr := fmt.Sprintf("127.0.0.1:%d", TEST_UNIX_DOMAIN_FRP_PORT)
	res, err := sendTcpMsg(addr, TEST_UNIX_DOMAIN_STR)
	assert.NoError(err)
	assert.Equal(TEST_UNIX_DOMAIN_STR, res)
}

func TestHttpServer(t *testing.T) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d", HTTP_PORT), nil)
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("do http request error: %v", err)
	}
	if res.StatusCode == 200 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("read from http server error: %v", err)
		}
		bodystr := string(body)
		if bodystr != HTTP_RES_STR {
			t.Fatalf("content from http server error [%s], correct string is [%s]", bodystr, HTTP_RES_STR)
		}
	} else {
		t.Fatalf("http code from http server error [%d]", res.StatusCode)
	}
}

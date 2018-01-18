package tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/server"
)

var (
	SERVER_ADDR = "127.0.0.1"
	ADMIN_ADDR  = "127.0.0.1:10600"
	ADMIN_USER  = "abc"
	ADMIN_PWD   = "abc"

	TEST_STR                    = "frp is a fast reverse proxy to help you expose a local server behind a NAT or firewall to the internet."
	TEST_TCP_PORT        int    = 10701
	TEST_TCP_FRP_PORT    int    = 10801
	TEST_TCP_EC_FRP_PORT int    = 10901
	TEST_TCP_ECHO_STR    string = "tcp type:" + TEST_STR

	TEST_UDP_PORT        int    = 10702
	TEST_UDP_FRP_PORT    int    = 10802
	TEST_UDP_EC_FRP_PORT int    = 10902
	TEST_UDP_ECHO_STR    string = "udp type:" + TEST_STR

	TEST_UNIX_DOMAIN_ADDR     string = "/tmp/frp_echo_server.sock"
	TEST_UNIX_DOMAIN_FRP_PORT int    = 10803
	TEST_UNIX_DOMAIN_STR      string = "unix domain type:" + TEST_STR

	TEST_HTTP_PORT       int    = 10704
	TEST_HTTP_FRP_PORT   int    = 10804
	TEST_HTTP_NORMAL_STR string = "http normal string: " + TEST_STR
	TEST_HTTP_FOO_STR    string = "http foo string: " + TEST_STR
	TEST_HTTP_BAR_STR    string = "http bar string: " + TEST_STR

	TEST_STCP_FRP_PORT    int    = 10805
	TEST_STCP_EC_FRP_PORT int    = 10905
	TEST_STCP_ECHO_STR    string = "stcp type:" + TEST_STR

	ProxyTcpPortNotAllowed  string = "tcp_port_not_allowed"
	ProxyTcpPortUnavailable string = "tcp_port_unavailable"
	ProxyTcpPortNormal      string = "tcp_port_normal"
	ProxyTcpRandomPort      string = "tcp_random_port"
	ProxyUdpPortNotAllowed  string = "udp_port_not_allowed"
	ProxyUdpPortNormal      string = "udp_port_normal"
	ProxyUdpRandomPort      string = "udp_random_port"
)

func init() {
	go StartTcpEchoServer()
	go StartUdpEchoServer()
	go StartUnixDomainServer()
	go StartHttpServer()
	time.Sleep(500 * time.Millisecond)
}

func TestTcp(t *testing.T) {
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

func TestUdp(t *testing.T) {
	assert := assert.New(t)
	// Normal
	addr := fmt.Sprintf("127.0.0.1:%d", TEST_UDP_FRP_PORT)
	res, err := sendUdpMsg(addr, TEST_UDP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(TEST_UDP_ECHO_STR, res)

	// Encrytion and compression
	addr = fmt.Sprintf("127.0.0.1:%d", TEST_UDP_EC_FRP_PORT)
	res, err = sendUdpMsg(addr, TEST_UDP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(TEST_UDP_ECHO_STR, res)
}

func TestUnixDomain(t *testing.T) {
	assert := assert.New(t)
	// Normal
	addr := fmt.Sprintf("127.0.0.1:%d", TEST_UNIX_DOMAIN_FRP_PORT)
	res, err := sendTcpMsg(addr, TEST_UNIX_DOMAIN_STR)
	if assert.NoError(err) {
		assert.Equal(TEST_UNIX_DOMAIN_STR, res)
	}
}

func TestStcp(t *testing.T) {
	assert := assert.New(t)
	// Normal
	addr := fmt.Sprintf("127.0.0.1:%d", TEST_STCP_FRP_PORT)
	res, err := sendTcpMsg(addr, TEST_STCP_ECHO_STR)
	if assert.NoError(err) {
		assert.Equal(TEST_STCP_ECHO_STR, res)
	}

	// Encrytion and compression
	addr = fmt.Sprintf("127.0.0.1:%d", TEST_STCP_EC_FRP_PORT)
	res, err = sendTcpMsg(addr, TEST_STCP_ECHO_STR)
	if assert.NoError(err) {
		assert.Equal(TEST_STCP_ECHO_STR, res)
	}
}

func TestHttp(t *testing.T) {
	assert := assert.New(t)
	// web01
	code, body, err := sendHttpMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", TEST_HTTP_FRP_PORT), "", nil)
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(TEST_HTTP_NORMAL_STR, body)
	}

	// web02
	code, body, err = sendHttpMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", TEST_HTTP_FRP_PORT), "test2.frp.com", nil)
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(TEST_HTTP_NORMAL_STR, body)
	}

	// error host header
	code, body, err = sendHttpMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", TEST_HTTP_FRP_PORT), "errorhost.frp.com", nil)
	if assert.NoError(err) {
		assert.Equal(404, code)
	}

	// web03
	code, body, err = sendHttpMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", TEST_HTTP_FRP_PORT), "test3.frp.com", nil)
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(TEST_HTTP_NORMAL_STR, body)
	}

	code, body, err = sendHttpMsg("GET", fmt.Sprintf("http://127.0.0.1:%d/foo", TEST_HTTP_FRP_PORT), "test3.frp.com", nil)
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(TEST_HTTP_FOO_STR, body)
	}

	// web04
	code, body, err = sendHttpMsg("GET", fmt.Sprintf("http://127.0.0.1:%d/bar", TEST_HTTP_FRP_PORT), "test3.frp.com", nil)
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(TEST_HTTP_BAR_STR, body)
	}

	// web05
	code, body, err = sendHttpMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", TEST_HTTP_FRP_PORT), "test5.frp.com", nil)
	if assert.NoError(err) {
		assert.Equal(401, code)
	}

	header := make(map[string]string)
	header["Authorization"] = basicAuth("test", "test")
	code, body, err = sendHttpMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", TEST_HTTP_FRP_PORT), "test5.frp.com", header)
	if assert.NoError(err) {
		assert.Equal(401, code)
	}
}

func TestPrivilegeAllowPorts(t *testing.T) {
	assert := assert.New(t)
	// Port not allowed
	status, err := getProxyStatus(ProxyTcpPortNotAllowed)
	if assert.NoError(err) {
		assert.Equal(client.ProxyStatusStartErr, status.Status)
		assert.True(strings.Contains(status.Err, server.ErrPortNotAllowed.Error()))
	}

	status, err = getProxyStatus(ProxyUdpPortNotAllowed)
	if assert.NoError(err) {
		assert.Equal(client.ProxyStatusStartErr, status.Status)
		assert.True(strings.Contains(status.Err, server.ErrPortNotAllowed.Error()))
	}

	status, err = getProxyStatus(ProxyTcpPortUnavailable)
	if assert.NoError(err) {
		assert.Equal(client.ProxyStatusStartErr, status.Status)
		assert.True(strings.Contains(status.Err, server.ErrPortUnAvailable.Error()))
	}

	// Port normal
	status, err = getProxyStatus(ProxyTcpPortNormal)
	if assert.NoError(err) {
		assert.Equal(client.ProxyStatusRunning, status.Status)
	}

	status, err = getProxyStatus(ProxyUdpPortNormal)
	if assert.NoError(err) {
		assert.Equal(client.ProxyStatusRunning, status.Status)
	}
}

func TestRandomPort(t *testing.T) {
	assert := assert.New(t)
	// tcp
	status, err := getProxyStatus(ProxyTcpRandomPort)
	if assert.NoError(err) {
		addr := status.RemoteAddr
		res, err := sendTcpMsg(addr, TEST_TCP_ECHO_STR)
		assert.NoError(err)
		assert.Equal(TEST_TCP_ECHO_STR, res)
	}

	// udp
	status, err = getProxyStatus(ProxyUdpRandomPort)
	if assert.NoError(err) {
		addr := status.RemoteAddr
		res, err := sendUdpMsg(addr, TEST_UDP_ECHO_STR)
		assert.NoError(err)
		assert.Equal(TEST_UDP_ECHO_STR, res)
	}
}

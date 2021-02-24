package ci

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/server/ports"
	"github.com/fatedier/frp/tests/consts"
	"github.com/fatedier/frp/tests/mock"
	"github.com/fatedier/frp/tests/util"

	gnet "github.com/fatedier/golib/net"
)

func TestMain(m *testing.M) {
	var err error
	tcpEcho1 := mock.NewEchoServer(consts.TEST_TCP_PORT, 1, "")
	tcpEcho2 := mock.NewEchoServer(consts.TEST_TCP2_PORT, 2, "")

	if err = tcpEcho1.Start(); err != nil {
		panic(err)
	}
	if err = tcpEcho2.Start(); err != nil {
		panic(err)
	}

	go mock.StartUDPEchoServer(consts.TEST_UDP_PORT)
	go mock.StartUnixDomainServer(consts.TEST_UNIX_DOMAIN_ADDR)
	go mock.StartHTTPServer(consts.TEST_HTTP_PORT)

	p1 := util.NewProcess(consts.FRPS_BIN_PATH, []string{"-c", "./auto_test_frps.ini"})
	if err = p1.Start(); err != nil {
		panic(err)
	}

	time.Sleep(500 * time.Millisecond)
	p2 := util.NewProcess(consts.FRPC_BIN_PATH, []string{"-c", "./auto_test_frpc.ini"})
	if err = p2.Start(); err != nil {
		panic(err)
	}

	p3 := util.NewProcess(consts.FRPC_BIN_PATH, []string{"-c", "./auto_test_frpc_visitor.ini"})
	if err = p3.Start(); err != nil {
		panic(err)
	}
	time.Sleep(500 * time.Millisecond)

	exitCode := m.Run()
	p1.Stop()
	p2.Stop()
	p3.Stop()
	os.Exit(exitCode)
}

func TestHTTP(t *testing.T) {
	assert := assert.New(t)
	// web01
	code, body, _, err := util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT), "", nil, "")
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(consts.TEST_HTTP_NORMAL_STR, body)
	}

	// web02
	code, body, _, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT), "test2.frp.com", nil, "")
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(consts.TEST_HTTP_NORMAL_STR, body)
	}

	// error host header
	code, body, _, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT), "errorhost.frp.com", nil, "")
	if assert.NoError(err) {
		assert.Equal(404, code)
	}

	// web03
	code, body, _, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT), "test3.frp.com", nil, "")
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(consts.TEST_HTTP_NORMAL_STR, body)
	}

	code, body, _, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d/foo", consts.TEST_HTTP_FRP_PORT), "test3.frp.com", nil, "")
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(consts.TEST_HTTP_FOO_STR, body)
	}

	// web04
	code, body, _, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d/bar", consts.TEST_HTTP_FRP_PORT), "test3.frp.com", nil, "")
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(consts.TEST_HTTP_BAR_STR, body)
	}

	// web05
	code, body, _, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT), "test5.frp.com", nil, "")
	if assert.NoError(err) {
		assert.Equal(401, code)
	}

	headers := make(map[string]string)
	headers["Authorization"] = util.BasicAuth("test", "test")
	code, body, _, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT), "test5.frp.com", headers, "")
	if assert.NoError(err) {
		assert.Equal(401, code)
	}

	// web06
	var header http.Header
	code, body, header, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT), "test6.frp.com", nil, "")
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(consts.TEST_HTTP_NORMAL_STR, body)
		assert.Equal("true", header.Get("X-Header-Set"))
	}

	// wildcard_http
	// test.frp1.com match *.frp1.com
	code, body, _, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT), "test.frp1.com", nil, "")
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(consts.TEST_HTTP_NORMAL_STR, body)
	}

	// new.test.frp1.com also match *.frp1.com
	code, body, _, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT), "new.test.frp1.com", nil, "")
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(consts.TEST_HTTP_NORMAL_STR, body)
	}

	// subhost01
	code, body, _, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT), "test01.sub.com", nil, "")
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal("test01.sub.com", body)
	}

	// subhost02
	code, body, _, err = util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT), "test02.sub.com", nil, "")
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal("test02.sub.com", body)
	}
}

func TestTCPMux(t *testing.T) {
	assert := assert.New(t)

	conn, err := gnet.DialTcpByProxy(fmt.Sprintf("http://%s:%d", "127.0.0.1", consts.TEST_TCP_MUX_FRP_PORT), "tunnel1")
	if assert.NoError(err) {
		res, err := util.SendTCPMsgByConn(conn, consts.TEST_TCP_ECHO_STR)
		assert.NoError(err)
		assert.Equal(consts.TEST_TCP_ECHO_STR, res)
	}
}

func TestWebSocket(t *testing.T) {
	assert := assert.New(t)

	u := url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:%d", "127.0.0.1", consts.TEST_HTTP_FRP_PORT), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	assert.NoError(err)
	defer c.Close()

	err = c.WriteMessage(websocket.TextMessage, []byte(consts.TEST_HTTP_NORMAL_STR))
	assert.NoError(err)

	_, msg, err := c.ReadMessage()
	assert.NoError(err)
	assert.Equal(consts.TEST_HTTP_NORMAL_STR, string(msg))
}

func TestAllowPorts(t *testing.T) {
	assert := assert.New(t)
	// Port not allowed
	status, err := util.GetProxyStatus(consts.ADMIN_ADDR, consts.ADMIN_USER, consts.ADMIN_PWD, consts.ProxyTCPPortNotAllowed)
	if assert.NoError(err) {
		assert.Equal(proxy.ProxyPhaseStartErr, status.Status)
		assert.True(strings.Contains(status.Err, ports.ErrPortNotAllowed.Error()))
	}

	status, err = util.GetProxyStatus(consts.ADMIN_ADDR, consts.ADMIN_USER, consts.ADMIN_PWD, consts.ProxyUDPPortNotAllowed)
	if assert.NoError(err) {
		assert.Equal(proxy.ProxyPhaseStartErr, status.Status)
		assert.True(strings.Contains(status.Err, ports.ErrPortNotAllowed.Error()))
	}

	status, err = util.GetProxyStatus(consts.ADMIN_ADDR, consts.ADMIN_USER, consts.ADMIN_PWD, consts.ProxyTCPPortUnavailable)
	if assert.NoError(err) {
		assert.Equal(proxy.ProxyPhaseStartErr, status.Status)
		assert.True(strings.Contains(status.Err, ports.ErrPortUnAvailable.Error()))
	}

	// Port normal
	status, err = util.GetProxyStatus(consts.ADMIN_ADDR, consts.ADMIN_USER, consts.ADMIN_PWD, consts.ProxyTCPPortNormal)
	if assert.NoError(err) {
		assert.Equal(proxy.ProxyPhaseRunning, status.Status)
	}

	status, err = util.GetProxyStatus(consts.ADMIN_ADDR, consts.ADMIN_USER, consts.ADMIN_PWD, consts.ProxyUDPPortNormal)
	if assert.NoError(err) {
		assert.Equal(proxy.ProxyPhaseRunning, status.Status)
	}
}

func TestRandomPort(t *testing.T) {
	assert := assert.New(t)
	// tcp
	status, err := util.GetProxyStatus(consts.ADMIN_ADDR, consts.ADMIN_USER, consts.ADMIN_PWD, consts.ProxyTCPRandomPort)
	if assert.NoError(err) {
		addr := status.RemoteAddr
		res, err := util.SendTCPMsg(addr, consts.TEST_TCP_ECHO_STR)
		assert.NoError(err)
		assert.Equal(consts.TEST_TCP_ECHO_STR, res)
	}

	// udp
	status, err = util.GetProxyStatus(consts.ADMIN_ADDR, consts.ADMIN_USER, consts.ADMIN_PWD, consts.ProxyUDPRandomPort)
	if assert.NoError(err) {
		addr := status.RemoteAddr
		res, err := util.SendUDPMsg(addr, consts.TEST_UDP_ECHO_STR)
		assert.NoError(err)
		assert.Equal(consts.TEST_UDP_ECHO_STR, res)
	}
}

func TestPluginHTTPProxy(t *testing.T) {
	assert := assert.New(t)
	status, err := util.GetProxyStatus(consts.ADMIN_ADDR, consts.ADMIN_USER, consts.ADMIN_PWD, consts.ProxyHTTPProxy)
	if assert.NoError(err) {
		assert.Equal(proxy.ProxyPhaseRunning, status.Status)

		// http proxy
		addr := status.RemoteAddr
		code, body, _, err := util.SendHTTPMsg("GET", fmt.Sprintf("http://127.0.0.1:%d", consts.TEST_HTTP_FRP_PORT),
			"", nil, "http://"+addr)
		if assert.NoError(err) {
			assert.Equal(200, code)
			assert.Equal(consts.TEST_HTTP_NORMAL_STR, body)
		}

		// connect method
		conn, err := gnet.DialTcpByProxy("http://"+addr, fmt.Sprintf("127.0.0.1:%d", consts.TEST_TCP_FRP_PORT))
		if assert.NoError(err) {
			res, err := util.SendTCPMsgByConn(conn, consts.TEST_TCP_ECHO_STR)
			assert.NoError(err)
			assert.Equal(consts.TEST_TCP_ECHO_STR, res)
		}
	}
}

func TestRangePortsMapping(t *testing.T) {
	assert := assert.New(t)

	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("%s_%d", consts.ProxyRangeTCPPrefix, i)
		status, err := util.GetProxyStatus(consts.ADMIN_ADDR, consts.ADMIN_USER, consts.ADMIN_PWD, name)
		if assert.NoError(err) {
			assert.Equal(proxy.ProxyPhaseRunning, status.Status)
		}
	}
}

func TestGroup(t *testing.T) {
	assert := assert.New(t)

	var (
		p1 int
		p2 int
	)
	addr := fmt.Sprintf("127.0.0.1:%d", consts.TEST_TCP2_FRP_PORT)

	for i := 0; i < 6; i++ {
		res, err := util.SendTCPMsg(addr, consts.TEST_TCP_ECHO_STR)
		assert.NoError(err)
		switch res {
		case consts.TEST_TCP_ECHO_STR:
			p1++
		case consts.TEST_TCP_ECHO_STR + consts.TEST_TCP_ECHO_STR:
			p2++
		}
	}
	assert.True(p1 > 0 && p2 > 0, "group proxies load balancing")
}

package ci

import (
	"testing"
	"time"

	"github.com/fatedier/frp/tests/consts"
	"github.com/fatedier/frp/tests/util"

	"github.com/stretchr/testify/assert"
)

func TestCmdTCP(t *testing.T) {
	assert := assert.New(t)

	var err error
	s := util.NewProcess(consts.FRPS_BIN_PATH, []string{"-t", "123", "-p", "20000"})
	err = s.Start()
	if assert.NoError(err) {
		defer s.Stop()
	}
	time.Sleep(500 * time.Millisecond)

	c := util.NewProcess(consts.FRPC_BIN_PATH, []string{"tcp", "-s", "127.0.0.1:20000", "-t", "123", "-u", "test",
		"-l", "10701", "-r", "20801", "-n", "tcp_test"})
	err = c.Start()
	if assert.NoError(err) {
		defer c.Stop()
	}
	time.Sleep(500 * time.Millisecond)

	res, err := util.SendTCPMsg("127.0.0.1:20801", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)
}

func TestCmdUDP(t *testing.T) {
	assert := assert.New(t)

	var err error
	s := util.NewProcess(consts.FRPS_BIN_PATH, []string{"-t", "123", "-p", "20000"})
	err = s.Start()
	if assert.NoError(err) {
		defer s.Stop()
	}
	time.Sleep(500 * time.Millisecond)

	c := util.NewProcess(consts.FRPC_BIN_PATH, []string{"udp", "-s", "127.0.0.1:20000", "-t", "123", "-u", "test",
		"-l", "10702", "-r", "20802", "-n", "udp_test"})
	err = c.Start()
	if assert.NoError(err) {
		defer c.Stop()
	}
	time.Sleep(500 * time.Millisecond)

	res, err := util.SendUDPMsg("127.0.0.1:20802", consts.TEST_UDP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_UDP_ECHO_STR, res)
}

func TestCmdHTTP(t *testing.T) {
	assert := assert.New(t)

	var err error
	s := util.NewProcess(consts.FRPS_BIN_PATH, []string{"-t", "123", "-p", "20000", "--vhost_http_port", "20001"})
	err = s.Start()
	if assert.NoError(err) {
		defer s.Stop()
	}
	time.Sleep(500 * time.Millisecond)

	c := util.NewProcess(consts.FRPC_BIN_PATH, []string{"http", "-s", "127.0.0.1:20000", "-t", "123", "-u", "test",
		"-n", "udp_test", "-l", "10704", "--custom_domain", "127.0.0.1"})
	err = c.Start()
	if assert.NoError(err) {
		defer c.Stop()
	}
	time.Sleep(500 * time.Millisecond)

	code, body, _, err := util.SendHTTPMsg("GET", "http://127.0.0.1:20001", "", nil, "")
	if assert.NoError(err) {
		assert.Equal(200, code)
		assert.Equal(consts.TEST_HTTP_NORMAL_STR, body)
	}
}

package ci

import (
	"os"
	"testing"
	"time"

	"github.com/fatedier/frp/tests/config"
	"github.com/fatedier/frp/tests/consts"
	"github.com/fatedier/frp/tests/util"

	"github.com/stretchr/testify/assert"
)

const FRPS_RECONNECT_CONF = `
[common]
bind_addr = 0.0.0.0
bind_port = 20000
log_file = console
log_level = debug
token = 123456
`

const FRPC_RECONNECT_CONF = `
[common]
server_addr = 127.0.0.1
server_port = 20000
log_file = console
log_level = debug
token = 123456
admin_port = 21000
admin_user = abc
admin_pwd = abc

[tcp]
type = tcp
local_port = 10701
remote_port = 20801
`

func TestReconnect(t *testing.T) {
	assert := assert.New(t)
	frpsCfgPath, err := config.GenerateConfigFile(consts.FRPS_NORMAL_CONFIG, FRPS_RECONNECT_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpsCfgPath)
	}

	frpcCfgPath, err := config.GenerateConfigFile(consts.FRPC_NORMAL_CONFIG, FRPC_RECONNECT_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpcCfgPath)
	}

	frpsProcess := util.NewProcess(consts.FRPS_BIN_PATH, []string{"-c", frpsCfgPath})
	err = frpsProcess.Start()
	if assert.NoError(err) {
		defer frpsProcess.Stop()
	}

	time.Sleep(500 * time.Millisecond)

	frpcProcess := util.NewProcess(consts.FRPC_BIN_PATH, []string{"-c", frpcCfgPath})
	err = frpcProcess.Start()
	if assert.NoError(err) {
		defer frpcProcess.Stop()
	}
	time.Sleep(500 * time.Millisecond)

	// test tcp
	res, err := util.SendTCPMsg("127.0.0.1:20801", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)

	// stop frpc
	frpcProcess.Stop()
	time.Sleep(200 * time.Millisecond)

	// test tcp, expect failed
	_, err = util.SendTCPMsg("127.0.0.1:20801", consts.TEST_TCP_ECHO_STR)
	assert.Error(err)

	// restart frpc
	newFrpcProcess := util.NewProcess(consts.FRPC_BIN_PATH, []string{"-c", frpcCfgPath})
	err = newFrpcProcess.Start()
	if assert.NoError(err) {
		defer newFrpcProcess.Stop()
	}
	time.Sleep(500 * time.Millisecond)

	// test tcp
	res, err = util.SendTCPMsg("127.0.0.1:20801", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)

	// stop frps
	frpsProcess.Stop()
	time.Sleep(200 * time.Millisecond)

	// test tcp, expect failed
	_, err = util.SendTCPMsg("127.0.0.1:20801", consts.TEST_TCP_ECHO_STR)
	assert.Error(err)

	// restart frps
	newFrpsProcess := util.NewProcess(consts.FRPS_BIN_PATH, []string{"-c", frpsCfgPath})
	err = newFrpsProcess.Start()
	if assert.NoError(err) {
		defer newFrpsProcess.Stop()
	}

	time.Sleep(2 * time.Second)

	// test tcp
	res, err = util.SendTCPMsg("127.0.0.1:20801", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)
}

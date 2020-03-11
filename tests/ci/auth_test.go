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

const FRPS_TOKEN_TCP_CONF = `
[common]
bind_addr = 0.0.0.0
bind_port = 20000
log_file = console
log_level = debug
authentication_method = token
token = 123456
`

const FRPC_TOKEN_TCP_CONF = `
[common]
server_addr = 127.0.0.1
server_port = 20000
log_file = console
log_level = debug
authentication_method = token
token = 123456
protocol = tcp

[tcp]
type = tcp
local_port = 10701
remote_port = 20801
`

func TestTcpWithTokenAuthentication(t *testing.T) {
	assert := assert.New(t)
	frpsCfgPath, err := config.GenerateConfigFile(consts.FRPS_NORMAL_CONFIG, FRPS_TOKEN_TCP_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpsCfgPath)
	}

	frpcCfgPath, err := config.GenerateConfigFile(consts.FRPC_NORMAL_CONFIG, FRPC_TOKEN_TCP_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpcCfgPath)
	}

	frpsProcess := util.NewProcess(consts.FRPS_BIN_PATH, []string{"-c", frpsCfgPath})
	err = frpsProcess.Start()
	if assert.NoError(err) {
		defer frpsProcess.Stop()
	}

	time.Sleep(200 * time.Millisecond)

	frpcProcess := util.NewProcess(consts.FRPC_BIN_PATH, []string{"-c", frpcCfgPath})
	err = frpcProcess.Start()
	if assert.NoError(err) {
		defer frpcProcess.Stop()
	}
	time.Sleep(500 * time.Millisecond)

	// test tcp
	res, err := util.SendTcpMsg("127.0.0.1:20801", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)
}

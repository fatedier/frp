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

const FRPS_TEMPLATE_CONF = `
[common]
bind_addr = 0.0.0.0
bind_port = {{ .Envs.SERVER_PORT }}
log_file = console
# debug, info, warn, error
log_level = debug
token = 123456
`

const FRPC_TEMPLATE_CONF = `
[common]
server_addr = 127.0.0.1
server_port = 20000
log_file = console
# debug, info, warn, error
log_level = debug
token = {{ .Envs.FRP_TOKEN }}

[tcp]
type = tcp
local_port = 10701
remote_port = {{ .Envs.TCP_REMOTE_PORT }}
`

func TestConfTemplate(t *testing.T) {
	assert := assert.New(t)
	frpsCfgPath, err := config.GenerateConfigFile(consts.FRPS_NORMAL_CONFIG, FRPS_TEMPLATE_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpsCfgPath)
	}

	frpcCfgPath, err := config.GenerateConfigFile(consts.FRPC_NORMAL_CONFIG, FRPC_TEMPLATE_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpcCfgPath)
	}

	frpsProcess := util.NewProcess("env", []string{"SERVER_PORT=20000", consts.FRPS_BIN_PATH, "-c", frpsCfgPath})
	err = frpsProcess.Start()
	if assert.NoError(err) {
		defer frpsProcess.Stop()
	}

	time.Sleep(200 * time.Millisecond)

	frpcProcess := util.NewProcess("env", []string{"FRP_TOKEN=123456", "TCP_REMOTE_PORT=20801", consts.FRPC_BIN_PATH, "-c", frpcCfgPath})
	err = frpcProcess.Start()
	if assert.NoError(err) {
		defer frpcProcess.Stop()
	}

	time.Sleep(500 * time.Millisecond)

	// test tcp1
	res, err := util.SendTcpMsg("127.0.0.1:20801", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)
}

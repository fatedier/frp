package ci

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fatedier/frp/tests/config"
	"github.com/fatedier/frp/tests/consts"
	"github.com/fatedier/frp/tests/util"
)

const FRPS_CONF = `
[common]
server_addr = 127.0.0.1
server_port = 10700
log_file = console
# debug, info, warn, error
log_level = debug
token = 123456
admin_port = 10600
admin_user = abc
admin_pwd = abc
`

const FRPC_CONF_1 = `
[common]
server_addr = 127.0.0.1
server_port = 20000
log_file = console
# debug, info, warn, error
log_level = debug
token = 123456
admin_port = 21000
admin_user = abc
admin_pwd = abc

[tcp]
type = tcp
local_port = 10701
remote_port = 20801

# change remote port
[tcp2]
type = tcp
local_port = 10701
remote_port = 20802

# delete
[tcp3]
type = tcp
local_port = 10701
remote_port = 20803
`

const FRPC_CONF_2 = `
[common]
server_addr = 127.0.0.1
server_port = 20000
log_file = console
# debug, info, warn, error
log_level = debug
token = 123456
admin_port = 21000
admin_user = abc
admin_pwd = abc

[tcp]
type = tcp
local_port = 10701
remote_port = 20801

[tcp2]
type = tcp
local_port = 10701
remote_port = 20902
`

func TestReload(t *testing.T) {
	assert := assert.New(t)
	frpsCfgPath, err := config.GenerateConfigFile("./auto_test_frps.ini", FRPS_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpsCfgPath)
	}

	frpcCfgPath, err := config.GenerateConfigFile("./auto_test_frpc.ini", FRPC_CONF_1)
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

	// TODO
}

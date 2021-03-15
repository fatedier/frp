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

const FRPS_RELOAD_CONF = `
[common]
bind_addr = 0.0.0.0
bind_port = 20000
log_file = console
# debug, info, warn, error
log_level = debug
token = 123456
`

const FRPC_RELOAD_CONF_1 = `
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

const FRPC_RELOAD_CONF_2 = `
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
	frpsCfgPath, err := config.GenerateConfigFile(consts.FRPS_NORMAL_CONFIG, FRPS_RELOAD_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpsCfgPath)
	}

	frpcCfgPath, err := config.GenerateConfigFile(consts.FRPC_NORMAL_CONFIG, FRPC_RELOAD_CONF_1)
	if assert.NoError(err) {
		rmFile1 := frpcCfgPath
		defer os.Remove(rmFile1)
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

	// test tcp1
	res, err := util.SendTCPMsg("127.0.0.1:20801", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)

	// test tcp2
	res, err = util.SendTCPMsg("127.0.0.1:20802", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)

	// test tcp3
	res, err = util.SendTCPMsg("127.0.0.1:20803", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)

	// reload frpc config
	frpcCfgPath, err = config.GenerateConfigFile(consts.FRPC_NORMAL_CONFIG, FRPC_RELOAD_CONF_2)
	if assert.NoError(err) {
		rmFile2 := frpcCfgPath
		defer os.Remove(rmFile2)
	}
	err = util.ReloadConf("127.0.0.1:21000", "abc", "abc")
	assert.NoError(err)

	time.Sleep(time.Second)

	// test tcp1
	res, err = util.SendTCPMsg("127.0.0.1:20801", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)

	// test origin tcp2, expect failed
	res, err = util.SendTCPMsg("127.0.0.1:20802", consts.TEST_TCP_ECHO_STR)
	assert.Error(err)

	// test new origin tcp2 with different port
	res, err = util.SendTCPMsg("127.0.0.1:20902", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)

	// test tcp3, expect failed
	res, err = util.SendTCPMsg("127.0.0.1:20803", consts.TEST_TCP_ECHO_STR)
	assert.Error(err)
}

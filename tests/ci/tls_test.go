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

const FRPS_TLS_TCP_CONF = `
[common]
bind_addr = 0.0.0.0
bind_port = 20000
log_file = console
log_level = debug
token = 123456
`

const FRPC_TLS_TCP_CONF = `
[common]
server_addr = 127.0.0.1
server_port = 20000
log_file = console
log_level = debug
token = 123456
protocol = tcp
tls_enable = true

[tcp]
type = tcp
local_port = 10701
remote_port = 20801
`

func TestTlsOverTCP(t *testing.T) {
	assert := assert.New(t)
	frpsCfgPath, err := config.GenerateConfigFile(consts.FRPS_NORMAL_CONFIG, FRPS_TLS_TCP_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpsCfgPath)
	}

	frpcCfgPath, err := config.GenerateConfigFile(consts.FRPC_NORMAL_CONFIG, FRPC_TLS_TCP_CONF)
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

const FRPS_TLS_KCP_CONF = `
[common]
bind_addr = 0.0.0.0
bind_port = 20000
kcp_bind_port = 20000
log_file = console
log_level = debug
token = 123456
`

const FRPC_TLS_KCP_CONF = `
[common]
server_addr = 127.0.0.1
server_port = 20000
log_file = console
log_level = debug
token = 123456
protocol = kcp
tls_enable = true

[tcp]
type = tcp
local_port = 10701
remote_port = 20801
`

func TestTLSOverKCP(t *testing.T) {
	assert := assert.New(t)
	frpsCfgPath, err := config.GenerateConfigFile(consts.FRPS_NORMAL_CONFIG, FRPS_TLS_KCP_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpsCfgPath)
	}

	frpcCfgPath, err := config.GenerateConfigFile(consts.FRPC_NORMAL_CONFIG, FRPC_TLS_KCP_CONF)
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

const FRPS_TLS_WS_CONF = `
[common]
bind_addr = 0.0.0.0
bind_port = 20000
log_file = console
log_level = debug
token = 123456
`

const FRPC_TLS_WS_CONF = `
[common]
server_addr = 127.0.0.1
server_port = 20000
log_file = console
log_level = debug
token = 123456
protocol = websocket
tls_enable = true

[tcp]
type = tcp
local_port = 10701
remote_port = 20801
`

func TestTLSOverWebsocket(t *testing.T) {
	assert := assert.New(t)
	frpsCfgPath, err := config.GenerateConfigFile(consts.FRPS_NORMAL_CONFIG, FRPS_TLS_WS_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpsCfgPath)
	}

	frpcCfgPath, err := config.GenerateConfigFile(consts.FRPC_NORMAL_CONFIG, FRPC_TLS_WS_CONF)
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

const FRPS_TLS_ONLY_TCP_CONF = `
[common]
bind_addr = 0.0.0.0
bind_port = 20000
log_file = console
log_level = debug
token = 123456
tls_only = true
`

const FRPC_TLS_ONLY_TCP_CONF = `
[common]
server_addr = 127.0.0.1
server_port = 20000
log_file = console
log_level = debug
token = 123456
protocol = tcp
tls_enable = true

[tcp]
type = tcp
local_port = 10701
remote_port = 20801
`

const FRPC_TLS_ONLY_NO_TLS_TCP_CONF = `
[common]
server_addr = 127.0.0.1
server_port = 20000
log_file = console
log_level = debug
token = 123456
protocol = tcp
tls_enable = false

[tcp]
type = tcp
local_port = 10701
remote_port = 20802
`

func TestTlsOnlyOverTCP(t *testing.T) {
	assert := assert.New(t)
	frpsCfgPath, err := config.GenerateConfigFile(consts.FRPS_NORMAL_CONFIG, FRPS_TLS_ONLY_TCP_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpsCfgPath)
	}

	frpcWithTlsCfgPath, err := config.GenerateConfigFile(consts.FRPC_NORMAL_CONFIG, FRPC_TLS_ONLY_TCP_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpcWithTlsCfgPath)
	}

	frpsProcess := util.NewProcess(consts.FRPS_BIN_PATH, []string{"-c", frpsCfgPath})
	err = frpsProcess.Start()
	if assert.NoError(err) {
		defer frpsProcess.Stop()
	}

	time.Sleep(200 * time.Millisecond)

	frpcProcessWithTls := util.NewProcess(consts.FRPC_BIN_PATH, []string{"-c", frpcWithTlsCfgPath})
	err = frpcProcessWithTls.Start()
	if assert.NoError(err) {
		defer frpcProcessWithTls.Stop()
	}
	time.Sleep(500 * time.Millisecond)

	// test tcp over tls
	res, err := util.SendTcpMsg("127.0.0.1:20801", consts.TEST_TCP_ECHO_STR)
	assert.NoError(err)
	assert.Equal(consts.TEST_TCP_ECHO_STR, res)
	frpcProcessWithTls.Stop()

	frpcWithoutTlsCfgPath, err := config.GenerateConfigFile(consts.FRPC_NORMAL_CONFIG, FRPC_TLS_ONLY_NO_TLS_TCP_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpcWithTlsCfgPath)
	}

	frpcProcessWithoutTls := util.NewProcess(consts.FRPC_BIN_PATH, []string{"-c", frpcWithoutTlsCfgPath})
	err = frpcProcessWithoutTls.Start()
	if assert.NoError(err) {
		defer frpcProcessWithoutTls.Stop()
	}
	time.Sleep(500 * time.Millisecond)

	// test tcp without tls
	_, err = util.SendTcpMsg("127.0.0.1:20802", consts.TEST_TCP_ECHO_STR)
	assert.Error(err)
}

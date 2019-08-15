package health

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fatedier/frp/tests/config"
	"github.com/fatedier/frp/tests/consts"
	"github.com/fatedier/frp/tests/mock"
	"github.com/fatedier/frp/tests/util"

	"github.com/stretchr/testify/assert"
)

const FRPS_CONF = `
[common]
bind_addr = 0.0.0.0
bind_port = 14000
vhost_http_port = 14000
log_file = console
log_level = debug
token = 123456
`

const FRPC_CONF = `
[common]
server_addr = 127.0.0.1
server_port = 14000
log_file = console
log_level = debug
token = 123456

[tcp1]
type = tcp
local_port = 15001
remote_port = 15000
group = test
group_key = 123
health_check_type = tcp
health_check_interval_s = 1

[tcp2]
type = tcp
local_port = 15002
remote_port = 15000
group = test
group_key = 123
health_check_type = tcp
health_check_interval_s = 1

[http1]
type = http
local_port = 15003
custom_domains = test1.com
health_check_type = http
health_check_interval_s = 1
health_check_url = /health

[http2]
type = http
local_port = 15004
custom_domains = test2.com
health_check_type = http
health_check_interval_s = 1
health_check_url = /health

[http3]
type = http
local_port = 15005
custom_domains = test.balancing.com
group = test-balancing
group_key = 123

[http4]
type = http
local_port = 15006
custom_domains = test.balancing.com
group = test-balancing
group_key = 123
`

func TestHealthCheck(t *testing.T) {
	assert := assert.New(t)

	// ****** start background services ******
	echoSvc1 := mock.NewEchoServer(15001, 1, "echo1")
	err := echoSvc1.Start()
	if assert.NoError(err) {
		defer echoSvc1.Stop()
	}

	echoSvc2 := mock.NewEchoServer(15002, 1, "echo2")
	err = echoSvc2.Start()
	if assert.NoError(err) {
		defer echoSvc2.Stop()
	}

	var healthMu sync.RWMutex
	svc1Health := true
	svc2Health := true
	httpSvc1 := mock.NewHttpServer(15003, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "health") {
			healthMu.RLock()
			defer healthMu.RUnlock()
			if svc1Health {
				w.WriteHeader(200)
			} else {
				w.WriteHeader(500)
			}
		} else {
			w.Write([]byte("http1"))
		}
	})
	err = httpSvc1.Start()
	if assert.NoError(err) {
		defer httpSvc1.Stop()
	}

	httpSvc2 := mock.NewHttpServer(15004, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "health") {
			healthMu.RLock()
			defer healthMu.RUnlock()
			if svc2Health {
				w.WriteHeader(200)
			} else {
				w.WriteHeader(500)
			}
		} else {
			w.Write([]byte("http2"))
		}
	})
	err = httpSvc2.Start()
	if assert.NoError(err) {
		defer httpSvc2.Stop()
	}

	httpSvc3 := mock.NewHttpServer(15005, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("http3"))
	})
	err = httpSvc3.Start()
	if assert.NoError(err) {
		defer httpSvc3.Stop()
	}

	httpSvc4 := mock.NewHttpServer(15006, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("http4"))
	})
	err = httpSvc4.Start()
	if assert.NoError(err) {
		defer httpSvc4.Stop()
	}

	time.Sleep(200 * time.Millisecond)

	// ****** start frps and frpc ******
	frpsCfgPath, err := config.GenerateConfigFile(consts.FRPS_NORMAL_CONFIG, FRPS_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpsCfgPath)
	}

	frpcCfgPath, err := config.GenerateConfigFile(consts.FRPC_NORMAL_CONFIG, FRPC_CONF)
	if assert.NoError(err) {
		defer os.Remove(frpcCfgPath)
	}

	frpsProcess := util.NewProcess(consts.FRPS_SUB_BIN_PATH, []string{"-c", frpsCfgPath})
	err = frpsProcess.Start()
	if assert.NoError(err) {
		defer frpsProcess.Stop()
	}

	time.Sleep(100 * time.Millisecond)

	frpcProcess := util.NewProcess(consts.FRPC_SUB_BIN_PATH, []string{"-c", frpcCfgPath})
	err = frpcProcess.Start()
	if assert.NoError(err) {
		defer frpcProcess.Stop()
	}
	time.Sleep(1000 * time.Millisecond)

	// ****** healcheck type tcp ******
	// echo1 and echo2 is ok
	result := make([]string, 0)
	res, err := util.SendTcpMsg("127.0.0.1:15000", "echo")
	assert.NoError(err)
	result = append(result, res)

	res, err = util.SendTcpMsg("127.0.0.1:15000", "echo")
	assert.NoError(err)
	result = append(result, res)

	assert.Contains(result, "echo1")
	assert.Contains(result, "echo2")

	// close echo2 server, echo1 is work
	echoSvc2.Stop()
	time.Sleep(1200 * time.Millisecond)

	result = make([]string, 0)
	res, err = util.SendTcpMsg("127.0.0.1:15000", "echo")
	assert.NoError(err)
	result = append(result, res)

	res, err = util.SendTcpMsg("127.0.0.1:15000", "echo")
	assert.NoError(err)
	result = append(result, res)

	assert.NotContains(result, "echo2")

	// resume echo2 server, all services are ok
	echoSvc2 = mock.NewEchoServer(15002, 1, "echo2")
	err = echoSvc2.Start()
	if assert.NoError(err) {
		defer echoSvc2.Stop()
	}

	time.Sleep(1200 * time.Millisecond)

	result = make([]string, 0)
	res, err = util.SendTcpMsg("127.0.0.1:15000", "echo")
	assert.NoError(err)
	result = append(result, res)

	res, err = util.SendTcpMsg("127.0.0.1:15000", "echo")
	assert.NoError(err)
	result = append(result, res)

	assert.Contains(result, "echo1")
	assert.Contains(result, "echo2")

	// ****** healcheck type http ******
	// http1 and http2 is ok
	code, body, _, err := util.SendHttpMsg("GET", "http://127.0.0.1:14000/xxx", "test1.com", nil, "")
	assert.NoError(err)
	assert.Equal(200, code)
	assert.Equal("http1", body)

	code, body, _, err = util.SendHttpMsg("GET", "http://127.0.0.1:14000/xxx", "test2.com", nil, "")
	assert.NoError(err)
	assert.Equal(200, code)
	assert.Equal("http2", body)

	// http2 health check error
	healthMu.Lock()
	svc2Health = false
	healthMu.Unlock()
	time.Sleep(1200 * time.Millisecond)

	code, body, _, err = util.SendHttpMsg("GET", "http://127.0.0.1:14000/xxx", "test1.com", nil, "")
	assert.NoError(err)
	assert.Equal(200, code)
	assert.Equal("http1", body)

	code, _, _, err = util.SendHttpMsg("GET", "http://127.0.0.1:14000/xxx", "test2.com", nil, "")
	assert.NoError(err)
	assert.Equal(404, code)

	// resume http2 service, http1 and http2 are ok
	healthMu.Lock()
	svc2Health = true
	healthMu.Unlock()
	time.Sleep(1200 * time.Millisecond)

	code, body, _, err = util.SendHttpMsg("GET", "http://127.0.0.1:14000/xxx", "test1.com", nil, "")
	assert.NoError(err)
	assert.Equal(200, code)
	assert.Equal("http1", body)

	code, body, _, err = util.SendHttpMsg("GET", "http://127.0.0.1:14000/xxx", "test2.com", nil, "")
	assert.NoError(err)
	assert.Equal(200, code)
	assert.Equal("http2", body)

	// ****** load balancing type http ******
	result = make([]string, 0)

	code, body, _, err = util.SendHttpMsg("GET", "http://127.0.0.1:14000/xxx", "test.balancing.com", nil, "")
	assert.NoError(err)
	assert.Equal(200, code)
	result = append(result, body)

	code, body, _, err = util.SendHttpMsg("GET", "http://127.0.0.1:14000/xxx", "test.balancing.com", nil, "")
	assert.NoError(err)
	assert.Equal(200, code)
	result = append(result, body)

	assert.Contains(result, "http3")
	assert.Contains(result, "http4")
}

package features

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

var _ = ginkgo.Describe("[Feature: Monitor]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("Prometheus metrics", func() {
		dashboardPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		enablePrometheus = true
		webServer.addr = "0.0.0.0"
		webServer.port = %d
		`, dashboardPort)

		clientConf := consts.DefaultClientConfig
		remotePort := f.AllocPort()
		clientConf += fmt.Sprintf(`
		[[proxies]]
		name = "tcp"
		type = "tcp"
		localPort = {{ .%s }}
		remotePort = %d
		`, framework.TCPEchoServerPort, remotePort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).Port(remotePort).Ensure()
		time.Sleep(500 * time.Millisecond)

		framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
			r.HTTP().Port(dashboardPort).HTTPPath("/metrics")
		}).Ensure(func(resp *request.Response) bool {
			log.Tracef("prometheus metrics response: \n%s", resp.Content)
			if resp.Code != 200 {
				return false
			}
			if !strings.Contains(string(resp.Content), "traffic_in") {
				return false
			}
			return true
		})
	})
})

package e2e

import (
	"fmt"
	"time"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"

	. "github.com/onsi/ginkgo"
)

var connTimeout = 2 * time.Second

var _ = Describe("[Feature: Example]", func() {
	f := framework.NewDefaultFramework()

	Describe("TCP", func() {
		It("Expose a TCP echo server", func() {
			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			clientConf += fmt.Sprintf(`
			[tcp]
			type = tcp
			local_port = {{ .%s }}
			remote_port = {{ .%s }}
			`, framework.TCPEchoServerPort, framework.GenPortName("TCP"))

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.ExpectTCPRequest(f.UsedPorts[framework.GenPortName("TCP")], []byte(consts.TestString), []byte(consts.TestString), connTimeout)
		})
	})
})

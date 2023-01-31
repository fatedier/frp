package e2e

import (
	"fmt"

	"github.com/onsi/ginkgo"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
)

var _ = ginkgo.Describe("[Feature: Example]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.Describe("TCP", func() {
		ginkgo.It("Expose a TCP echo server", func() {
			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			remotePort := f.AllocPort()
			clientConf += fmt.Sprintf(`
			[tcp]
			type = tcp
			local_port = {{ .%s }}
			remote_port = %d
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()
		})
	})
})

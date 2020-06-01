package e2e

import (
	"fmt"
	"time"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"

	. "github.com/onsi/ginkgo"
)

var connTimeout = 5 * time.Second

var _ = Describe("[Feature: Example]", func() {
	f := framework.NewDefaultFramework()

	Describe("TCP", func() {
		It("Expose a TCP echo server", func() {
			serverConf := `
			[common]
			bind_port = {{ .PortServer }}
			`

			clientConf := fmt.Sprintf(`
			[common]
			server_port = {{ .PortServer }}

			[tcp]
			type = tcp
			local_port = {{ .%s }}
			remote_port = {{ .PortTCP }}
			`, framework.TCPEchoServerPort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.ExpectTCPReuqest(f.UsedPorts["PortTCP"], []byte(consts.TestString), []byte(consts.TestString), connTimeout)
		})
	})
})

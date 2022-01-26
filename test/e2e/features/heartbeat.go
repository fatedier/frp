package features

import (
	"fmt"
	"time"

	"github.com/fatedier/frp/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("[Feature: Heartbeat]", func() {
	f := framework.NewDefaultFramework()

	It("disable application layer heartbeat", func() {
		serverPort := f.AllocPort()
		serverConf := fmt.Sprintf(`
		[common]
		bind_addr = 0.0.0.0
		bind_port = %d
		heartbeat_timeout = -1
		tcp_mux_keepalive_interval = 2
		`, serverPort)

		remotePort := f.AllocPort()
		clientConf := fmt.Sprintf(`
		[common]
		server_port = %d
		log_level = trace
		heartbeat_interval = -1
		heartbeat_timeout = -1
		tcp_mux_keepalive_interval = 2

		[tcp]
		type = tcp
		local_port = %d
		remote_port = %d
		`, serverPort, f.PortByName(framework.TCPEchoServerPort), remotePort)

		// run frps and frpc
		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).Protocol("tcp").Port(remotePort).Ensure()

		time.Sleep(5 * time.Second)
		framework.NewRequestExpect(f).Protocol("tcp").Port(remotePort).Ensure()
	})
})

package features

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
)

var _ = ginkgo.Describe("[Feature: Heartbeat]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("disable application layer heartbeat", func() {
		serverPort := f.AllocPort()
		serverConf := fmt.Sprintf(`
		bindAddr = "0.0.0.0"
		bindPort = %d
		transport.heartbeatTimeout = -1
		transport.tcpMuxKeepaliveInterval = 2
		`, serverPort)

		remotePort := f.AllocPort()
		clientConf := fmt.Sprintf(`
		serverPort = %d
		log.level = "trace"
		transport.heartbeatInterval = -1
		transport.heartbeatTimeout = -1
		transport.tcpMuxKeepaliveInterval = 2

		[[proxies]]
		name = "tcp"
		type = "tcp"
		localPort = %d
		remotePort = %d
		`, serverPort, f.PortByName(framework.TCPEchoServerPort), remotePort)

		// run frps and frpc
		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).Protocol("tcp").Port(remotePort).Ensure()

		time.Sleep(5 * time.Second)
		framework.NewRequestExpect(f).Protocol("tcp").Port(remotePort).Ensure()
	})
})

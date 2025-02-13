package features

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
)

var _ = ginkgo.Describe("[Feature: Chaos]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("reconnect after frps restart", func() {
		serverPort := f.AllocPort()
		serverConfigPath := f.GenerateConfigFile(fmt.Sprintf(`
		bindAddr = "0.0.0.0"
		bindPort = %d
		`, serverPort))

		remotePort := f.AllocPort()
		clientConfigPath := f.GenerateConfigFile(fmt.Sprintf(`
		serverPort = %d
		log.level = "trace"

		[[proxies]]
		name = "tcp"
		type = "tcp"
		localPort = %d
		remotePort = %d
		`, serverPort, f.PortByName(framework.TCPEchoServerPort), remotePort))

		// 1. start frps and frpc, expect request success
		ps, _, err := f.RunFrps("-c", serverConfigPath)
		framework.ExpectNoError(err)

		pc, _, err := f.RunFrpc("-c", clientConfigPath)
		framework.ExpectNoError(err)
		framework.NewRequestExpect(f).Port(remotePort).Ensure()

		// 2. stop frps, expect request failed
		_ = ps.Stop()
		time.Sleep(200 * time.Millisecond)
		framework.NewRequestExpect(f).Port(remotePort).ExpectError(true).Ensure()

		// 3. restart frps, expect request success
		_, _, err = f.RunFrps("-c", serverConfigPath)
		framework.ExpectNoError(err)
		time.Sleep(2 * time.Second)
		framework.NewRequestExpect(f).Port(remotePort).Ensure()

		// 4. stop frpc, expect request failed
		_ = pc.Stop()
		time.Sleep(200 * time.Millisecond)
		framework.NewRequestExpect(f).Port(remotePort).ExpectError(true).Ensure()

		// 5. restart frpc, expect request success
		_, _, err = f.RunFrpc("-c", clientConfigPath)
		framework.ExpectNoError(err)
		time.Sleep(time.Second)
		framework.NewRequestExpect(f).Port(remotePort).Ensure()
	})
})

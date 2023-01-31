package basic

import (
	"fmt"

	"github.com/onsi/ginkgo"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/port"
)

var _ = ginkgo.Describe("[Feature: Config]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.Describe("Template", func() {
		ginkgo.It("render by env", func() {
			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			portName := port.GenName("TCP")
			serverConf += fmt.Sprintf(`
			token = {{ %s{{ .Envs.FRP_TOKEN }}%s }}
			`, "`", "`")

			clientConf += fmt.Sprintf(`
			token = {{ %s{{ .Envs.FRP_TOKEN }}%s }}

			[tcp]
			type = tcp
			local_port = {{ .%s }}
			remote_port = {{ .%s }}
			`, "`", "`", framework.TCPEchoServerPort, portName)

			f.SetEnvs([]string{"FRP_TOKEN=123"})
			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).PortName(portName).Ensure()
		})
	})

	ginkgo.Describe("Includes", func() {
		ginkgo.It("split tcp proxies into different files", func() {
			serverPort := f.AllocPort()
			serverConfigPath := f.GenerateConfigFile(fmt.Sprintf(`
			[common]
			bind_addr = 0.0.0.0
			bind_port = %d
			`, serverPort))

			remotePort := f.AllocPort()
			proxyConfigPath := f.GenerateConfigFile(fmt.Sprintf(`
			[tcp]
			type = tcp
			local_port = %d
			remote_port = %d
			`, f.PortByName(framework.TCPEchoServerPort), remotePort))

			remotePort2 := f.AllocPort()
			proxyConfigPath2 := f.GenerateConfigFile(fmt.Sprintf(`
			[tcp2]
			type = tcp
			local_port = %d
			remote_port = %d
			`, f.PortByName(framework.TCPEchoServerPort), remotePort2))

			clientConfigPath := f.GenerateConfigFile(fmt.Sprintf(`
			[common]
			server_port = %d
			includes = %s,%s
			`, serverPort, proxyConfigPath, proxyConfigPath2))

			_, _, err := f.RunFrps("-c", serverConfigPath)
			framework.ExpectNoError(err)

			_, _, err = f.RunFrpc("-c", clientConfigPath)
			framework.ExpectNoError(err)

			framework.NewRequestExpect(f).Port(remotePort).Ensure()
			framework.NewRequestExpect(f).Port(remotePort2).Ensure()
		})
	})
})

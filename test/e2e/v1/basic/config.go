package basic

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"

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
			auth.token = "{{ %s{{ .Envs.FRP_TOKEN }}%s }}"
			`, "`", "`")

			clientConf += fmt.Sprintf(`
			auth.token = "{{ %s{{ .Envs.FRP_TOKEN }}%s }}"

			[[proxies]]
			name = "tcp"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = {{ .%s }}
			`, "`", "`", framework.TCPEchoServerPort, portName)

			f.SetEnvs([]string{"FRP_TOKEN=123"})
			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).PortName(portName).Ensure()
		})

		ginkgo.It("Range ports mapping", func() {
			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			adminPort := f.AllocPort()

			localPortsRange := "13010-13012,13014"
			remotePortsRange := "23010-23012,23014"
			escapeTemplate := func(s string) string {
				return "{{ `" + s + "` }}"
			}
			clientConf += fmt.Sprintf(`
			webServer.port = %d

			%s
			[[proxies]]
			name = "tcp-%s"
			type = "tcp"
			localPort = %s
			remotePort = %s
			%s
			`, adminPort,
				escapeTemplate(fmt.Sprintf(`{{- range $_, $v := parseNumberRangePair "%s" "%s" }}`, localPortsRange, remotePortsRange)),
				escapeTemplate("{{ $v.First }}"),
				escapeTemplate("{{ $v.First }}"),
				escapeTemplate("{{ $v.Second }}"),
				escapeTemplate("{{- end }}"),
			)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			client := f.APIClientForFrpc(adminPort)
			checkProxyFn := func(name string, localPort, remotePort int) {
				status, err := client.GetProxyStatus(context.Background(), name)
				framework.ExpectNoError(err)

				framework.ExpectContainSubstring(status.LocalAddr, fmt.Sprintf(":%d", localPort))
				framework.ExpectContainSubstring(status.RemoteAddr, fmt.Sprintf(":%d", remotePort))
			}
			checkProxyFn("tcp-13010", 13010, 23010)
			checkProxyFn("tcp-13011", 13011, 23011)
			checkProxyFn("tcp-13012", 13012, 23012)
			checkProxyFn("tcp-13014", 13014, 23014)
		})
	})

	ginkgo.Describe("Includes", func() {
		ginkgo.It("split tcp proxies into different files", func() {
			serverPort := f.AllocPort()
			serverConfigPath := f.GenerateConfigFile(fmt.Sprintf(`
			bindAddr = "0.0.0.0"
			bindPort = %d
			`, serverPort))

			remotePort := f.AllocPort()
			proxyConfigPath := f.GenerateConfigFile(fmt.Sprintf(`
			[[proxies]]
			name = "tcp"
			type = "tcp"
			localPort = %d
			remotePort = %d
			`, f.PortByName(framework.TCPEchoServerPort), remotePort))

			remotePort2 := f.AllocPort()
			proxyConfigPath2 := f.GenerateConfigFile(fmt.Sprintf(`
			[[proxies]]
			name = "tcp2"
			type = "tcp"
			localPort = %d
			remotePort = %d
			`, f.PortByName(framework.TCPEchoServerPort), remotePort2))

			clientConfigPath := f.GenerateConfigFile(fmt.Sprintf(`
			serverPort = %d
			includes = ["%s","%s"]
			`, serverPort, proxyConfigPath, proxyConfigPath2))

			_, _, err := f.RunFrps("-c", serverConfigPath)
			framework.ExpectNoError(err)

			_, _, err = f.RunFrpc("-c", clientConfigPath)
			framework.ExpectNoError(err)

			framework.NewRequestExpect(f).Port(remotePort).Ensure()
			framework.NewRequestExpect(f).Port(remotePort2).Ensure()
		})
	})

	ginkgo.Describe("Support Formats", func() {
		ginkgo.It("YAML", func() {
			serverConf := fmt.Sprintf(`
bindPort: {{ .%s }}
log:
  level: trace
`, port.GenName("Server"))

			remotePort := f.AllocPort()
			clientConf := fmt.Sprintf(`
serverPort: {{ .%s }}
log:
  level: trace

proxies:
- name: tcp
  type: tcp
  localPort: {{ .%s }}
  remotePort: %d
`, port.GenName("Server"), framework.TCPEchoServerPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})
			framework.NewRequestExpect(f).Port(remotePort).Ensure()
		})

		ginkgo.It("JSON", func() {
			serverConf := fmt.Sprintf(`{"bindPort": {{ .%s }}, "log": {"level": "trace"}}`, port.GenName("Server"))

			remotePort := f.AllocPort()
			clientConf := fmt.Sprintf(`{"serverPort": {{ .%s }}, "log": {"level": "trace"},
"proxies": [{"name": "tcp", "type": "tcp", "localPort": {{ .%s }}, "remotePort": %d}]}`,
				port.GenName("Server"), framework.TCPEchoServerPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})
			framework.NewRequestExpect(f).Port(remotePort).Ensure()
		})
	})
})

package basic

import (
	"fmt"
	"strings"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/port"

	. "github.com/onsi/ginkgo"
)

type generalTestConfigures struct {
	server      string
	client      string
	expectError bool
}

// defineClientServerTest test a normal tcp and udp proxy with specified TestConfigures.
func defineClientServerTest(desc string, f *framework.Framework, configures *generalTestConfigures) {
	It(desc, func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		serverConf += fmt.Sprintf(`
				%s
				`, configures.server)

		tcpPortName := port.GenName("TCP")
		udpPortName := port.GenName("UDP")
		clientConf += fmt.Sprintf(`
				%s

				[tcp]
				type = tcp
				local_port = {{ .%s }}
				remote_port = {{ .%s }}

				[udp]
				type = udp
				local_port = {{ .%s }}
				remote_port = {{ .%s }}
				`, configures.client,
			framework.TCPEchoServerPort, tcpPortName,
			framework.UDPEchoServerPort, udpPortName,
		)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).PortName(tcpPortName).ExpectError(configures.expectError).Explain("tcp proxy").Ensure()
		framework.NewRequestExpect(f).RequestModify(framework.SetRequestProtocol("udp")).
			PortName(udpPortName).ExpectError(configures.expectError).Explain("udp proxy").Ensure()
	})
}

var _ = Describe("[Feature: Client-Server]", func() {
	f := framework.NewDefaultFramework()

	Describe("Protocol", func() {
		supportProtocols := []string{"tcp", "kcp", "websocket"}
		for _, protocol := range supportProtocols {
			configures := &generalTestConfigures{
				server: fmt.Sprintf(`
				kcp_bind_port = {{ .%s }}
				protocol = %s"
				`, consts.PortServerName, protocol),
				client: "protocol = " + protocol,
			}
			defineClientServerTest(protocol, f, configures)
		}
	})

	Describe("Authentication", func() {
		defineClientServerTest("Token Correct", f, &generalTestConfigures{
			server: "token = 123456",
			client: "token = 123456",
		})

		defineClientServerTest("Token Incorrect", f, &generalTestConfigures{
			server:      "token = 123456",
			client:      "token = invalid",
			expectError: true,
		})
	})

	Describe("TLS", func() {
		supportProtocols := []string{"tcp", "kcp", "websocket"}
		for _, protocol := range supportProtocols {
			tmp := protocol
			defineClientServerTest("TLS over "+strings.ToUpper(tmp), f, &generalTestConfigures{
				server: fmt.Sprintf(`
				kcp_bind_port = {{ .%s }}
				protocol = %s
				`, consts.PortServerName, protocol),
				client: fmt.Sprintf(`tls_enable = true
				protocol = %s
				`, protocol),
			})
		}

		defineClientServerTest("enable tls_only, client with TLS", f, &generalTestConfigures{
			server: "tls_only = true",
			client: "tls_enable = true",
		})
		defineClientServerTest("enable tls_only, client without TLS", f, &generalTestConfigures{
			server:      "tls_only = true",
			expectError: true,
		})
	})
})

package basic

import (
	"fmt"
	"strings"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/cert"
	"github.com/fatedier/frp/test/e2e/pkg/port"

	. "github.com/onsi/ginkgo"
)

type generalTestConfigures struct {
	server      string
	client      string
	expectError bool
}

func runClientServerTest(f *framework.Framework, configures *generalTestConfigures) {
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
	framework.NewRequestExpect(f).Protocol("udp").
		PortName(udpPortName).ExpectError(configures.expectError).Explain("udp proxy").Ensure()
}

// defineClientServerTest test a normal tcp and udp proxy with specified TestConfigures.
func defineClientServerTest(desc string, f *framework.Framework, configures *generalTestConfigures) {
	It(desc, func() {
		runClientServerTest(f, configures)
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

	Describe("TLS with custom certificate", func() {
		supportProtocols := []string{"tcp", "kcp", "websocket"}

		var (
			caCrtPath                    string
			serverCrtPath, serverKeyPath string
			clientCrtPath, clientKeyPath string
		)
		JustBeforeEach(func() {
			generator := &cert.SelfSignedCertGenerator{}
			artifacts, err := generator.Generate("0.0.0.0")
			framework.ExpectNoError(err)

			caCrtPath = f.WriteTempFile("ca.crt", string(artifacts.CACert))
			serverCrtPath = f.WriteTempFile("server.crt", string(artifacts.Cert))
			serverKeyPath = f.WriteTempFile("server.key", string(artifacts.Key))
			generator.SetCA(artifacts.CACert, artifacts.CAKey)
			generator.Generate("0.0.0.0")
			clientCrtPath = f.WriteTempFile("client.crt", string(artifacts.Cert))
			clientKeyPath = f.WriteTempFile("client.key", string(artifacts.Key))
		})

		for _, protocol := range supportProtocols {
			tmp := protocol

			It("one-way authentication: "+tmp, func() {
				runClientServerTest(f, &generalTestConfigures{
					server: fmt.Sprintf(`
						protocol = %s
						kcp_bind_port = {{ .%s }}
						tls_trusted_ca_file = %s
					`, tmp, consts.PortServerName, caCrtPath),
					client: fmt.Sprintf(`
						protocol = %s
						tls_enable = true
						tls_cert_file = %s
						tls_key_file = %s
					`, tmp, clientCrtPath, clientKeyPath),
				})
			})

			It("mutual authentication: "+tmp, func() {
				runClientServerTest(f, &generalTestConfigures{
					server: fmt.Sprintf(`
						protocol = %s
						kcp_bind_port = {{ .%s }}
						tls_cert_file = %s
						tls_key_file = %s
						tls_trusted_ca_file = %s
					`, tmp, consts.PortServerName, serverCrtPath, serverKeyPath, caCrtPath),
					client: fmt.Sprintf(`
						protocol = %s
						tls_enable = true
						tls_cert_file = %s
						tls_key_file = %s
						tls_trusted_ca_file = %s
					`, tmp, clientCrtPath, clientKeyPath, caCrtPath),
				})
			})
		}
	})

	Describe("TLS with custom certificate and specified server name", func() {
		var (
			caCrtPath                    string
			serverCrtPath, serverKeyPath string
			clientCrtPath, clientKeyPath string
		)
		JustBeforeEach(func() {
			generator := &cert.SelfSignedCertGenerator{}
			artifacts, err := generator.Generate("example.com")
			framework.ExpectNoError(err)

			caCrtPath = f.WriteTempFile("ca.crt", string(artifacts.CACert))
			serverCrtPath = f.WriteTempFile("server.crt", string(artifacts.Cert))
			serverKeyPath = f.WriteTempFile("server.key", string(artifacts.Key))
			generator.SetCA(artifacts.CACert, artifacts.CAKey)
			generator.Generate("example.com")
			clientCrtPath = f.WriteTempFile("client.crt", string(artifacts.Cert))
			clientKeyPath = f.WriteTempFile("client.key", string(artifacts.Key))
		})

		It("mutual authentication", func() {
			runClientServerTest(f, &generalTestConfigures{
				server: fmt.Sprintf(`
				tls_cert_file = %s
				tls_key_file = %s
				tls_trusted_ca_file = %s
				`, serverCrtPath, serverKeyPath, caCrtPath),
				client: fmt.Sprintf(`
				tls_enable = true
				tls_server_name = example.com
				tls_cert_file = %s
				tls_key_file = %s
				tls_trusted_ca_file = %s
				`, clientCrtPath, clientKeyPath, caCrtPath),
			})
		})

		It("mutual authentication with incorrect server name", func() {
			runClientServerTest(f, &generalTestConfigures{
				server: fmt.Sprintf(`
				tls_cert_file = %s
				tls_key_file = %s
				tls_trusted_ca_file = %s
				`, serverCrtPath, serverKeyPath, caCrtPath),
				client: fmt.Sprintf(`
				tls_enable = true
				tls_server_name = invalid.com
				tls_cert_file = %s
				tls_key_file = %s
				tls_trusted_ca_file = %s
				`, clientCrtPath, clientKeyPath, caCrtPath),
				expectError: true,
			})
		})
	})

	Describe("TLS with disable_custom_tls_first_byte", func() {
		supportProtocols := []string{"tcp", "kcp", "websocket"}
		for _, protocol := range supportProtocols {
			tmp := protocol
			defineClientServerTest("TLS over "+strings.ToUpper(tmp), f, &generalTestConfigures{
				server: fmt.Sprintf(`
					kcp_bind_port = {{ .%s }}
					protocol = %s
					`, consts.PortServerName, protocol),
				client: fmt.Sprintf(`
					tls_enable = true
					protocol = %s
					disable_custom_tls_first_byte = true
					`, protocol),
			})
		}
	})

	Describe("IPv6 bind address", func() {
		supportProtocols := []string{"tcp", "kcp", "websocket"}
		for _, protocol := range supportProtocols {
			tmp := protocol
			defineClientServerTest("IPv6 bind address: "+strings.ToUpper(tmp), f, &generalTestConfigures{
				server: fmt.Sprintf(`
					bind_addr = ::
					kcp_bind_port = {{ .%s }}
					protocol = %s
					`, consts.PortServerName, protocol),
				client: fmt.Sprintf(`
					tls_enable = true
					protocol = %s
					disable_custom_tls_first_byte = true
					`, protocol),
			})
		}
	})
})

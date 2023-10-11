package basic

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/cert"
	"github.com/fatedier/frp/test/e2e/pkg/port"
)

type generalTestConfigures struct {
	server        string
	client        string
	clientPrefix  string
	client2       string
	client2Prefix string
	testDelay     time.Duration
	expectError   bool
}

func renderBindPortConfig(protocol string) string {
	if protocol == "kcp" {
		return fmt.Sprintf(`kcpBindPort = {{ .%s }}`, consts.PortServerName)
	} else if protocol == "quic" {
		return fmt.Sprintf(`quicBindPort = {{ .%s }}`, consts.PortServerName)
	}
	return ""
}

func runClientServerTest(f *framework.Framework, configures *generalTestConfigures) {
	serverConf := consts.DefaultServerConfig
	clientConf := consts.DefaultClientConfig
	if configures.clientPrefix != "" {
		clientConf = configures.clientPrefix
	}

	serverConf += fmt.Sprintf(`
	%s
	`, configures.server)

	tcpPortName := port.GenName("TCP")
	udpPortName := port.GenName("UDP")
	clientConf += fmt.Sprintf(`
		%s

		[[proxies]]
		name = "tcp"
		type = "tcp"
		localPort = {{ .%s }}
		remotePort = {{ .%s }}

		[[proxies]]
		name = "udp"
		type = "udp"
		localPort = {{ .%s }}
		remotePort = {{ .%s }}
		`, configures.client,
		framework.TCPEchoServerPort, tcpPortName,
		framework.UDPEchoServerPort, udpPortName,
	)

	clientConfs := []string{clientConf}
	if configures.client2 != "" {
		client2Conf := consts.DefaultClientConfig
		if configures.client2Prefix != "" {
			client2Conf = configures.client2Prefix
		}
		client2Conf += fmt.Sprintf(`
			%s
		`, configures.client2)
		clientConfs = append(clientConfs, client2Conf)
	}

	f.RunProcesses([]string{serverConf}, clientConfs)

	if configures.testDelay > 0 {
		time.Sleep(configures.testDelay)
	}

	framework.NewRequestExpect(f).PortName(tcpPortName).ExpectError(configures.expectError).Explain("tcp proxy").Ensure()
	framework.NewRequestExpect(f).Protocol("udp").
		PortName(udpPortName).ExpectError(configures.expectError).Explain("udp proxy").Ensure()
}

// defineClientServerTest test a normal tcp and udp proxy with specified TestConfigures.
func defineClientServerTest(desc string, f *framework.Framework, configures *generalTestConfigures) {
	ginkgo.It(desc, func() {
		runClientServerTest(f, configures)
	})
}

var _ = ginkgo.Describe("[Feature: Client-Server]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.Describe("Protocol", func() {
		supportProtocols := []string{"tcp", "kcp", "quic", "websocket"}
		for _, protocol := range supportProtocols {
			configures := &generalTestConfigures{
				server: fmt.Sprintf(`
				%s
				`, renderBindPortConfig(protocol)),
				client: fmt.Sprintf(`transport.protocol = "%s"`, protocol),
			}
			defineClientServerTest(protocol, f, configures)
		}
	})

	// wss is special, it needs to be tested separately.
	// frps only supports ws, so there should be a proxy to terminate TLS before frps.
	ginkgo.Describe("Protocol wss", func() {
		wssPort := f.AllocPort()
		configures := &generalTestConfigures{
			clientPrefix: fmt.Sprintf(`
				serverAddr = "127.0.0.1"
				serverPort = %d
				loginFailExit = false
				transport.protocol = "wss"
				log.level = "trace"
			`, wssPort),
			// Due to the fact that frps cannot directly accept wss connections, we use the https2http plugin of another frpc to terminate TLS.
			client2: fmt.Sprintf(`
				[[proxies]]
				name = "wss2ws"
				type = "tcp"
				remotePort = %d
				[proxies.plugin]
				type = "https2http"
				localAddr = "127.0.0.1:{{ .%s }}"
			`, wssPort, consts.PortServerName),
			testDelay: 10 * time.Second,
		}

		defineClientServerTest("wss", f, configures)
	})

	ginkgo.Describe("Authentication", func() {
		defineClientServerTest("Token Correct", f, &generalTestConfigures{
			server: `auth.token = "123456"`,
			client: `auth.token = "123456"`,
		})

		defineClientServerTest("Token Incorrect", f, &generalTestConfigures{
			server:      `auth.token = "123456"`,
			client:      `auth.token = "invalid"`,
			expectError: true,
		})
	})

	ginkgo.Describe("TLS", func() {
		supportProtocols := []string{"tcp", "kcp", "quic", "websocket"}
		for _, protocol := range supportProtocols {
			tmp := protocol
			// Since v0.50.0, the default value of tls_enable has been changed to true.
			// Therefore, here it needs to be set as false to test the scenario of turning it off.
			defineClientServerTest("Disable TLS over "+strings.ToUpper(tmp), f, &generalTestConfigures{
				server: fmt.Sprintf(`
				%s
				`, renderBindPortConfig(protocol)),
				client: fmt.Sprintf(`transport.tls.enable = false
				transport.protocol = "%s"
				`, protocol),
			})
		}

		defineClientServerTest("enable tls force, client with TLS", f, &generalTestConfigures{
			server: "transport.tls.force = true",
		})
		defineClientServerTest("enable tls force, client without TLS", f, &generalTestConfigures{
			server:      "transport.tls.force = true",
			client:      "transport.tls.enable = false",
			expectError: true,
		})
	})

	ginkgo.Describe("TLS with custom certificate", func() {
		supportProtocols := []string{"tcp", "kcp", "quic", "websocket"}

		var (
			caCrtPath                    string
			serverCrtPath, serverKeyPath string
			clientCrtPath, clientKeyPath string
		)
		ginkgo.JustBeforeEach(func() {
			generator := &cert.SelfSignedCertGenerator{}
			artifacts, err := generator.Generate("127.0.0.1")
			framework.ExpectNoError(err)

			caCrtPath = f.WriteTempFile("ca.crt", string(artifacts.CACert))
			serverCrtPath = f.WriteTempFile("server.crt", string(artifacts.Cert))
			serverKeyPath = f.WriteTempFile("server.key", string(artifacts.Key))
			generator.SetCA(artifacts.CACert, artifacts.CAKey)
			_, err = generator.Generate("127.0.0.1")
			framework.ExpectNoError(err)
			clientCrtPath = f.WriteTempFile("client.crt", string(artifacts.Cert))
			clientKeyPath = f.WriteTempFile("client.key", string(artifacts.Key))
		})

		for _, protocol := range supportProtocols {
			tmp := protocol

			ginkgo.It("one-way authentication: "+tmp, func() {
				runClientServerTest(f, &generalTestConfigures{
					server: fmt.Sprintf(`
						%s
						transport.tls.trustedCaFile = "%s"
					`, renderBindPortConfig(tmp), caCrtPath),
					client: fmt.Sprintf(`
						transport.protocol = "%s"
						transport.tls.certFile = "%s"
						transport.tls.keyFile = "%s"
					`, tmp, clientCrtPath, clientKeyPath),
				})
			})

			ginkgo.It("mutual authentication: "+tmp, func() {
				runClientServerTest(f, &generalTestConfigures{
					server: fmt.Sprintf(`
						%s
						transport.tls.certFile = "%s"
						transport.tls.keyFile = "%s"
						transport.tls.trustedCaFile = "%s"
					`, renderBindPortConfig(tmp), serverCrtPath, serverKeyPath, caCrtPath),
					client: fmt.Sprintf(`
						transport.protocol = "%s"
						transport.tls.certFile = "%s"
						transport.tls.keyFile = "%s"
						transport.tls.trustedCaFile = "%s"
					`, tmp, clientCrtPath, clientKeyPath, caCrtPath),
				})
			})
		}
	})

	ginkgo.Describe("TLS with custom certificate and specified server name", func() {
		var (
			caCrtPath                    string
			serverCrtPath, serverKeyPath string
			clientCrtPath, clientKeyPath string
		)
		ginkgo.JustBeforeEach(func() {
			generator := &cert.SelfSignedCertGenerator{}
			artifacts, err := generator.Generate("example.com")
			framework.ExpectNoError(err)

			caCrtPath = f.WriteTempFile("ca.crt", string(artifacts.CACert))
			serverCrtPath = f.WriteTempFile("server.crt", string(artifacts.Cert))
			serverKeyPath = f.WriteTempFile("server.key", string(artifacts.Key))
			generator.SetCA(artifacts.CACert, artifacts.CAKey)
			_, err = generator.Generate("example.com")
			framework.ExpectNoError(err)
			clientCrtPath = f.WriteTempFile("client.crt", string(artifacts.Cert))
			clientKeyPath = f.WriteTempFile("client.key", string(artifacts.Key))
		})

		ginkgo.It("mutual authentication", func() {
			runClientServerTest(f, &generalTestConfigures{
				server: fmt.Sprintf(`
				transport.tls.certFile = "%s"
				transport.tls.keyFile = "%s"
				transport.tls.trustedCaFile = "%s"
				`, serverCrtPath, serverKeyPath, caCrtPath),
				client: fmt.Sprintf(`
				transport.tls.serverName = "example.com"
				transport.tls.certFile = "%s"
				transport.tls.keyFile = "%s"
				transport.tls.trustedCaFile = "%s"
				`, clientCrtPath, clientKeyPath, caCrtPath),
			})
		})

		ginkgo.It("mutual authentication with incorrect server name", func() {
			runClientServerTest(f, &generalTestConfigures{
				server: fmt.Sprintf(`
				transport.tls.certFile = "%s"
				transport.tls.keyFile = "%s"
				transport.tls.trustedCaFile = "%s"
				`, serverCrtPath, serverKeyPath, caCrtPath),
				client: fmt.Sprintf(`
				transport.tls.serverName = "invalid.com"
				transport.tls.certFile = "%s"
				transport.tls.keyFile = "%s"
				transport.tls.trustedCaFile = "%s"
				`, clientCrtPath, clientKeyPath, caCrtPath),
				expectError: true,
			})
		})
	})

	ginkgo.Describe("TLS with disableCustomTLSFirstByte set to false", func() {
		supportProtocols := []string{"tcp", "kcp", "quic", "websocket"}
		for _, protocol := range supportProtocols {
			tmp := protocol
			defineClientServerTest("TLS over "+strings.ToUpper(tmp), f, &generalTestConfigures{
				server: fmt.Sprintf(`
					%s
					`, renderBindPortConfig(protocol)),
				client: fmt.Sprintf(`
					transport.protocol = "%s"
					transport.tls.disableCustomTLSFirstByte = false
					`, protocol),
			})
		}
	})

	ginkgo.Describe("IPv6 bind address", func() {
		supportProtocols := []string{"tcp", "kcp", "quic", "websocket"}
		for _, protocol := range supportProtocols {
			tmp := protocol
			defineClientServerTest("IPv6 bind address: "+strings.ToUpper(tmp), f, &generalTestConfigures{
				server: fmt.Sprintf(`
					bindAddr = "::"
					%s
					`, renderBindPortConfig(protocol)),
				client: fmt.Sprintf(`
					transport.protocol = "%s"
					`, protocol),
			})
		}
	})

	ginkgo.Describe("Use same port for bindPort and vhostHTTPSPort", func() {
		supportProtocols := []string{"tcp", "kcp", "quic", "websocket"}
		for _, protocol := range supportProtocols {
			tmp := protocol
			defineClientServerTest("Use same port for bindPort and vhostHTTPSPort: "+strings.ToUpper(tmp), f, &generalTestConfigures{
				server: fmt.Sprintf(`
					vhostHTTPSPort = {{ .%s }}
					%s
					`, consts.PortServerName, renderBindPortConfig(protocol)),
				// transport.tls.disableCustomTLSFirstByte should set to false when vhostHTTPSPort is same as bindPort
				client: fmt.Sprintf(`
					transport.protocol = "%s"
					transport.tls.disableCustomTLSFirstByte = false
					`, protocol),
			})
		}
	})
})

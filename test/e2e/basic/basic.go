package basic

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/httpserver"
	"github.com/fatedier/frp/test/e2e/mock/server/streamserver"
	"github.com/fatedier/frp/test/e2e/pkg/port"
	"github.com/fatedier/frp/test/e2e/pkg/request"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("[Feature: Basic]", func() {
	f := framework.NewDefaultFramework()

	Describe("TCP && UDP", func() {
		types := []string{"tcp", "udp"}
		for _, t := range types {
			proxyType := t
			It(fmt.Sprintf("Expose a %s echo server", strings.ToUpper(proxyType)), func() {
				serverConf := consts.DefaultServerConfig
				clientConf := consts.DefaultClientConfig

				localPortName := ""
				protocol := "tcp"
				switch proxyType {
				case "tcp":
					localPortName = framework.TCPEchoServerPort
					protocol = "tcp"
				case "udp":
					localPortName = framework.UDPEchoServerPort
					protocol = "udp"
				}
				getProxyConf := func(proxyName string, portName string, extra string) string {
					return fmt.Sprintf(`
				[%s]
				type = %s
				local_port = {{ .%s }}
				remote_port = {{ .%s }}
				`+extra, proxyName, proxyType, localPortName, portName)
				}

				tests := []struct {
					proxyName   string
					portName    string
					extraConfig string
				}{
					{
						proxyName: "normal",
						portName:  port.GenName("Normal"),
					},
					{
						proxyName:   "with-encryption",
						portName:    port.GenName("WithEncryption"),
						extraConfig: "use_encryption = true",
					},
					{
						proxyName:   "with-compression",
						portName:    port.GenName("WithCompression"),
						extraConfig: "use_compression = true",
					},
					{
						proxyName: "with-encryption-and-compression",
						portName:  port.GenName("WithEncryptionAndCompression"),
						extraConfig: `
						use_encryption = true
						use_compression = true
						`,
					},
				}

				// build all client config
				for _, test := range tests {
					clientConf += getProxyConf(test.proxyName, test.portName, test.extraConfig) + "\n"
				}
				// run frps and frpc
				f.RunProcesses([]string{serverConf}, []string{clientConf})

				for _, test := range tests {
					framework.NewRequestExpect(f).
						Protocol(protocol).
						PortName(test.portName).
						Explain(test.proxyName).
						Ensure()
				}
			})
		}
	})

	Describe("HTTP", func() {
		It("proxy to HTTP server", func() {
			serverConf := consts.DefaultServerConfig
			vhostHTTPPort := f.AllocPort()
			serverConf += fmt.Sprintf(`
			vhost_http_port = %d
			`, vhostHTTPPort)

			clientConf := consts.DefaultClientConfig

			getProxyConf := func(proxyName string, customDomains string, extra string) string {
				return fmt.Sprintf(`
				[%s]
				type = http
				local_port = {{ .%s }}
				custom_domains = %s
				`+extra, proxyName, framework.HTTPSimpleServerPort, customDomains)
			}

			tests := []struct {
				proxyName     string
				customDomains string
				extraConfig   string
			}{
				{
					proxyName: "normal",
				},
				{
					proxyName:   "with-encryption",
					extraConfig: "use_encryption = true",
				},
				{
					proxyName:   "with-compression",
					extraConfig: "use_compression = true",
				},
				{
					proxyName: "with-encryption-and-compression",
					extraConfig: `
						use_encryption = true
						use_compression = true
						`,
				},
				{
					proxyName:     "multiple-custom-domains",
					customDomains: "a.example.com, b.example.com",
				},
			}

			// build all client config
			for i, test := range tests {
				if tests[i].customDomains == "" {
					tests[i].customDomains = test.proxyName + ".example.com"
				}
				clientConf += getProxyConf(test.proxyName, tests[i].customDomains, test.extraConfig) + "\n"
			}
			// run frps and frpc
			f.RunProcesses([]string{serverConf}, []string{clientConf})

			for _, test := range tests {
				for _, domain := range strings.Split(test.customDomains, ",") {
					domain = strings.TrimSpace(domain)
					framework.NewRequestExpect(f).
						Explain(test.proxyName + "-" + domain).
						Port(vhostHTTPPort).
						RequestModify(func(r *request.Request) {
							r.HTTP().HTTPHost(domain)
						}).
						Ensure()
				}
			}

			// not exist host
			framework.NewRequestExpect(f).
				Explain("not exist host").
				Port(vhostHTTPPort).
				RequestModify(func(r *request.Request) {
					r.HTTP().HTTPHost("not-exist.example.com")
				}).
				Ensure(framework.ExpectResponseCode(404))
		})
	})

	Describe("HTTPS", func() {
		It("proxy to HTTPS server", func() {
			serverConf := consts.DefaultServerConfig
			vhostHTTPSPort := f.AllocPort()
			serverConf += fmt.Sprintf(`
			vhost_https_port = %d
			`, vhostHTTPSPort)

			localPort := f.AllocPort()
			clientConf := consts.DefaultClientConfig
			getProxyConf := func(proxyName string, customDomains string, extra string) string {
				return fmt.Sprintf(`
				[%s]
				type = https
				local_port = %d
				custom_domains = %s
				`+extra, proxyName, localPort, customDomains)
			}

			tests := []struct {
				proxyName     string
				customDomains string
				extraConfig   string
			}{
				{
					proxyName: "normal",
				},
				{
					proxyName:   "with-encryption",
					extraConfig: "use_encryption = true",
				},
				{
					proxyName:   "with-compression",
					extraConfig: "use_compression = true",
				},
				{
					proxyName: "with-encryption-and-compression",
					extraConfig: `
						use_encryption = true
						use_compression = true
						`,
				},
				{
					proxyName:     "multiple-custom-domains",
					customDomains: "a.example.com, b.example.com",
				},
			}

			// build all client config
			for i, test := range tests {
				if tests[i].customDomains == "" {
					tests[i].customDomains = test.proxyName + ".example.com"
				}
				clientConf += getProxyConf(test.proxyName, tests[i].customDomains, test.extraConfig) + "\n"
			}
			// run frps and frpc
			f.RunProcesses([]string{serverConf}, []string{clientConf})

			tlsConfig, err := transport.NewServerTLSConfig("", "", "")
			framework.ExpectNoError(err)
			localServer := httpserver.New(
				httpserver.WithBindPort(localPort),
				httpserver.WithTlsConfig(tlsConfig),
				httpserver.WithResponse([]byte("test")),
			)
			f.RunServer("", localServer)

			for _, test := range tests {
				for _, domain := range strings.Split(test.customDomains, ",") {
					domain = strings.TrimSpace(domain)
					framework.NewRequestExpect(f).
						Explain(test.proxyName + "-" + domain).
						Port(vhostHTTPSPort).
						RequestModify(func(r *request.Request) {
							r.HTTPS().HTTPHost(domain).TLSConfig(&tls.Config{
								ServerName:         domain,
								InsecureSkipVerify: true,
							})
						}).
						ExpectResp([]byte("test")).
						Ensure()
				}
			}

			// not exist host
			notExistDomain := "not-exist.example.com"
			framework.NewRequestExpect(f).
				Explain("not exist host").
				Port(vhostHTTPSPort).
				RequestModify(func(r *request.Request) {
					r.HTTPS().HTTPHost(notExistDomain).TLSConfig(&tls.Config{
						ServerName:         notExistDomain,
						InsecureSkipVerify: true,
					})
				}).
				ExpectError(true).
				Ensure()
		})
	})

	Describe("STCP && SUDP", func() {
		types := []string{"stcp", "sudp"}
		for _, t := range types {
			proxyType := t
			It(fmt.Sprintf("Expose echo server with %s", strings.ToUpper(proxyType)), func() {
				serverConf := consts.DefaultServerConfig
				clientServerConf := consts.DefaultClientConfig
				clientVisitorConf := consts.DefaultClientConfig

				localPortName := ""
				protocol := "tcp"
				switch proxyType {
				case "stcp":
					localPortName = framework.TCPEchoServerPort
					protocol = "tcp"
				case "sudp":
					localPortName = framework.UDPEchoServerPort
					protocol = "udp"
				}

				correctSK := "abc"
				wrongSK := "123"

				getProxyServerConf := func(proxyName string, extra string) string {
					return fmt.Sprintf(`
				[%s]
				type = %s
				role = server
				sk = %s
				local_port = {{ .%s }}
				`+extra, proxyName, proxyType, correctSK, localPortName)
				}
				getProxyVisitorConf := func(proxyName string, portName, visitorSK, extra string) string {
					return fmt.Sprintf(`
				[%s]
				type = %s
				role = visitor
				server_name = %s
				sk = %s
				bind_port = {{ .%s }}
				`+extra, proxyName, proxyType, proxyName, visitorSK, portName)
				}

				tests := []struct {
					proxyName    string
					bindPortName string
					visitorSK    string
					extraConfig  string
					expectError  bool
				}{
					{
						proxyName:    "normal",
						bindPortName: port.GenName("Normal"),
						visitorSK:    correctSK,
					},
					{
						proxyName:    "with-encryption",
						bindPortName: port.GenName("WithEncryption"),
						visitorSK:    correctSK,
						extraConfig:  "use_encryption = true",
					},
					{
						proxyName:    "with-compression",
						bindPortName: port.GenName("WithCompression"),
						visitorSK:    correctSK,
						extraConfig:  "use_compression = true",
					},
					{
						proxyName:    "with-encryption-and-compression",
						bindPortName: port.GenName("WithEncryptionAndCompression"),
						visitorSK:    correctSK,
						extraConfig: `
						use_encryption = true
						use_compression = true
						`,
					},
					{
						proxyName:    "with-error-sk",
						bindPortName: port.GenName("WithErrorSK"),
						visitorSK:    wrongSK,
						expectError:  true,
					},
				}

				// build all client config
				for _, test := range tests {
					clientServerConf += getProxyServerConf(test.proxyName, test.extraConfig) + "\n"
				}
				for _, test := range tests {
					clientVisitorConf += getProxyVisitorConf(test.proxyName, test.bindPortName, test.visitorSK, test.extraConfig) + "\n"
				}
				// run frps and frpc
				f.RunProcesses([]string{serverConf}, []string{clientServerConf, clientVisitorConf})

				for _, test := range tests {
					framework.NewRequestExpect(f).
						Protocol(protocol).
						PortName(test.bindPortName).
						Explain(test.proxyName).
						ExpectError(test.expectError).
						Ensure()
				}
			})
		}
	})

	Describe("TCPMUX", func() {
		It("Type tcpmux", func() {
			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			tcpmuxHTTPConnectPortName := port.GenName("TCPMUX")
			serverConf += fmt.Sprintf(`
			tcpmux_httpconnect_port = {{ .%s }}
			`, tcpmuxHTTPConnectPortName)

			getProxyConf := func(proxyName string, extra string) string {
				return fmt.Sprintf(`
				[%s]
				type = tcpmux
				multiplexer = httpconnect
				local_port = {{ .%s }}
				custom_domains = %s
				`+extra, proxyName, port.GenName(proxyName), proxyName)
			}

			tests := []struct {
				proxyName   string
				extraConfig string
			}{
				{
					proxyName: "normal",
				},
				{
					proxyName:   "with-encryption",
					extraConfig: "use_encryption = true",
				},
				{
					proxyName:   "with-compression",
					extraConfig: "use_compression = true",
				},
				{
					proxyName: "with-encryption-and-compression",
					extraConfig: `
						use_encryption = true
						use_compression = true
					`,
				},
			}

			// build all client config
			for _, test := range tests {
				clientConf += getProxyConf(test.proxyName, test.extraConfig) + "\n"

				localServer := streamserver.New(streamserver.TCP, streamserver.WithBindPort(f.AllocPort()), streamserver.WithRespContent([]byte(test.proxyName)))
				f.RunServer(port.GenName(test.proxyName), localServer)
			}

			// run frps and frpc
			f.RunProcesses([]string{serverConf}, []string{clientConf})

			// Request without HTTP connect should get error
			framework.NewRequestExpect(f).
				PortName(tcpmuxHTTPConnectPortName).
				ExpectError(true).
				Explain("request without HTTP connect expect error").
				Ensure()

			proxyURL := fmt.Sprintf("http://127.0.0.1:%d", f.PortByName(tcpmuxHTTPConnectPortName))
			// Request with incorrect connect hostname
			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.Addr("invalid").Proxy(proxyURL)
			}).ExpectError(true).Explain("request without HTTP connect expect error").Ensure()

			// Request with correct connect hostname
			for _, test := range tests {
				framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
					r.Addr(test.proxyName).Proxy(proxyURL)
				}).ExpectResp([]byte(test.proxyName)).Explain(test.proxyName).Ensure()
			}
		})
	})
})

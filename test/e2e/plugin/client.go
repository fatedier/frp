package plugin

import (
	"crypto/tls"
	"fmt"
	"strconv"

	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/httpserver"
	"github.com/fatedier/frp/test/e2e/pkg/cert"
	"github.com/fatedier/frp/test/e2e/pkg/port"
	"github.com/fatedier/frp/test/e2e/pkg/request"
	"github.com/fatedier/frp/test/e2e/pkg/utils"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("[Feature: Client-Plugins]", func() {
	f := framework.NewDefaultFramework()

	Describe("UnixDomainSocket", func() {
		It("Expose a unix domain socket echo server", func() {
			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			getProxyConf := func(proxyName string, portName string, extra string) string {
				return fmt.Sprintf(`
				[%s]
				type = tcp
				remote_port = {{ .%s }}
				plugin = unix_domain_socket
				plugin_unix_path = {{ .%s }}
				`+extra, proxyName, portName, framework.UDSEchoServerAddr)
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
				framework.NewRequestExpect(f).Port(f.PortByName(test.portName)).Ensure()
			}
		})
	})

	It("http_proxy", func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		remotePort := f.AllocPort()
		clientConf += fmt.Sprintf(`
		[tcp]
		type = tcp
		remote_port = %d
		plugin = http_proxy
		plugin_http_user = abc
		plugin_http_passwd = 123
		`, remotePort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		// http proxy, no auth info
		framework.NewRequestExpect(f).PortName(framework.HTTPSimpleServerPort).RequestModify(func(r *request.Request) {
			r.HTTP().Proxy("http://127.0.0.1:" + strconv.Itoa(remotePort))
		}).Ensure(framework.ExpectResponseCode(407))

		// http proxy, correct auth
		framework.NewRequestExpect(f).PortName(framework.HTTPSimpleServerPort).RequestModify(func(r *request.Request) {
			r.HTTP().Proxy("http://abc:123@127.0.0.1:" + strconv.Itoa(remotePort))
		}).Ensure()

		// connect TCP server by CONNECT method
		framework.NewRequestExpect(f).PortName(framework.TCPEchoServerPort).RequestModify(func(r *request.Request) {
			r.TCP().Proxy("http://abc:123@127.0.0.1:" + strconv.Itoa(remotePort))
		})
	})

	It("socks5 proxy", func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		remotePort := f.AllocPort()
		clientConf += fmt.Sprintf(`
		[tcp]
		type = tcp
		remote_port = %d
		plugin = socks5
		plugin_user = abc
		plugin_passwd = 123
		`, remotePort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		// http proxy, no auth info
		framework.NewRequestExpect(f).PortName(framework.TCPEchoServerPort).RequestModify(func(r *request.Request) {
			r.TCP().Proxy("socks5://127.0.0.1:" + strconv.Itoa(remotePort))
		}).ExpectError(true).Ensure()

		// http proxy, correct auth
		framework.NewRequestExpect(f).PortName(framework.TCPEchoServerPort).RequestModify(func(r *request.Request) {
			r.TCP().Proxy("socks5://abc:123@127.0.0.1:" + strconv.Itoa(remotePort))
		}).Ensure()
	})

	It("static_file", func() {
		vhostPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		vhost_http_port = %d
		`, vhostPort)
		clientConf := consts.DefaultClientConfig

		remotePort := f.AllocPort()
		f.WriteTempFile("test_static_file", "foo")
		clientConf += fmt.Sprintf(`
		[tcp]
		type = tcp
		remote_port = %d
		plugin = static_file
		plugin_local_path = %s

		[http]
		type = http
		custom_domains = example.com
		plugin = static_file
		plugin_local_path = %s

		[http-with-auth]
		type = http
		custom_domains = other.example.com
		plugin = static_file
		plugin_local_path = %s
		plugin_http_user = abc
		plugin_http_passwd = 123
		`, remotePort, f.TempDirectory, f.TempDirectory, f.TempDirectory)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		// from tcp proxy
		framework.NewRequestExpect(f).Request(
			framework.NewHTTPRequest().HTTPPath("/test_static_file").Port(remotePort),
		).ExpectResp([]byte("foo")).Ensure()

		// from http proxy without auth
		framework.NewRequestExpect(f).Request(
			framework.NewHTTPRequest().HTTPHost("example.com").HTTPPath("/test_static_file").Port(vhostPort),
		).ExpectResp([]byte("foo")).Ensure()

		// from http proxy with auth
		framework.NewRequestExpect(f).Request(
			framework.NewHTTPRequest().HTTPHost("other.example.com").HTTPPath("/test_static_file").Port(vhostPort).HTTPHeaders(map[string]string{
				"Authorization": utils.BasicAuth("abc", "123"),
			}),
		).ExpectResp([]byte("foo")).Ensure()
	})

	It("http2https", func() {
		serverConf := consts.DefaultServerConfig
		vhostHTTPPort := f.AllocPort()
		serverConf += fmt.Sprintf(`
		vhost_http_port = %d
		`, vhostHTTPPort)

		localPort := f.AllocPort()
		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[http2https]
		type = http
		custom_domains = example.com
		plugin = http2https
		plugin_local_addr = 127.0.0.1:%d
		`, localPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		tlsConfig, err := transport.NewServerTLSConfig("", "", "")
		framework.ExpectNoError(err)
		localServer := httpserver.New(
			httpserver.WithBindPort(localPort),
			httpserver.WithTlsConfig(tlsConfig),
			httpserver.WithResponse([]byte("test")),
		)
		f.RunServer("", localServer)

		framework.NewRequestExpect(f).
			Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("example.com")
			}).
			ExpectResp([]byte("test")).
			Ensure()
	})

	It("https2http", func() {
		generator := &cert.SelfSignedCertGenerator{}
		artifacts, err := generator.Generate("example.com")
		framework.ExpectNoError(err)
		crtPath := f.WriteTempFile("server.crt", string(artifacts.Cert))
		keyPath := f.WriteTempFile("server.key", string(artifacts.Key))

		serverConf := consts.DefaultServerConfig
		vhostHTTPSPort := f.AllocPort()
		serverConf += fmt.Sprintf(`
		vhost_https_port = %d
		`, vhostHTTPSPort)

		localPort := f.AllocPort()
		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[https2http]
		type = https
		custom_domains = example.com
		plugin = https2http
		plugin_local_addr = 127.0.0.1:%d
		plugin_crt_path = %s
		plugin_key_path = %s
		`, localPort, crtPath, keyPath)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		localServer := httpserver.New(
			httpserver.WithBindPort(localPort),
			httpserver.WithResponse([]byte("test")),
		)
		f.RunServer("", localServer)

		framework.NewRequestExpect(f).
			Port(vhostHTTPSPort).
			RequestModify(func(r *request.Request) {
				r.HTTPS().HTTPHost("example.com").TLSConfig(&tls.Config{
					ServerName:         "example.com",
					InsecureSkipVerify: true,
				})
			}).
			ExpectResp([]byte("test")).
			Ensure()
	})

	It("https2https", func() {
		generator := &cert.SelfSignedCertGenerator{}
		artifacts, err := generator.Generate("example.com")
		framework.ExpectNoError(err)
		crtPath := f.WriteTempFile("server.crt", string(artifacts.Cert))
		keyPath := f.WriteTempFile("server.key", string(artifacts.Key))

		serverConf := consts.DefaultServerConfig
		vhostHTTPSPort := f.AllocPort()
		serverConf += fmt.Sprintf(`
		vhost_https_port = %d
		`, vhostHTTPSPort)

		localPort := f.AllocPort()
		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[https2https]
		type = https
		custom_domains = example.com
		plugin = https2https
		plugin_local_addr = 127.0.0.1:%d
		plugin_crt_path = %s
		plugin_key_path = %s
		`, localPort, crtPath, keyPath)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		tlsConfig, err := transport.NewServerTLSConfig("", "", "")
		framework.ExpectNoError(err)
		localServer := httpserver.New(
			httpserver.WithBindPort(localPort),
			httpserver.WithResponse([]byte("test")),
			httpserver.WithTlsConfig(tlsConfig),
		)
		f.RunServer("", localServer)

		framework.NewRequestExpect(f).
			Port(vhostHTTPSPort).
			RequestModify(func(r *request.Request) {
				r.HTTPS().HTTPHost("example.com").TLSConfig(&tls.Config{
					ServerName:         "example.com",
					InsecureSkipVerify: true,
				})
			}).
			ExpectResp([]byte("test")).
			Ensure()
	})
})

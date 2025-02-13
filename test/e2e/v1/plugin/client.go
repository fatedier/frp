package plugin

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/httpserver"
	"github.com/fatedier/frp/test/e2e/pkg/cert"
	"github.com/fatedier/frp/test/e2e/pkg/port"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

var _ = ginkgo.Describe("[Feature: Client-Plugins]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.Describe("UnixDomainSocket", func() {
		ginkgo.It("Expose a unix domain socket echo server", func() {
			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			getProxyConf := func(proxyName string, portName string, extra string) string {
				return fmt.Sprintf(`
				[[proxies]]
				name = "%s"
				type = "tcp"
				remotePort = {{ .%s }}
				`+extra, proxyName, portName) + fmt.Sprintf(`
				[proxies.plugin]
				type = "unix_domain_socket"
				unixPath = "{{ .%s }}"
				`, framework.UDSEchoServerAddr)
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
					extraConfig: "transport.useEncryption = true",
				},
				{
					proxyName:   "with-compression",
					portName:    port.GenName("WithCompression"),
					extraConfig: "transport.useCompression = true",
				},
				{
					proxyName: "with-encryption-and-compression",
					portName:  port.GenName("WithEncryptionAndCompression"),
					extraConfig: `
					transport.useEncryption = true
					transport.useCompression = true
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

	ginkgo.It("http_proxy", func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		remotePort := f.AllocPort()
		clientConf += fmt.Sprintf(`
		[[proxies]]
		name = "tcp"
		type = "tcp"
		remotePort = %d
		[proxies.plugin]
		type = "http_proxy"
		httpUser = "abc"
		httpPassword = "123"
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

	ginkgo.It("socks5 proxy", func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		remotePort := f.AllocPort()
		clientConf += fmt.Sprintf(`
		[[proxies]]
		name = "tcp"
		type = "tcp"
		remotePort = %d
		[proxies.plugin]
		type = "socks5"
		username = "abc"
		password = "123"
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

	ginkgo.It("static_file", func() {
		vhostPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		vhostHTTPPort = %d
		`, vhostPort)
		clientConf := consts.DefaultClientConfig

		remotePort := f.AllocPort()
		f.WriteTempFile("test_static_file", "foo")
		clientConf += fmt.Sprintf(`
		[[proxies]]
		name = "tcp"
		type = "tcp"
		remotePort = %d
		[proxies.plugin]
		type = "static_file"
		localPath = "%s"

		[[proxies]]
		name = "http"
		type = "http"
		customDomains = ["example.com"]
		[proxies.plugin]
		type = "static_file"
		localPath = "%s"

		[[proxies]]
		name = "http-with-auth"
		type = "http"
		customDomains = ["other.example.com"]
		[proxies.plugin]
		type = "static_file"
		localPath = "%s"
		httpUser = "abc"
		httpPassword = "123"
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
			framework.NewHTTPRequest().HTTPHost("other.example.com").HTTPPath("/test_static_file").Port(vhostPort).HTTPAuth("abc", "123"),
		).ExpectResp([]byte("foo")).Ensure()
	})

	ginkgo.It("http2https", func() {
		serverConf := consts.DefaultServerConfig
		vhostHTTPPort := f.AllocPort()
		serverConf += fmt.Sprintf(`
		vhostHTTPPort = %d
		`, vhostHTTPPort)

		localPort := f.AllocPort()
		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[[proxies]]
		name = "http2https"
		type = "http"
		customDomains = ["example.com"]
		[proxies.plugin]
		type = "http2https"
		localAddr = "127.0.0.1:%d"
		`, localPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		tlsConfig, err := transport.NewServerTLSConfig("", "", "")
		framework.ExpectNoError(err)
		localServer := httpserver.New(
			httpserver.WithBindPort(localPort),
			httpserver.WithTLSConfig(tlsConfig),
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

	ginkgo.It("https2http", func() {
		generator := &cert.SelfSignedCertGenerator{}
		artifacts, err := generator.Generate("example.com")
		framework.ExpectNoError(err)
		crtPath := f.WriteTempFile("server.crt", string(artifacts.Cert))
		keyPath := f.WriteTempFile("server.key", string(artifacts.Key))

		serverConf := consts.DefaultServerConfig
		vhostHTTPSPort := f.AllocPort()
		serverConf += fmt.Sprintf(`
		vhostHTTPSPort = %d
		`, vhostHTTPSPort)

		localPort := f.AllocPort()
		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[[proxies]]
		name = "https2http"
		type = "https"
		customDomains = ["example.com"]
		[proxies.plugin]
		type = "https2http"
		localAddr = "127.0.0.1:%d"
		crtPath = "%s"
		keyPath = "%s"
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

	ginkgo.It("https2https", func() {
		generator := &cert.SelfSignedCertGenerator{}
		artifacts, err := generator.Generate("example.com")
		framework.ExpectNoError(err)
		crtPath := f.WriteTempFile("server.crt", string(artifacts.Cert))
		keyPath := f.WriteTempFile("server.key", string(artifacts.Key))

		serverConf := consts.DefaultServerConfig
		vhostHTTPSPort := f.AllocPort()
		serverConf += fmt.Sprintf(`
		vhostHTTPSPort = %d
		`, vhostHTTPSPort)

		localPort := f.AllocPort()
		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[[proxies]]
		name = "https2https"
		type = "https"
		customDomains = ["example.com"]
		[proxies.plugin]
		type = "https2https"
		localAddr = "127.0.0.1:%d"
		crtPath = "%s"
		keyPath = "%s"
		`, localPort, crtPath, keyPath)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		tlsConfig, err := transport.NewServerTLSConfig("", "", "")
		framework.ExpectNoError(err)
		localServer := httpserver.New(
			httpserver.WithBindPort(localPort),
			httpserver.WithResponse([]byte("test")),
			httpserver.WithTLSConfig(tlsConfig),
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

	ginkgo.Describe("http2http", func() {
		ginkgo.It("host header rewrite", func() {
			serverConf := consts.DefaultServerConfig

			localPort := f.AllocPort()
			remotePort := f.AllocPort()
			clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			[[proxies]]
			name = "http2http"
			type = "tcp"
			remotePort = %d
			[proxies.plugin]
			type = "http2http"
			localAddr = "127.0.0.1:%d"
			hostHeaderRewrite = "rewrite.test.com"
			`, remotePort, localPort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			localServer := httpserver.New(
				httpserver.WithBindPort(localPort),
				httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					_, _ = w.Write([]byte(req.Host))
				})),
			)
			f.RunServer("", localServer)

			framework.NewRequestExpect(f).
				Port(remotePort).
				RequestModify(func(r *request.Request) {
					r.HTTP().HTTPHost("example.com")
				}).
				ExpectResp([]byte("rewrite.test.com")).
				Ensure()
		})

		ginkgo.It("set request header", func() {
			serverConf := consts.DefaultServerConfig

			localPort := f.AllocPort()
			remotePort := f.AllocPort()
			clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			[[proxies]]
			name = "http2http"
			type = "tcp"
			remotePort = %d
			[proxies.plugin]
			type = "http2http"
			localAddr = "127.0.0.1:%d"
			requestHeaders.set.x-from-where = "frp"
			`, remotePort, localPort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			localServer := httpserver.New(
				httpserver.WithBindPort(localPort),
				httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					_, _ = w.Write([]byte(req.Header.Get("x-from-where")))
				})),
			)
			f.RunServer("", localServer)

			framework.NewRequestExpect(f).
				Port(remotePort).
				RequestModify(func(r *request.Request) {
					r.HTTP().HTTPHost("example.com")
				}).
				ExpectResp([]byte("frp")).
				Ensure()
		})
	})

	ginkgo.It("tls2raw", func() {
		generator := &cert.SelfSignedCertGenerator{}
		artifacts, err := generator.Generate("example.com")
		framework.ExpectNoError(err)
		crtPath := f.WriteTempFile("tls2raw_server.crt", string(artifacts.Cert))
		keyPath := f.WriteTempFile("tls2raw_server.key", string(artifacts.Key))

		serverConf := consts.DefaultServerConfig
		vhostHTTPSPort := f.AllocPort()
		serverConf += fmt.Sprintf(`
		vhostHTTPSPort = %d
		`, vhostHTTPSPort)

		localPort := f.AllocPort()
		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[[proxies]]
		name = "tls2raw-test"
		type = "https"
		customDomains = ["example.com"]
		[proxies.plugin]
		type = "tls2raw"
		localAddr = "127.0.0.1:%d"
		crtPath = "%s"
		keyPath = "%s"
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
})

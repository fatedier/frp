package features

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/httpserver"
	"github.com/fatedier/frp/test/e2e/mock/server/streamserver"
	"github.com/fatedier/frp/test/e2e/pkg/request"
	"github.com/fatedier/frp/test/e2e/pkg/ssh"
)

var _ = ginkgo.Describe("[Feature: SSH Tunnel]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("tcp", func() {
		sshPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		sshTunnelGateway.bindPort = %d
		`, sshPort)

		f.RunProcesses([]string{serverConf}, nil)

		localPort := f.PortByName(framework.TCPEchoServerPort)
		remotePort := f.AllocPort()
		tc := ssh.NewTunnelClient(
			fmt.Sprintf("127.0.0.1:%d", localPort),
			fmt.Sprintf("127.0.0.1:%d", sshPort),
			fmt.Sprintf("tcp --remote-port %d", remotePort),
		)
		framework.ExpectNoError(tc.Start())
		defer tc.Close()

		time.Sleep(time.Second)
		framework.NewRequestExpect(f).Port(remotePort).Ensure()
	})

	ginkgo.It("http", func() {
		sshPort := f.AllocPort()
		vhostPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		vhostHTTPPort = %d
		sshTunnelGateway.bindPort = %d
		`, vhostPort, sshPort)

		f.RunProcesses([]string{serverConf}, nil)

		localPort := f.PortByName(framework.HTTPSimpleServerPort)
		tc := ssh.NewTunnelClient(
			fmt.Sprintf("127.0.0.1:%d", localPort),
			fmt.Sprintf("127.0.0.1:%d", sshPort),
			"http --custom-domain test.example.com",
		)
		framework.ExpectNoError(tc.Start())
		defer tc.Close()

		time.Sleep(time.Second)
		framework.NewRequestExpect(f).Port(vhostPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("test.example.com")
			}).
			Ensure()
	})

	ginkgo.It("https", func() {
		sshPort := f.AllocPort()
		vhostPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		vhostHTTPSPort = %d
		sshTunnelGateway.bindPort = %d
		`, vhostPort, sshPort)

		f.RunProcesses([]string{serverConf}, nil)

		localPort := f.AllocPort()
		testDomain := "test.example.com"
		tc := ssh.NewTunnelClient(
			fmt.Sprintf("127.0.0.1:%d", localPort),
			fmt.Sprintf("127.0.0.1:%d", sshPort),
			fmt.Sprintf("https --custom-domain %s", testDomain),
		)
		framework.ExpectNoError(tc.Start())
		defer tc.Close()

		tlsConfig, err := transport.NewServerTLSConfig("", "", "")
		framework.ExpectNoError(err)
		localServer := httpserver.New(
			httpserver.WithBindPort(localPort),
			httpserver.WithTLSConfig(tlsConfig),
			httpserver.WithResponse([]byte("test")),
		)
		f.RunServer("", localServer)

		time.Sleep(time.Second)
		framework.NewRequestExpect(f).
			Port(vhostPort).
			RequestModify(func(r *request.Request) {
				r.HTTPS().HTTPHost(testDomain).TLSConfig(&tls.Config{
					ServerName:         testDomain,
					InsecureSkipVerify: true,
				})
			}).
			ExpectResp([]byte("test")).
			Ensure()
	})

	ginkgo.It("tcpmux", func() {
		sshPort := f.AllocPort()
		tcpmuxPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		tcpmuxHTTPConnectPort = %d
		sshTunnelGateway.bindPort = %d
		`, tcpmuxPort, sshPort)

		f.RunProcesses([]string{serverConf}, nil)

		localPort := f.AllocPort()
		testDomain := "test.example.com"
		tc := ssh.NewTunnelClient(
			fmt.Sprintf("127.0.0.1:%d", localPort),
			fmt.Sprintf("127.0.0.1:%d", sshPort),
			fmt.Sprintf("tcpmux --mux=httpconnect --custom-domain %s", testDomain),
		)
		framework.ExpectNoError(tc.Start())
		defer tc.Close()

		localServer := streamserver.New(
			streamserver.TCP,
			streamserver.WithBindPort(localPort),
			streamserver.WithRespContent([]byte("test")),
		)
		f.RunServer("", localServer)

		time.Sleep(time.Second)
		// Request without HTTP connect should get error
		framework.NewRequestExpect(f).
			Port(tcpmuxPort).
			ExpectError(true).
			Explain("request without HTTP connect expect error").
			Ensure()

		proxyURL := fmt.Sprintf("http://127.0.0.1:%d", tcpmuxPort)
		// Request with incorrect connect hostname
		framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
			r.Addr("invalid").Proxy(proxyURL)
		}).ExpectError(true).Explain("request without HTTP connect expect error").Ensure()

		// Request with correct connect hostname
		framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
			r.Addr(testDomain).Proxy(proxyURL)
		}).ExpectResp([]byte("test")).Ensure()
	})

	ginkgo.It("stcp", func() {
		sshPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		sshTunnelGateway.bindPort = %d
		`, sshPort)

		bindPort := f.AllocPort()
		visitorConf := consts.DefaultClientConfig + fmt.Sprintf(`
        [[visitors]]
		name = "stcp-test-visitor"
		type = "stcp"
		serverName = "stcp-test"
		secretKey = "abcdefg"
		bindPort = %d
		`, bindPort)

		f.RunProcesses([]string{serverConf}, []string{visitorConf})

		localPort := f.PortByName(framework.TCPEchoServerPort)
		tc := ssh.NewTunnelClient(
			fmt.Sprintf("127.0.0.1:%d", localPort),
			fmt.Sprintf("127.0.0.1:%d", sshPort),
			"stcp -n stcp-test --sk=abcdefg --allow-users=\"*\"",
		)
		framework.ExpectNoError(tc.Start())
		defer tc.Close()

		time.Sleep(time.Second)

		framework.NewRequestExpect(f).
			Port(bindPort).
			Ensure()
	})
})

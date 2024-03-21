package basic

import (
	"fmt"
	"net"
	"strconv"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/port"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

var _ = ginkgo.Describe("[Feature: Server Manager]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("Ports Whitelist", func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		serverConf += `
		allowPorts = [
		  { start = 10000, end = 11000 },
		  { single = 11002 },
		  { start = 12000, end = 13000 },
		]
		`

		tcpPortName := port.GenName("TCP", port.WithRangePorts(10000, 11000))
		udpPortName := port.GenName("UDP", port.WithRangePorts(12000, 13000))
		clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "tcp-allowded-in-range"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = {{ .%s }}
			`, framework.TCPEchoServerPort, tcpPortName)
		clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "tcp-port-not-allowed"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = 11001
			`, framework.TCPEchoServerPort)
		clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "tcp-port-unavailable"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = {{ .%s }}
			`, framework.TCPEchoServerPort, consts.PortServerName)
		clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "udp-allowed-in-range"
			type = "udp"
			localPort = {{ .%s }}
			remotePort = {{ .%s }}
			`, framework.UDPEchoServerPort, udpPortName)
		clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "udp-port-not-allowed"
			type = "udp"
			localPort = {{ .%s }}
			remotePort = 11003
			`, framework.UDPEchoServerPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		// TCP
		// Allowed in range
		framework.NewRequestExpect(f).PortName(tcpPortName).Ensure()

		// Not Allowed
		framework.NewRequestExpect(f).Port(11001).ExpectError(true).Ensure()

		// Unavailable, already bind by frps
		framework.NewRequestExpect(f).PortName(consts.PortServerName).ExpectError(true).Ensure()

		// UDP
		// Allowed in range
		framework.NewRequestExpect(f).Protocol("udp").PortName(udpPortName).Ensure()

		// Not Allowed
		framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
			r.UDP().Port(11003)
		}).ExpectError(true).Ensure()
	})

	ginkgo.It("Alloc Random Port", func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		adminPort := f.AllocPort()
		clientConf += fmt.Sprintf(`
		webServer.port = %d

		[[proxies]]
		name = "tcp"
		type = "tcp"
		localPort = {{ .%s }}

		[[proxies]]
		name = "udp"
		type = "udp"
		localPort = {{ .%s }}
		`, adminPort, framework.TCPEchoServerPort, framework.UDPEchoServerPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		client := f.APIClientForFrpc(adminPort)

		// tcp random port
		status, err := client.GetProxyStatus("tcp")
		framework.ExpectNoError(err)

		_, portStr, err := net.SplitHostPort(status.RemoteAddr)
		framework.ExpectNoError(err)
		port, err := strconv.Atoi(portStr)
		framework.ExpectNoError(err)

		framework.NewRequestExpect(f).Port(port).Ensure()

		// udp random port
		status, err = client.GetProxyStatus("udp")
		framework.ExpectNoError(err)

		_, portStr, err = net.SplitHostPort(status.RemoteAddr)
		framework.ExpectNoError(err)
		port, err = strconv.Atoi(portStr)
		framework.ExpectNoError(err)

		framework.NewRequestExpect(f).Protocol("udp").Port(port).Ensure()
	})

	ginkgo.It("Port Reuse", func() {
		serverConf := consts.DefaultServerConfig
		// Use same port as PortServer
		serverConf += fmt.Sprintf(`
		vhostHTTPPort = {{ .%s }}
		`, consts.PortServerName)

		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[[proxies]]
		name = "http"
		type = "http"
		localPort = {{ .%s }}
		customDomains = ["example.com"]
		`, framework.HTTPSimpleServerPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
			r.HTTP().HTTPHost("example.com")
		}).PortName(consts.PortServerName).Ensure()
	})

	ginkgo.It("healthz", func() {
		serverConf := consts.DefaultServerConfig
		dashboardPort := f.AllocPort()

		// Use same port as PortServer
		serverConf += fmt.Sprintf(`
		vhostHTTPPort = {{ .%s }}
		webServer.addr = "0.0.0.0"
		webServer.port = %d
		webServer.user = "admin"
		webServer.password = "admin"
		`, consts.PortServerName, dashboardPort)

		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[[proxies]]
		name = "http"
		type = "http"
		localPort = {{ .%s }}
		customDomains = ["example.com"]
		`, framework.HTTPSimpleServerPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
			r.HTTP().HTTPPath("/healthz")
		}).Port(dashboardPort).ExpectResp([]byte("")).Ensure()

		framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
			r.HTTP().HTTPPath("/")
		}).Port(dashboardPort).
			Ensure(framework.ExpectResponseCode(401))
	})
})

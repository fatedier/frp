package basic

import (
	"fmt"
	"net"
	"strconv"

	"github.com/onsi/ginkgo"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/port"
	"github.com/fatedier/frp/test/e2e/pkg/request"
	clientsdk "github.com/fatedier/frp/test/e2e/pkg/sdk/client"
)

var _ = ginkgo.Describe("[Feature: Server Manager]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("Ports Whitelist", func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		serverConf += `
			allow_ports = 20000-25000,25002,30000-50000
		`

		tcpPortName := port.GenName("TCP", port.WithRangePorts(20000, 25000))
		udpPortName := port.GenName("UDP", port.WithRangePorts(30000, 50000))
		clientConf += fmt.Sprintf(`
			[tcp-allowded-in-range]
			type = tcp
			local_port = {{ .%s }}
			remote_port = {{ .%s }}
			`, framework.TCPEchoServerPort, tcpPortName)
		clientConf += fmt.Sprintf(`
			[tcp-port-not-allowed]
			type = tcp
			local_port = {{ .%s }}
			remote_port = 20001
			`, framework.TCPEchoServerPort)
		clientConf += fmt.Sprintf(`
			[tcp-port-unavailable]
			type = tcp
			local_port = {{ .%s }}
			remote_port = {{ .%s }}
			`, framework.TCPEchoServerPort, consts.PortServerName)
		clientConf += fmt.Sprintf(`
			[udp-allowed-in-range]
			type = udp
			local_port = {{ .%s }}
			remote_port = {{ .%s }}
			`, framework.UDPEchoServerPort, udpPortName)
		clientConf += fmt.Sprintf(`
			[udp-port-not-allowed]
			type = udp
			local_port = {{ .%s }}
			remote_port = 20003
			`, framework.UDPEchoServerPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		// TCP
		// Allowed in range
		framework.NewRequestExpect(f).PortName(tcpPortName).Ensure()

		// Not Allowed
		framework.NewRequestExpect(f).Port(25003).ExpectError(true).Ensure()

		// Unavailable, already bind by frps
		framework.NewRequestExpect(f).PortName(consts.PortServerName).ExpectError(true).Ensure()

		// UDP
		// Allowed in range
		framework.NewRequestExpect(f).Protocol("udp").PortName(udpPortName).Ensure()

		// Not Allowed
		framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
			r.UDP().Port(25003)
		}).ExpectError(true).Ensure()
	})

	ginkgo.It("Alloc Random Port", func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		adminPort := f.AllocPort()
		clientConf += fmt.Sprintf(`
		admin_port = %d

		[tcp]
		type = tcp
		local_port = {{ .%s }}

		[udp]
		type = udp
		local_port = {{ .%s }}
		`, adminPort, framework.TCPEchoServerPort, framework.UDPEchoServerPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		client := clientsdk.New("127.0.0.1", adminPort)

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
		vhost_http_port = {{ .%s }}
		`, consts.PortServerName)

		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[http]
		type = http
		local_port = {{ .%s }}
		custom_domains = example.com
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
		vhost_http_port = {{ .%s }}
		dashboard_addr = 0.0.0.0
		dashboard_port = %d
		dashboard_user = admin
		dashboard_pwd = admin
		`, consts.PortServerName, dashboardPort)

		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[http]
		type = http
		local_port = {{ .%s }}
		custom_domains = example.com
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

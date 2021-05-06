package basic

import (
	"fmt"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/port"
	"github.com/fatedier/frp/test/e2e/pkg/request"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("[Feature: Server Manager]", func() {
	f := framework.NewDefaultFramework()

	It("Ports Whitelist", func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		serverConf += `
			allow_ports = 10000-20000,20002,30000-50000
		`

		tcpPortName := port.GenName("TCP", port.WithRangePorts(10000, 20000))
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
		framework.NewRequestExpect(f).RequestModify(framework.SetRequestPort(20001)).ExpectError(true).Ensure()

		// Unavailable, already bind by frps
		framework.NewRequestExpect(f).PortName(consts.PortServerName).ExpectError(true).Ensure()

		// UDP
		// Allowed in range
		framework.NewRequestExpect(f).RequestModify(framework.SetRequestProtocol("udp")).PortName(udpPortName).Ensure()

		// Not Allowed
		framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
			r.UDP().Port(20003)
		}).ExpectError(true).Ensure()
	})
})

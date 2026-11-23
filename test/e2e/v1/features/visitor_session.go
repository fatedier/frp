package features

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/port"
)

var _ = ginkgo.Describe("[Feature: VisitorSession]", func() {
	f := framework.NewDefaultFramework()

	// Two stcp proxies and two visitors in one frpc, each visitor on its own session to frps.
	run := func(serverExtra, proxyClientExtra, visitorClientExtra string) (string, string) {
		serverConf := consts.DefaultServerConfig + serverExtra
		bindA, bindB := port.GenName("VisitorSessionA"), port.GenName("VisitorSessionB")
		clientServerConf := consts.DefaultClientConfig + `
		user = "user1"
		` + proxyClientExtra + fmt.Sprintf(`
		[[proxies]]
		name = "stcp-a"
		type = "stcp"
		secretKey = "abc"
		localPort = {{ .%s }}

		[[proxies]]
		name = "stcp-b"
		type = "stcp"
		secretKey = "abc"
		localPort = {{ .%s }}
		`, framework.TCPEchoServerPort, framework.TCPEchoServerPort)
		clientVisitorConf := consts.DefaultClientConfig + `
		user = "user1"
		transport.visitorSession = "dedicated"
		` + visitorClientExtra + fmt.Sprintf(`
		[[visitors]]
		name = "visitor-a"
		type = "stcp"
		serverName = "stcp-a"
		secretKey = "abc"
		bindPort = {{ .%s }}

		[[visitors]]
		name = "visitor-b"
		type = "stcp"
		serverName = "stcp-b"
		secretKey = "abc"
		bindPort = {{ .%s }}
		`, bindA, bindB)

		f.RunProcesses(serverConf, []string{clientServerConf, clientVisitorConf})
		return bindA, bindB
	}

	ginkgo.It("dedicated sessions over tcp", func() {
		a, b := run("", "", "")
		framework.NewRequestExpect(f).PortName(a).Ensure()
		framework.NewRequestExpect(f).PortName(b).Ensure()
	})

	ginkgo.It("dedicated sessions over websocket", func() {
		ws := "transport.protocol = \"websocket\"\n"
		a, b := run("", ws, ws)
		framework.NewRequestExpect(f).PortName(a).Ensure()
		framework.NewRequestExpect(f).PortName(b).Ensure()
	})

	ginkgo.It("dedicated sessions with tcpMux disabled", func() {
		a, b := run("transport.tcpMux = false\n", "transport.tcpMux = false\n", "transport.tcpMux = false\n")
		framework.NewRequestExpect(f).PortName(a).Ensure()
		framework.NewRequestExpect(f).PortName(b).Ensure()
	})

	ginkgo.It("dedicated sessions over quic while login stays on tcp", func() {
		quic := fmt.Sprintf("quicBindPort = {{ .%s }}\n", consts.PortServerName)
		a, b := run(quic, "", fmt.Sprintf("transport.visitorSessionQuicPort = {{ .%s }}\n", consts.PortServerName))
		framework.NewRequestExpect(f).PortName(a).Ensure()
		framework.NewRequestExpect(f).PortName(b).Ensure()
	})
})

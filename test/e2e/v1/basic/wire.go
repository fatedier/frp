package basic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/port"
)

var _ = ginkgo.Describe("[Feature: WireProtocol]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("v1 tcp and udp proxies", func() {
		serverConf := consts.DefaultServerConfig
		tcpPortName := port.GenName("WireV1TCP")
		udpPortName := port.GenName("WireV1UDP")
		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		transport.wireProtocol = "v1"

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
		`, framework.TCPEchoServerPort, tcpPortName, framework.UDPEchoServerPort, udpPortName)

		f.RunProcesses(serverConf, []string{clientConf})

		framework.NewRequestExpect(f).PortName(tcpPortName).Ensure()
		framework.NewRequestExpect(f).Protocol("udp").PortName(udpPortName).Ensure()
	})

	ginkgo.It("v2 tcp and udp proxies", func() {
		serverConf := consts.DefaultServerConfig
		tcpPortName := port.GenName("WireV2TCP")
		udpPortName := port.GenName("WireV2UDP")
		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		transport.wireProtocol = "v2"

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
		`, framework.TCPEchoServerPort, tcpPortName, framework.UDPEchoServerPort, udpPortName)

		f.RunProcesses(serverConf, []string{clientConf})

		framework.NewRequestExpect(f).PortName(tcpPortName).Ensure()
		framework.NewRequestExpect(f).Protocol("udp").PortName(udpPortName).Ensure()
	})

	ginkgo.It("v2 stcp visitor", func() {
		serverConf := consts.DefaultServerConfig
		bindPortName := port.GenName("WireV2STCP")
		clientServerConf := consts.DefaultClientConfig + fmt.Sprintf(`
		user = "user1"
		transport.wireProtocol = "v2"

		[[proxies]]
		name = "stcp"
		type = "stcp"
		secretKey = "abc"
		localPort = {{ .%s }}
		`, framework.TCPEchoServerPort)
		clientVisitorConf := consts.DefaultClientConfig + fmt.Sprintf(`
		user = "user1"
		transport.wireProtocol = "v2"

		[[visitors]]
		name = "stcp-visitor"
		type = "stcp"
		serverName = "stcp"
		secretKey = "abc"
		bindPort = {{ .%s }}
		`, bindPortName)

		f.RunProcesses(serverConf, []string{clientServerConf, clientVisitorConf})

		framework.NewRequestExpect(f).PortName(bindPortName).Ensure()
	})

	ginkgo.It("reports client wire protocol", func() {
		webPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		webServer.port = %d
		`, webPort)

		v1PortName := port.GenName("WireReportV1")
		v1ClientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		clientID = "wire-v1"
		transport.wireProtocol = "v1"

		[[proxies]]
		name = "v1"
		type = "tcp"
		localPort = {{ .%s }}
		remotePort = {{ .%s }}
		`, framework.TCPEchoServerPort, v1PortName)

		v2PortName := port.GenName("WireReportV2")
		v2ClientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		clientID = "wire-v2"
		transport.wireProtocol = "v2"

		[[proxies]]
		name = "v2"
		type = "tcp"
		localPort = {{ .%s }}
		remotePort = {{ .%s }}
		`, framework.TCPEchoServerPort, v2PortName)

		f.RunProcesses(serverConf, []string{v1ClientConf, v2ClientConf})

		framework.NewRequestExpect(f).PortName(v1PortName).Ensure()
		framework.NewRequestExpect(f).PortName(v2PortName).Ensure()
		expectClientWireProtocol(webPort, "wire-v1", "v1")
		expectClientWireProtocol(webPort, "wire-v2", "v2")
	})
})

type wireClientInfo struct {
	ClientID     string `json:"clientID"`
	WireProtocol string `json:"wireProtocol"`
}

func expectClientWireProtocol(webPort int, clientID string, wireProtocol string) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/clients", webPort))
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	framework.ExpectEqual(resp.StatusCode, 200)

	content, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)

	var clients []wireClientInfo
	framework.ExpectNoError(json.Unmarshal(content, &clients))
	for _, client := range clients {
		if client.ClientID == clientID {
			framework.ExpectEqual(client.WireProtocol, wireProtocol)
			return
		}
	}
	framework.Failf("client %q not found in /api/clients response: %s", clientID, string(content))
}

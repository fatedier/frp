package basic

import (
	"strconv"
	"strings"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

const (
	ConfigValidStr = "syntax is ok"
)

var _ = ginkgo.Describe("[Feature: Cmd]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.Describe("Verify", func() {
		ginkgo.It("frps valid", func() {
			path := f.GenerateConfigFile(`
			bindAddr = "0.0.0.0"
			bindPort = 7000
			`)
			_, output, err := f.RunFrps("verify", "-c", path)
			framework.ExpectNoError(err)
			framework.ExpectTrue(strings.Contains(output, ConfigValidStr), "output: %s", output)
		})
		ginkgo.It("frps invalid", func() {
			path := f.GenerateConfigFile(`
			bindAddr = "0.0.0.0"
			bindPort = 70000
			`)
			_, output, err := f.RunFrps("verify", "-c", path)
			framework.ExpectNoError(err)
			framework.ExpectTrue(!strings.Contains(output, ConfigValidStr), "output: %s", output)
		})
		ginkgo.It("frpc valid", func() {
			path := f.GenerateConfigFile(`
			serverAddr = "0.0.0.0"
			serverPort = 7000
			`)
			_, output, err := f.RunFrpc("verify", "-c", path)
			framework.ExpectNoError(err)
			framework.ExpectTrue(strings.Contains(output, ConfigValidStr), "output: %s", output)
		})
		ginkgo.It("frpc invalid", func() {
			path := f.GenerateConfigFile(`
			serverAddr = "0.0.0.0"
			serverPort = 7000
			transport.protocol = "invalid"
			`)
			_, output, err := f.RunFrpc("verify", "-c", path)
			framework.ExpectNoError(err)
			framework.ExpectTrue(!strings.Contains(output, ConfigValidStr), "output: %s", output)
		})
	})

	ginkgo.Describe("Single proxy", func() {
		ginkgo.It("TCP", func() {
			serverPort := f.AllocPort()
			_, _, err := f.RunFrps("-t", "123", "-p", strconv.Itoa(serverPort))
			framework.ExpectNoError(err)

			localPort := f.PortByName(framework.TCPEchoServerPort)
			remotePort := f.AllocPort()
			_, _, err = f.RunFrpc("tcp", "-s", "127.0.0.1", "-P", strconv.Itoa(serverPort), "-t", "123", "-u", "test",
				"-l", strconv.Itoa(localPort), "-r", strconv.Itoa(remotePort), "-n", "tcp_test")
			framework.ExpectNoError(err)

			framework.NewRequestExpect(f).Port(remotePort).Ensure()
		})

		ginkgo.It("UDP", func() {
			serverPort := f.AllocPort()
			_, _, err := f.RunFrps("-t", "123", "-p", strconv.Itoa(serverPort))
			framework.ExpectNoError(err)

			localPort := f.PortByName(framework.UDPEchoServerPort)
			remotePort := f.AllocPort()
			_, _, err = f.RunFrpc("udp", "-s", "127.0.0.1", "-P", strconv.Itoa(serverPort), "-t", "123", "-u", "test",
				"-l", strconv.Itoa(localPort), "-r", strconv.Itoa(remotePort), "-n", "udp_test")
			framework.ExpectNoError(err)

			framework.NewRequestExpect(f).Protocol("udp").
				Port(remotePort).Ensure()
		})

		ginkgo.It("HTTP", func() {
			serverPort := f.AllocPort()
			vhostHTTPPort := f.AllocPort()
			_, _, err := f.RunFrps("-t", "123", "-p", strconv.Itoa(serverPort), "--vhost-http-port", strconv.Itoa(vhostHTTPPort))
			framework.ExpectNoError(err)

			_, _, err = f.RunFrpc("http", "-s", "127.0.0.1", "-P", strconv.Itoa(serverPort), "-t", "123", "-u", "test",
				"-n", "udp_test", "-l", strconv.Itoa(f.PortByName(framework.HTTPSimpleServerPort)),
				"--custom-domain", "test.example.com")
			framework.ExpectNoError(err)

			framework.NewRequestExpect(f).Port(vhostHTTPPort).
				RequestModify(func(r *request.Request) {
					r.HTTP().HTTPHost("test.example.com")
				}).
				Ensure()
		})
	})
})

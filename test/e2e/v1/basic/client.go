package basic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

var _ = ginkgo.Describe("[Feature: ClientManage]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("Update && Reload API", func() {
		serverConf := consts.DefaultServerConfig

		adminPort := f.AllocPort()

		p1Port := f.AllocPort()
		p2Port := f.AllocPort()
		p3Port := f.AllocPort()

		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		webServer.port = %d

		[[proxies]]
		name = "p1"
		type = "tcp"
		localPort = {{ .%s }}
		remotePort = %d

		[[proxies]]
		name = "p2"
		type = "tcp"
		localPort = {{ .%s }}
		remotePort = %d

		[[proxies]]
		name = "p3"
		type = "tcp"
		localPort = {{ .%s }}
		remotePort = %d
		`, adminPort,
			framework.TCPEchoServerPort, p1Port,
			framework.TCPEchoServerPort, p2Port,
			framework.TCPEchoServerPort, p3Port)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).Port(p1Port).Ensure()
		framework.NewRequestExpect(f).Port(p2Port).Ensure()
		framework.NewRequestExpect(f).Port(p3Port).Ensure()

		client := f.APIClientForFrpc(adminPort)
		conf, err := client.GetConfig(context.Background())
		framework.ExpectNoError(err)

		newP2Port := f.AllocPort()
		// change p2 port and remove p3 proxy
		newClientConf := strings.ReplaceAll(conf, strconv.Itoa(p2Port), strconv.Itoa(newP2Port))
		p3Index := strings.LastIndex(newClientConf, "[[proxies]]")
		if p3Index >= 0 {
			newClientConf = newClientConf[:p3Index]
		}

		err = client.UpdateConfig(context.Background(), newClientConf)
		framework.ExpectNoError(err)

		err = client.Reload(context.Background(), true)
		framework.ExpectNoError(err)
		time.Sleep(time.Second)

		framework.NewRequestExpect(f).Port(p1Port).Explain("p1 port").Ensure()
		framework.NewRequestExpect(f).Port(p2Port).Explain("original p2 port").ExpectError(true).Ensure()
		framework.NewRequestExpect(f).Port(newP2Port).Explain("new p2 port").Ensure()
		framework.NewRequestExpect(f).Port(p3Port).Explain("p3 port").ExpectError(true).Ensure()
	})

	ginkgo.It("healthz", func() {
		serverConf := consts.DefaultServerConfig

		dashboardPort := f.AllocPort()
		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
        webServer.addr = "0.0.0.0"
		webServer.port = %d
		webServer.user = "admin"
		webServer.password = "admin"
		`, dashboardPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
			r.HTTP().HTTPPath("/healthz")
		}).Port(dashboardPort).ExpectResp([]byte("")).Ensure()

		framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
			r.HTTP().HTTPPath("/")
		}).Port(dashboardPort).
			Ensure(framework.ExpectResponseCode(401))
	})

	ginkgo.It("stop", func() {
		serverConf := consts.DefaultServerConfig

		adminPort := f.AllocPort()
		testPort := f.AllocPort()
		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		webServer.port = %d

		[[proxies]]
		name = "test"
		type = "tcp"
		localPort = {{ .%s }}
		remotePort = %d
		`, adminPort, framework.TCPEchoServerPort, testPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).Port(testPort).Ensure()

		client := f.APIClientForFrpc(adminPort)
		err := client.Stop(context.Background())
		framework.ExpectNoError(err)

		time.Sleep(3 * time.Second)

		// frpc stopped so the port is not listened, expect error
		framework.NewRequestExpect(f).Port(testPort).ExpectError(true).Ensure()
	})
})

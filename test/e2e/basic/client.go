package basic

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/request"
	clientsdk "github.com/fatedier/frp/test/e2e/pkg/sdk/client"
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
		admin_port = %d

		[p1]
		type = tcp
		local_port = {{ .%s }}
		remote_port = %d

		[p2]
		type = tcp
		local_port = {{ .%s }}
		remote_port = %d

		[p3]
		type = tcp
		local_port = {{ .%s }}
		remote_port = %d
		`, adminPort,
			framework.TCPEchoServerPort, p1Port,
			framework.TCPEchoServerPort, p2Port,
			framework.TCPEchoServerPort, p3Port)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).Port(p1Port).Ensure()
		framework.NewRequestExpect(f).Port(p2Port).Ensure()
		framework.NewRequestExpect(f).Port(p3Port).Ensure()

		client := clientsdk.New("127.0.0.1", adminPort)
		conf, err := client.GetConfig()
		framework.ExpectNoError(err)

		newP2Port := f.AllocPort()
		// change p2 port and remove p3 proxy
		newClientConf := strings.ReplaceAll(conf, strconv.Itoa(p2Port), strconv.Itoa(newP2Port))
		p3Index := strings.Index(newClientConf, "[p3]")
		if p3Index >= 0 {
			newClientConf = newClientConf[:p3Index]
		}

		err = client.UpdateConfig(newClientConf)
		framework.ExpectNoError(err)

		err = client.Reload()
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
		admin_addr = 0.0.0.0
		admin_port = %d
		admin_user = admin
		admin_pwd = admin
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
})

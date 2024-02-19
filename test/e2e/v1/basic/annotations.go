package basic

import (
	"fmt"
	"io"
	"net/http"

	"github.com/onsi/ginkgo/v2"
	"github.com/tidwall/gjson"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
)

var _ = ginkgo.Describe("[Feature: Annotations]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("Set Proxy Annotations", func() {
		webPort := f.AllocPort()

		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		webServer.port = %d
		`, webPort)

		p1Port := f.AllocPort()

		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
		[[proxies]]
		name = "p1"
		type = "tcp"
		localPort = {{ .%s }}
		remotePort = %d
		[proxies.annotations]
		"frp.e2e.test/foo" = "value1"
		"frp.e2e.test/bar" = "value2"
		`, framework.TCPEchoServerPort, p1Port)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).Port(p1Port).Ensure()

		// check annotations in frps
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/proxy/tcp/%s", webPort, "p1"))
		framework.ExpectNoError(err)
		framework.ExpectEqual(resp.StatusCode, 200)
		defer resp.Body.Close()
		content, err := io.ReadAll(resp.Body)
		framework.ExpectNoError(err)

		annotations := gjson.Get(string(content), "conf.annotations").Map()
		framework.ExpectEqual("value1", annotations["frp.e2e.test/foo"].String())
		framework.ExpectEqual("value2", annotations["frp.e2e.test/bar"].String())
	})
})

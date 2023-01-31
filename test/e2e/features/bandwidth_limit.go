package features

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/streamserver"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

var _ = ginkgo.Describe("[Feature: Bandwidth Limit]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("Proxy Bandwidth Limit", func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		localPort := f.AllocPort()
		localServer := streamserver.New(streamserver.TCP, streamserver.WithBindPort(localPort))
		f.RunServer("", localServer)

		remotePort := f.AllocPort()
		clientConf += fmt.Sprintf(`
			[tcp]
			type = tcp
			local_port = %d
			remote_port = %d
			bandwidth_limit = 10KB
			`, localPort, remotePort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		content := strings.Repeat("a", 50*1024) // 5KB
		start := time.Now()
		framework.NewRequestExpect(f).Port(remotePort).RequestModify(func(r *request.Request) {
			r.Body([]byte(content)).Timeout(30 * time.Second)
		}).ExpectResp([]byte(content)).Ensure()
		duration := time.Since(start)

		framework.ExpectTrue(duration.Seconds() > 7, "100Kb with 10KB limit, want > 7 seconds, but got %d seconds", duration.Seconds())
	})
})

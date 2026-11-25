package features

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/fatedier/frp/server/http/model"
	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

func metricsServerConfig(port int, pprof bool) string {
	return fmt.Sprintf(`
metricsServer.port = %d
metricsServer.user = "monitor"
metricsServer.password = "secret"
metricsServer.pprofEnable = %t
`, port, pprof)
}

func metricsRequest(port int, path string, authenticated bool) *request.Request {
	r := framework.NewRequest().HTTP().Port(port).HTTPPath(path)
	if authenticated {
		r.HTTPAuth("monitor", "secret")
	}
	return r
}

var _ = ginkgo.Describe("[Feature: Metrics Server]", func() {
	f := framework.NewDefaultFramework()

	// A single connected client must produce exactly one event, including when
	// both listeners expose /metrics. Checking names alone would also pass with
	// registered collectors that never receive business events.
	expectBusinessMetrics := func(port int) {
		gomega.Eventually(func() bool {
			resp, err := metricsRequest(port, "/metrics", true).Do()
			return err == nil && resp.Code == http.StatusOK &&
				strings.Contains(string(resp.Content), "\nfrp_server_client_counts 1\n")
		}, 5*time.Second, 50*time.Millisecond).Should(gomega.BeTrue())
	}

	// frpc serves Go runtime metrics only; no process, business or handler
	// instrumentation metrics may appear there, on any platform.
	expectRuntimeMetrics := func(port int) {
		framework.NewRequestExpect(f).Request(metricsRequest(port, "/metrics", false)).
			Ensure(framework.ExpectResponseCode(http.StatusUnauthorized))
		framework.NewRequestExpect(f).Request(metricsRequest(port, "/metrics", true)).Ensure(func(resp *request.Response) bool {
			body := string(resp.Content)
			return resp.Code == http.StatusOK &&
				strings.Contains(body, "go_goroutines") &&
				!strings.Contains(body, "process_") &&
				!strings.Contains(body, "frp_") &&
				!strings.Contains(body, "promhttp_")
		})
	}

	ginkgo.It("serves anonymous health alongside both web servers", func() {
		serverMetrics, clientMetrics := f.AllocPort(), f.AllocPort()
		serverWeb, clientWeb := f.AllocPort(), f.AllocPort()
		webConfig := func(port int) string {
			return fmt.Sprintf(`
webServer.port = %d
webServer.user = "admin"
webServer.password = "admin-secret"
`, port)
		}
		serverConf := consts.DefaultServerConfig + metricsServerConfig(serverMetrics, false) + webConfig(serverWeb)
		clientConf := consts.DefaultClientConfig + metricsServerConfig(clientMetrics, false) + webConfig(clientWeb)
		f.RunProcesses(serverConf, []string{clientConf})
		for _, port := range []int{serverMetrics, clientMetrics, serverWeb, clientWeb} {
			framework.NewRequestExpect(f).Request(metricsRequest(port, "/healthz", false)).ExpectResp([]byte("")).
				Ensure(framework.ExpectResponseCode(http.StatusOK), func(resp *request.Response) bool { return len(resp.Content) == 0 })
		}
		for _, port := range []int{serverMetrics, clientMetrics} {
			for _, path := range []string{"/", "/api/serverinfo", "/api/config", "/static/app.js"} {
				framework.NewRequestExpect(f).Request(metricsRequest(port, path, true)).Ensure(framework.ExpectResponseCode(http.StatusNotFound))
			}
		}
	})

	ginkgo.It("authenticates business metrics without enablePrometheus", func() {
		port := f.AllocPort()
		serverConf := consts.DefaultServerConfig + metricsServerConfig(port, false) + fmt.Sprintf("\nwebServer.port = %d\n", f.AllocPort())
		f.RunProcesses(serverConf, []string{consts.DefaultClientConfig})
		framework.NewRequestExpect(f).Request(metricsRequest(port, "/metrics", false)).Ensure(framework.ExpectResponseCode(http.StatusUnauthorized))
		expectBusinessMetrics(port)
	})

	ginkgo.It("requires opt-in and authentication for pprof and excludes cmdline", func() {
		serverPort, clientPort := f.AllocPort(), f.AllocPort()
		f.RunProcesses(consts.DefaultServerConfig+metricsServerConfig(serverPort, true),
			[]string{consts.DefaultClientConfig + metricsServerConfig(clientPort, false)})
		for _, path := range []string{"/debug/pprof/", "/debug/pprof/goroutine"} {
			framework.NewRequestExpect(f).Request(metricsRequest(serverPort, path, false)).Ensure(framework.ExpectResponseCode(http.StatusUnauthorized))
			framework.NewRequestExpect(f).Request(metricsRequest(serverPort, path, true)).Ensure(framework.ExpectResponseCode(http.StatusOK))
			framework.NewRequestExpect(f).Request(metricsRequest(clientPort, path, true)).Ensure(framework.ExpectResponseCode(http.StatusNotFound))
		}
		for _, authenticated := range []bool{false, true} {
			framework.NewRequestExpect(f).Request(metricsRequest(serverPort, "/debug/pprof/cmdline", authenticated)).
				Ensure(framework.ExpectResponseCode(http.StatusNotFound))
		}
	})

	ginkgo.It("serves health and business metrics without either web server", func() {
		serverPort, clientPort := f.AllocPort(), f.AllocPort()
		f.RunProcesses(consts.DefaultServerConfig+metricsServerConfig(serverPort, false),
			[]string{consts.DefaultClientConfig + metricsServerConfig(clientPort, false)})
		for _, port := range []int{serverPort, clientPort} {
			framework.NewRequestExpect(f).Request(metricsRequest(port, "/healthz", false)).Ensure(framework.ExpectResponseCode(http.StatusOK))
		}
		expectBusinessMetrics(serverPort)
		expectRuntimeMetrics(clientPort)
	})

	ginkgo.It("collects once when both metrics endpoints are enabled", func() {
		metricsPort, webPort := f.AllocPort(), f.AllocPort()
		serverConf := consts.DefaultServerConfig + metricsServerConfig(metricsPort, false) + fmt.Sprintf(`
enablePrometheus = true
webServer.port = %d
webServer.user = "monitor"
webServer.password = "secret"
`, webPort)
		f.RunProcesses(serverConf, []string{consts.DefaultClientConfig})
		expectBusinessMetrics(metricsPort)
		expectBusinessMetrics(webPort)
		framework.NewRequestExpect(f).Request(metricsRequest(webPort, "/api/serverinfo", true)).Ensure(func(resp *request.Response) bool {
			var info model.ServerInfoResp
			return resp.Code == http.StatusOK && json.Unmarshal(resp.Content, &info) == nil && info.ClientCounts == 1
		})
	})
})

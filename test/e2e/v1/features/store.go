package features

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

var _ = ginkgo.Describe("[Feature: Store]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.Describe("Store API", func() {
		ginkgo.It("create proxy via API and verify connection", func() {
			adminPort := f.AllocPort()
			remotePort := f.AllocPort()

			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			webServer.addr = "127.0.0.1"
			webServer.port = %d

			[store]
			path = "%s/store.json"
			`, adminPort, f.TempDirectory)

			f.RunProcesses([]string{serverConf}, []string{clientConf})
			time.Sleep(500 * time.Millisecond)

			proxyConfig := map[string]any{
				"name":       "test-tcp",
				"type":       "tcp",
				"localIP":    "127.0.0.1",
				"localPort":  f.PortByName(framework.TCPEchoServerPort),
				"remotePort": remotePort,
			}
			proxyBody, _ := json.Marshal(proxyConfig)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies").HTTPParams("POST", "", "/api/store/proxies", map[string]string{
					"Content-Type": "application/json",
				}).Body(proxyBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			time.Sleep(time.Second)

			framework.NewRequestExpect(f).Port(remotePort).Ensure()
		})

		ginkgo.It("update proxy via API", func() {
			adminPort := f.AllocPort()
			remotePort1 := f.AllocPort()
			remotePort2 := f.AllocPort()

			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			webServer.addr = "127.0.0.1"
			webServer.port = %d

			[store]
			path = "%s/store.json"
			`, adminPort, f.TempDirectory)

			f.RunProcesses([]string{serverConf}, []string{clientConf})
			time.Sleep(500 * time.Millisecond)

			proxyConfig := map[string]any{
				"name":       "test-tcp",
				"type":       "tcp",
				"localIP":    "127.0.0.1",
				"localPort":  f.PortByName(framework.TCPEchoServerPort),
				"remotePort": remotePort1,
			}
			proxyBody, _ := json.Marshal(proxyConfig)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies").HTTPParams("POST", "", "/api/store/proxies", map[string]string{
					"Content-Type": "application/json",
				}).Body(proxyBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			time.Sleep(time.Second)
			framework.NewRequestExpect(f).Port(remotePort1).Ensure()

			proxyConfig["remotePort"] = remotePort2
			proxyBody, _ = json.Marshal(proxyConfig)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies/test-tcp").HTTPParams("PUT", "", "/api/store/proxies/test-tcp", map[string]string{
					"Content-Type": "application/json",
				}).Body(proxyBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			time.Sleep(time.Second)
			framework.NewRequestExpect(f).Port(remotePort2).Ensure()
			framework.NewRequestExpect(f).Port(remotePort1).ExpectError(true)
		})

		ginkgo.It("delete proxy via API", func() {
			adminPort := f.AllocPort()
			remotePort := f.AllocPort()

			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			webServer.addr = "127.0.0.1"
			webServer.port = %d

			[store]
			path = "%s/store.json"
			`, adminPort, f.TempDirectory)

			f.RunProcesses([]string{serverConf}, []string{clientConf})
			time.Sleep(500 * time.Millisecond)

			proxyConfig := map[string]any{
				"name":       "test-tcp",
				"type":       "tcp",
				"localIP":    "127.0.0.1",
				"localPort":  f.PortByName(framework.TCPEchoServerPort),
				"remotePort": remotePort,
			}
			proxyBody, _ := json.Marshal(proxyConfig)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies").HTTPParams("POST", "", "/api/store/proxies", map[string]string{
					"Content-Type": "application/json",
				}).Body(proxyBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			time.Sleep(time.Second)
			framework.NewRequestExpect(f).Port(remotePort).Ensure()

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies/test-tcp").HTTPParams("DELETE", "", "/api/store/proxies/test-tcp", nil)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			time.Sleep(time.Second)
			framework.NewRequestExpect(f).Port(remotePort).ExpectError(true)
		})

		ginkgo.It("list and get proxy via API", func() {
			adminPort := f.AllocPort()
			remotePort := f.AllocPort()

			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			webServer.addr = "127.0.0.1"
			webServer.port = %d

			[store]
			path = "%s/store.json"
			`, adminPort, f.TempDirectory)

			f.RunProcesses([]string{serverConf}, []string{clientConf})
			time.Sleep(500 * time.Millisecond)

			proxyConfig := map[string]any{
				"name":       "test-tcp",
				"type":       "tcp",
				"localIP":    "127.0.0.1",
				"localPort":  f.PortByName(framework.TCPEchoServerPort),
				"remotePort": remotePort,
			}
			proxyBody, _ := json.Marshal(proxyConfig)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies").HTTPParams("POST", "", "/api/store/proxies", map[string]string{
					"Content-Type": "application/json",
				}).Body(proxyBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			time.Sleep(500 * time.Millisecond)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies")
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200 && strings.Contains(string(resp.Content), "test-tcp")
			})

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies/test-tcp")
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200 && strings.Contains(string(resp.Content), "test-tcp")
			})

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies/nonexistent")
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 404
			})
		})

		ginkgo.It("store disabled returns 404", func() {
			adminPort := f.AllocPort()

			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			webServer.addr = "127.0.0.1"
			webServer.port = %d
			`, adminPort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})
			time.Sleep(500 * time.Millisecond)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies")
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 404
			})
		})
	})
})

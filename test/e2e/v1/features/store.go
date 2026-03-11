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

			f.RunProcesses(serverConf, []string{clientConf})
			framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", adminPort), 5*time.Second))

			proxyConfig := map[string]any{
				"name": "test-tcp",
				"type": "tcp",
				"tcp": map[string]any{
					"localIP":    "127.0.0.1",
					"localPort":  f.PortByName(framework.TCPEchoServerPort),
					"remotePort": remotePort,
				},
			}
			proxyBody, _ := json.Marshal(proxyConfig)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies").HTTPParams("POST", "", "/api/store/proxies", map[string]string{
					"Content-Type": "application/json",
				}).Body(proxyBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", remotePort), 5*time.Second))

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

			f.RunProcesses(serverConf, []string{clientConf})
			framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", adminPort), 5*time.Second))

			proxyConfig := map[string]any{
				"name": "test-tcp",
				"type": "tcp",
				"tcp": map[string]any{
					"localIP":    "127.0.0.1",
					"localPort":  f.PortByName(framework.TCPEchoServerPort),
					"remotePort": remotePort1,
				},
			}
			proxyBody, _ := json.Marshal(proxyConfig)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies").HTTPParams("POST", "", "/api/store/proxies", map[string]string{
					"Content-Type": "application/json",
				}).Body(proxyBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", remotePort1), 5*time.Second))
			framework.NewRequestExpect(f).Port(remotePort1).Ensure()

			proxyConfig["tcp"].(map[string]any)["remotePort"] = remotePort2
			proxyBody, _ = json.Marshal(proxyConfig)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies/test-tcp").HTTPParams("PUT", "", "/api/store/proxies/test-tcp", map[string]string{
					"Content-Type": "application/json",
				}).Body(proxyBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", remotePort2), 5*time.Second))
			framework.ExpectNoError(framework.WaitForTCPUnreachable(fmt.Sprintf("127.0.0.1:%d", remotePort1), 100*time.Millisecond, 5*time.Second))
			framework.NewRequestExpect(f).Port(remotePort2).Ensure()
			framework.NewRequestExpect(f).Port(remotePort1).ExpectError(true).Ensure()
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

			f.RunProcesses(serverConf, []string{clientConf})
			framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", adminPort), 5*time.Second))

			proxyConfig := map[string]any{
				"name": "test-tcp",
				"type": "tcp",
				"tcp": map[string]any{
					"localIP":    "127.0.0.1",
					"localPort":  f.PortByName(framework.TCPEchoServerPort),
					"remotePort": remotePort,
				},
			}
			proxyBody, _ := json.Marshal(proxyConfig)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies").HTTPParams("POST", "", "/api/store/proxies", map[string]string{
					"Content-Type": "application/json",
				}).Body(proxyBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", remotePort), 5*time.Second))
			framework.NewRequestExpect(f).Port(remotePort).Ensure()

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies/test-tcp").HTTPParams("DELETE", "", "/api/store/proxies/test-tcp", nil)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			framework.ExpectNoError(framework.WaitForTCPUnreachable(fmt.Sprintf("127.0.0.1:%d", remotePort), 100*time.Millisecond, 5*time.Second))
			framework.NewRequestExpect(f).Port(remotePort).ExpectError(true).Ensure()
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

			f.RunProcesses(serverConf, []string{clientConf})
			framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", adminPort), 5*time.Second))

			proxyConfig := map[string]any{
				"name": "test-tcp",
				"type": "tcp",
				"tcp": map[string]any{
					"localIP":    "127.0.0.1",
					"localPort":  f.PortByName(framework.TCPEchoServerPort),
					"remotePort": remotePort,
				},
			}
			proxyBody, _ := json.Marshal(proxyConfig)

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies").HTTPParams("POST", "", "/api/store/proxies", map[string]string{
					"Content-Type": "application/json",
				}).Body(proxyBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

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

			f.RunProcesses(serverConf, []string{clientConf})
			framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", adminPort), 5*time.Second))

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies")
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 404
			})
		})

		ginkgo.It("rejects mismatched type block", func() {
			adminPort := f.AllocPort()

			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			webServer.addr = "127.0.0.1"
			webServer.port = %d

			[store]
			path = "%s/store.json"
			`, adminPort, f.TempDirectory)

			f.RunProcesses(serverConf, []string{clientConf})
			framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", adminPort), 5*time.Second))

			invalidBody, _ := json.Marshal(map[string]any{
				"name": "bad-proxy",
				"type": "tcp",
				"udp": map[string]any{
					"localPort": 1234,
				},
			})

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies").HTTPParams("POST", "", "/api/store/proxies", map[string]string{
					"Content-Type": "application/json",
				}).Body(invalidBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 400
			})
		})

		ginkgo.It("rejects path/body name mismatch on update", func() {
			adminPort := f.AllocPort()
			remotePort := f.AllocPort()

			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			webServer.addr = "127.0.0.1"
			webServer.port = %d

			[store]
			path = "%s/store.json"
			`, adminPort, f.TempDirectory)

			f.RunProcesses(serverConf, []string{clientConf})
			framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", adminPort), 5*time.Second))

			createBody, _ := json.Marshal(map[string]any{
				"name": "proxy-a",
				"type": "tcp",
				"tcp": map[string]any{
					"localIP":    "127.0.0.1",
					"localPort":  f.PortByName(framework.TCPEchoServerPort),
					"remotePort": remotePort,
				},
			})

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies").HTTPParams("POST", "", "/api/store/proxies", map[string]string{
					"Content-Type": "application/json",
				}).Body(createBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 200
			})

			updateBody, _ := json.Marshal(map[string]any{
				"name": "proxy-b",
				"type": "tcp",
				"tcp": map[string]any{
					"localIP":    "127.0.0.1",
					"localPort":  f.PortByName(framework.TCPEchoServerPort),
					"remotePort": remotePort,
				},
			})

			framework.NewRequestExpect(f).RequestModify(func(r *request.Request) {
				r.HTTP().Port(adminPort).HTTPPath("/api/store/proxies/proxy-a").HTTPParams("PUT", "", "/api/store/proxies/proxy-a", map[string]string{
					"Content-Type": "application/json",
				}).Body(updateBody)
			}).Ensure(func(resp *request.Response) bool {
				return resp.Code == 400
			})
		})
	})
})

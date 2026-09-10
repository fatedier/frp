package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/onsi/ginkgo/v2"

	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/httpserver"
	pluginpkg "github.com/fatedier/frp/test/e2e/pkg/plugin"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

var _ = ginkgo.Describe("[Feature: Server-Plugins]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.Describe("Login", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.LoginContent{}
			return &r
		}

		ginkgo.It("Auth for custom meta token", func() {
			localPort := f.AllocPort()

			clientAddressGot := false
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.LoginContent)
				if content.ClientAddress != "" {
					clientAddressGot = true
				}
				if content.Metas["token"] == "123" {
					ret.Unchange = true
				} else {
					ret.Reject = true
					ret.RejectReason = "invalid token"
				}
				return &ret
			}
			pluginServer := pluginpkg.NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[[httpPlugins]]
			name = "user-manager"
			addr = "127.0.0.1:%d"
			path = "/handler"
			ops = ["Login"]
			`, localPort)
			clientConf := consts.DefaultClientConfig

			remotePort := f.AllocPort()
			clientConf += fmt.Sprintf(`
			metadatas.token = "123"

			[[proxies]]
			name = "tcp"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = %d
			`, framework.TCPEchoServerPort, remotePort)

			remotePort2 := f.AllocPort()
			invalidTokenClientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			[[proxies]]
			name = "tcp2"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = %d
			`, framework.TCPEchoServerPort, remotePort2)

			f.RunProcesses(serverConf, []string{clientConf, invalidTokenClientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()
			framework.NewRequestExpect(f).Port(remotePort2).ExpectError(true).Ensure()

			framework.ExpectTrue(clientAddressGot)
		})
	})

	ginkgo.Describe("NewProxy", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.NewProxyContent{}
			return &r
		}

		ginkgo.It("Validate Info", func() {
			localPort := f.AllocPort()
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.NewProxyContent)
				if content.ProxyName == "tcp" {
					ret.Unchange = true
				} else {
					ret.Reject = true
				}
				return &ret
			}
			pluginServer := pluginpkg.NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[[httpPlugins]]
			name = "test"
			addr = "127.0.0.1:%d"
			path = "/handler"
			ops = ["NewProxy"]
			`, localPort)
			clientConf := consts.DefaultClientConfig

			remotePort := f.AllocPort()
			clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "tcp"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = %d
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses(serverConf, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()
		})

		ginkgo.It("Modify RemotePort", func() {
			localPort := f.AllocPort()
			remotePort := f.AllocPort()
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.NewProxyContent)
				content.RemotePort = remotePort
				ret.Content = content
				return &ret
			}
			pluginServer := pluginpkg.NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[[httpPlugins]]
			name = "test"
			addr = "127.0.0.1:%d"
			path = "/handler"
			ops = ["NewProxy"]
			`, localPort)
			clientConf := consts.DefaultClientConfig

			clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "tcp"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = 0
			`, framework.TCPEchoServerPort)

			f.RunProcesses(serverConf, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()
		})
	})

	ginkgo.Describe("CloseProxy", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.CloseProxyContent{}
			return &r
		}

		ginkgo.It("Validate Info", func() {
			localPort := f.AllocPort()
			var recordProxyName string
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.CloseProxyContent)
				recordProxyName = content.ProxyName
				return &ret
			}
			pluginServer := pluginpkg.NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[[httpPlugins]]
			name = "test"
			addr = "127.0.0.1:%d"
			path = "/handler"
			ops = ["CloseProxy"]
			`, localPort)
			clientConf := consts.DefaultClientConfig

			remotePort := f.AllocPort()
			clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "tcp"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = %d
			`, framework.TCPEchoServerPort, remotePort)

			_, clients := f.RunProcesses(serverConf, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()

			for _, c := range clients {
				_ = c.Stop()
			}

			time.Sleep(1 * time.Second)

			framework.ExpectEqual(recordProxyName, "tcp")
		})
	})

	ginkgo.Describe("Ping", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.PingContent{}
			return &r
		}

		ginkgo.It("Validate Info", func() {
			localPort := f.AllocPort()

			var record string
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.PingContent)
				record = content.PrivilegeKey
				ret.Unchange = true
				return &ret
			}
			pluginServer := pluginpkg.NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[[httpPlugins]]
			name = "test"
			addr = "127.0.0.1:%d"
			path = "/handler"
			ops = ["Ping"]
			`, localPort)

			remotePort := f.AllocPort()
			clientConf := consts.DefaultClientConfig
			clientConf += fmt.Sprintf(`
			transport.heartbeatInterval = 1
			auth.additionalScopes = ["HeartBeats"]

			[[proxies]]
			name = "tcp"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = %d
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses(serverConf, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()

			time.Sleep(3 * time.Second)
			framework.ExpectNotEqual("", record)
		})
	})

	ginkgo.Describe("NewWorkConn", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.NewWorkConnContent{}
			return &r
		}

		ginkgo.It("Validate Info", func() {
			localPort := f.AllocPort()

			var record string
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.NewWorkConnContent)
				record = content.RunID
				ret.Unchange = true
				return &ret
			}
			pluginServer := pluginpkg.NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[[httpPlugins]]
			name = "test"
			addr = "127.0.0.1:%d"
			path = "/handler"
			ops = ["NewWorkConn"]
			`, localPort)

			remotePort := f.AllocPort()
			clientConf := consts.DefaultClientConfig
			clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "tcp"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = %d
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses(serverConf, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()

			framework.ExpectNotEqual("", record)
		})
	})

	ginkgo.Describe("NewUserConn", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.NewUserConnContent{}
			return &r
		}
		ginkgo.It("Validate Info", func() {
			localPort := f.AllocPort()

			var record string
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.NewUserConnContent)
				record = content.RemoteAddr
				ret.Unchange = true
				return &ret
			}
			pluginServer := pluginpkg.NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[[httpPlugins]]
			name = "test"
			addr = "127.0.0.1:%d"
			path = "/handler"
			ops = ["NewUserConn"]
			`, localPort)

			remotePort := f.AllocPort()
			clientConf := consts.DefaultClientConfig
			clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "tcp"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = %d
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses(serverConf, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()

			framework.ExpectNotEqual("", record)
		})
	})

	ginkgo.Describe("NewHTTPRequest", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.NewHTTPRequestContent{}
			return &r
		}

		ginkgo.It("checks every routed request before backend access", func() {
			pluginPort := f.AllocPort()
			records := make(chan plugin.NewHTTPRequestContent, 2)
			handler := func(req *plugin.Request) *plugin.Response {
				content := req.Content.(*plugin.NewHTTPRequestContent)
				records <- *content
				if content.URI == "/api/blocked" {
					return &plugin.Response{Reject: true, RejectReason: "blocked"}
				}
				return &plugin.Response{Unchange: true}
			}
			pluginServer := pluginpkg.NewHTTPPluginServer(pluginPort, newFunc, handler, nil)
			f.RunServer("", pluginServer)

			backendPort := f.AllocPort()
			var backendCalls atomic.Int32
			backend := httpserver.New(
				httpserver.WithBindPort(backendPort),
				httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					backendCalls.Add(1)
					_, _ = w.Write([]byte("backend"))
				})),
			)
			f.RunServer("", backend)

			vhostHTTPPort := f.AllocPort()
			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			vhostHTTPPort = %d

			[[httpPlugins]]
			name = "request-policy"
			addr = "127.0.0.1:%d"
			path = "/handler"
			ops = ["NewHTTPRequest"]
			`, vhostHTTPPort, pluginPort)
			clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			user = "request-user"

			[[proxies]]
			name = "http-request"
			type = "http"
			localPort = %d
			customDomains = ["plugin.example.com"]
			locations = ["/api"]
			`, backendPort)
			f.RunProcesses(serverConf, []string{clientConf})

			framework.NewRequestExpect(f).Port(vhostHTTPPort).
				RequestModify(func(r *request.Request) {
					r.HTTP().HTTPHost("plugin.example.com").HTTPPath("/api/blocked?secret=value").
						HTTPHeaders(map[string]string{
							"Authorization": "Bearer private-token",
							"Cookie":        "session=private-cookie",
						})
				}).
				Ensure(framework.ExpectResponseCode(http.StatusForbidden))
			framework.ExpectEqual(int32(0), backendCalls.Load())

			blocked := <-records
			framework.ExpectEqual("request-user", blocked.User.User)
			framework.ExpectEqual("request-user.http-request", blocked.ProxyName)
			framework.ExpectNotEqual("", blocked.RemoteAddr)
			framework.ExpectEqual("plugin.example.com", blocked.Host)
			framework.ExpectEqual(http.MethodGet, blocked.Method)
			framework.ExpectEqual("/api/blocked", blocked.URI)
			framework.ExpectEqual("plugin.example.com", blocked.RouteDomain)
			framework.ExpectEqual("/api", blocked.RouteLocation)
			encoded, err := json.Marshal(blocked)
			framework.ExpectNoError(err)
			framework.ExpectEqual(false, strings.Contains(string(encoded), "secret=value"))
			framework.ExpectEqual(false, strings.Contains(strings.ToLower(string(encoded)), "authorization"))
			framework.ExpectEqual(false, strings.Contains(strings.ToLower(string(encoded)), "cookie"))

			framework.NewRequestExpect(f).Port(vhostHTTPPort).
				RequestModify(func(r *request.Request) {
					r.HTTP().HTTPHost("plugin.example.com").HTTPPath("/api/allowed")
				}).
				ExpectResp([]byte("backend")).
				Ensure()
			allowed := <-records
			framework.ExpectEqual("/api/allowed", allowed.URI)
			framework.ExpectEqual(int32(1), backendCalls.Load())
		})
	})

	ginkgo.Describe("HTTPS Protocol", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.NewUserConnContent{}
			return &r
		}
		ginkgo.It("Validate Login Info, disable tls verify", func() {
			localPort := f.AllocPort()

			var record string
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.NewUserConnContent)
				record = content.RemoteAddr
				ret.Unchange = true
				return &ret
			}
			tlsConfig, err := transport.NewServerTLSConfig("", "", "")
			framework.ExpectNoError(err)
			pluginServer := pluginpkg.NewHTTPPluginServer(localPort, newFunc, handler, tlsConfig)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[[httpPlugins]]
			name = "test"
			addr = "https://127.0.0.1:%d"
			path = "/handler"
			ops = ["NewUserConn"]
			`, localPort)

			remotePort := f.AllocPort()
			clientConf := consts.DefaultClientConfig
			clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "tcp"
			type = "tcp"
			localPort = {{ .%s }}
			remotePort = %d
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses(serverConf, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()

			framework.ExpectNotEqual("", record)
		})
	})
})

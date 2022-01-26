package plugin

import (
	"fmt"
	"time"

	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("[Feature: Server-Plugins]", func() {
	f := framework.NewDefaultFramework()

	Describe("Login", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.LoginContent{}
			return &r
		}

		It("Auth for custom meta token", func() {
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
			pluginServer := NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[plugin.user-manager]
			addr = 127.0.0.1:%d
			path = /handler
			ops = Login
			`, localPort)
			clientConf := consts.DefaultClientConfig

			remotePort := f.AllocPort()
			clientConf += fmt.Sprintf(`
			meta_token = 123

			[tcp]
			type = tcp
			local_port = {{ .%s }}
			remote_port = %d
			`, framework.TCPEchoServerPort, remotePort)

			remotePort2 := f.AllocPort()
			invalidTokenClientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			[tcp2]
			type = tcp
			local_port = {{ .%s }}
			remote_port = %d
			`, framework.TCPEchoServerPort, remotePort2)

			f.RunProcesses([]string{serverConf}, []string{clientConf, invalidTokenClientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()
			framework.NewRequestExpect(f).Port(remotePort2).ExpectError(true).Ensure()

			framework.ExpectTrue(clientAddressGot)
		})
	})

	Describe("NewProxy", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.NewProxyContent{}
			return &r
		}

		It("Validate Info", func() {
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
			pluginServer := NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[plugin.test]
			addr = 127.0.0.1:%d
			path = /handler
			ops = NewProxy
			`, localPort)
			clientConf := consts.DefaultClientConfig

			remotePort := f.AllocPort()
			clientConf += fmt.Sprintf(`
			[tcp]
			type = tcp
			local_port = {{ .%s }}
			remote_port = %d
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()
		})

		It("Mofify RemotePort", func() {
			localPort := f.AllocPort()
			remotePort := f.AllocPort()
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.NewProxyContent)
				content.RemotePort = remotePort
				ret.Content = content
				return &ret
			}
			pluginServer := NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[plugin.test]
			addr = 127.0.0.1:%d
			path = /handler
			ops = NewProxy
			`, localPort)
			clientConf := consts.DefaultClientConfig

			clientConf += fmt.Sprintf(`
			[tcp]
			type = tcp
			local_port = {{ .%s }}
			remote_port = 0
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()
		})
	})

	Describe("Ping", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.PingContent{}
			return &r
		}

		It("Validate Info", func() {
			localPort := f.AllocPort()

			var record string
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.PingContent)
				record = content.Ping.PrivilegeKey
				ret.Unchange = true
				return &ret
			}
			pluginServer := NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[plugin.test]
			addr = 127.0.0.1:%d
			path = /handler
			ops = Ping
			`, localPort)

			remotePort := f.AllocPort()
			clientConf := consts.DefaultClientConfig
			clientConf += fmt.Sprintf(`
			heartbeat_interval = 1
			authenticate_heartbeats = true

			[tcp]
			type = tcp
			local_port = {{ .%s }}
			remote_port = %d
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()

			time.Sleep(3 * time.Second)
			framework.ExpectNotEqual("", record)
		})
	})

	Describe("NewWorkConn", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.NewWorkConnContent{}
			return &r
		}

		It("Validate Info", func() {
			localPort := f.AllocPort()

			var record string
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.NewWorkConnContent)
				record = content.NewWorkConn.RunID
				ret.Unchange = true
				return &ret
			}
			pluginServer := NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[plugin.test]
			addr = 127.0.0.1:%d
			path = /handler
			ops = NewWorkConn
			`, localPort)

			remotePort := f.AllocPort()
			clientConf := consts.DefaultClientConfig
			clientConf += fmt.Sprintf(`
			[tcp]
			type = tcp
			local_port = {{ .%s }}
			remote_port = %d
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()

			framework.ExpectNotEqual("", record)
		})
	})

	Describe("NewUserConn", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.NewUserConnContent{}
			return &r
		}
		It("Validate Info", func() {
			localPort := f.AllocPort()

			var record string
			handler := func(req *plugin.Request) *plugin.Response {
				var ret plugin.Response
				content := req.Content.(*plugin.NewUserConnContent)
				record = content.RemoteAddr
				ret.Unchange = true
				return &ret
			}
			pluginServer := NewHTTPPluginServer(localPort, newFunc, handler, nil)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[plugin.test]
			addr = 127.0.0.1:%d
			path = /handler
			ops = NewUserConn
			`, localPort)

			remotePort := f.AllocPort()
			clientConf := consts.DefaultClientConfig
			clientConf += fmt.Sprintf(`
			[tcp]
			type = tcp
			local_port = {{ .%s }}
			remote_port = %d
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()

			framework.ExpectNotEqual("", record)
		})
	})

	Describe("HTTPS Protocol", func() {
		newFunc := func() *plugin.Request {
			var r plugin.Request
			r.Content = &plugin.NewUserConnContent{}
			return &r
		}
		It("Validate Login Info, disable tls verify", func() {
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
			pluginServer := NewHTTPPluginServer(localPort, newFunc, handler, tlsConfig)

			f.RunServer("", pluginServer)

			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			[plugin.test]
			addr = https://127.0.0.1:%d
			path = /handler
			ops = NewUserConn
			`, localPort)

			remotePort := f.AllocPort()
			clientConf := consts.DefaultClientConfig
			clientConf += fmt.Sprintf(`
			[tcp]
			type = tcp
			local_port = {{ .%s }}
			remote_port = %d
			`, framework.TCPEchoServerPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure()

			framework.ExpectNotEqual("", record)
		})
	})
})

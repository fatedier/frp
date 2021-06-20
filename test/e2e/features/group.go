package features

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/httpserver"
	"github.com/fatedier/frp/test/e2e/mock/server/streamserver"
	"github.com/fatedier/frp/test/e2e/pkg/request"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("[Feature: Group]", func() {
	f := framework.NewDefaultFramework()

	newHTTPServer := func(port int, respContent string) *httpserver.Server {
		return httpserver.New(
			httpserver.WithBindPort(port),
			httpserver.WithHandler(framework.SpecifiedHTTPBodyHandler([]byte(respContent))),
		)
	}

	validateFooBarResponse := func(resp *request.Response) bool {
		if string(resp.Content) == "foo" || string(resp.Content) == "bar" {
			return true
		}
		return false
	}

	doFooBarHTTPRequest := func(vhostPort int, host string) []string {
		results := []string{}
		var wait sync.WaitGroup
		var mu sync.Mutex
		expectFn := func() {
			framework.NewRequestExpect(f).Port(vhostPort).
				RequestModify(func(r *request.Request) {
					r.HTTP().HTTPHost(host)
				}).
				Ensure(validateFooBarResponse, func(resp *request.Response) bool {
					mu.Lock()
					defer mu.Unlock()
					results = append(results, string(resp.Content))
					return true
				})
		}
		for i := 0; i < 10; i++ {
			wait.Add(1)
			go func() {
				defer wait.Done()
				expectFn()
			}()
		}

		wait.Wait()
		return results
	}

	Describe("Load Balancing", func() {
		It("TCP", func() {
			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			fooPort := f.AllocPort()
			fooServer := streamserver.New(streamserver.TCP, streamserver.WithBindPort(fooPort), streamserver.WithRespContent([]byte("foo")))
			f.RunServer("", fooServer)

			barPort := f.AllocPort()
			barServer := streamserver.New(streamserver.TCP, streamserver.WithBindPort(barPort), streamserver.WithRespContent([]byte("bar")))
			f.RunServer("", barServer)

			remotePort := f.AllocPort()
			clientConf += fmt.Sprintf(`
			[foo]
			type = tcp
			local_port = %d
			remote_port = %d
			group = test
			group_key = 123

			[bar]
			type = tcp
			local_port = %d
			remote_port = %d
			group = test
			group_key = 123
			`, fooPort, remotePort, barPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			fooCount := 0
			barCount := 0
			for i := 0; i < 10; i++ {
				framework.NewRequestExpect(f).Explain("times " + strconv.Itoa(i)).Port(remotePort).Ensure(func(resp *request.Response) bool {
					switch string(resp.Content) {
					case "foo":
						fooCount++
					case "bar":
						barCount++
					default:
						return false
					}
					return true
				})
			}

			framework.ExpectTrue(fooCount > 1 && barCount > 1, "fooCount: %d, barCount: %d", fooCount, barCount)
		})
	})

	Describe("Health Check", func() {
		It("TCP", func() {
			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			fooPort := f.AllocPort()
			fooServer := streamserver.New(streamserver.TCP, streamserver.WithBindPort(fooPort), streamserver.WithRespContent([]byte("foo")))
			f.RunServer("", fooServer)

			barPort := f.AllocPort()
			barServer := streamserver.New(streamserver.TCP, streamserver.WithBindPort(barPort), streamserver.WithRespContent([]byte("bar")))
			f.RunServer("", barServer)

			remotePort := f.AllocPort()
			clientConf += fmt.Sprintf(`
			[foo]
			type = tcp
			local_port = %d
			remote_port = %d
			group = test
			group_key = 123
			health_check_type = tcp
			health_check_interval_s = 1

			[bar]
			type = tcp
			local_port = %d
			remote_port = %d
			group = test
			group_key = 123
			health_check_type = tcp
			health_check_interval_s = 1
			`, fooPort, remotePort, barPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			// check foo and bar is ok
			results := []string{}
			for i := 0; i < 10; i++ {
				framework.NewRequestExpect(f).Port(remotePort).Ensure(validateFooBarResponse, func(resp *request.Response) bool {
					results = append(results, string(resp.Content))
					return true
				})
			}
			framework.ExpectContainElements(results, []string{"foo", "bar"})

			// close bar server, check foo is ok
			barServer.Close()
			time.Sleep(2 * time.Second)
			for i := 0; i < 10; i++ {
				framework.NewRequestExpect(f).Port(remotePort).ExpectResp([]byte("foo")).Ensure()
			}

			// resume bar server, check foo and bar is ok
			f.RunServer("", barServer)
			time.Sleep(2 * time.Second)
			results = []string{}
			for i := 0; i < 10; i++ {
				framework.NewRequestExpect(f).Port(remotePort).Ensure(validateFooBarResponse, func(resp *request.Response) bool {
					results = append(results, string(resp.Content))
					return true
				})
			}
			framework.ExpectContainElements(results, []string{"foo", "bar"})
		})

		It("HTTP", func() {
			vhostPort := f.AllocPort()
			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			vhost_http_port = %d
			`, vhostPort)
			clientConf := consts.DefaultClientConfig

			fooPort := f.AllocPort()
			fooServer := newHTTPServer(fooPort, "foo")
			f.RunServer("", fooServer)

			barPort := f.AllocPort()
			barServer := newHTTPServer(barPort, "bar")
			f.RunServer("", barServer)

			clientConf += fmt.Sprintf(`
			[foo]
			type = http
			local_port = %d
			custom_domains = example.com
			group = test
			group_key = 123
			health_check_type = http
			health_check_interval_s = 1
			health_check_url = /healthz

			[bar]
			type = http
			local_port = %d
			custom_domains = example.com
			group = test
			group_key = 123
			health_check_type = http
			health_check_interval_s = 1
			health_check_url = /healthz
			`, fooPort, barPort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			// check foo and bar is ok
			results := doFooBarHTTPRequest(vhostPort, "example.com")
			framework.ExpectContainElements(results, []string{"foo", "bar"})

			// close bar server, check foo is ok
			barServer.Close()
			time.Sleep(2 * time.Second)
			results = doFooBarHTTPRequest(vhostPort, "example.com")
			framework.ExpectContainElements(results, []string{"foo"})
			framework.ExpectNotContainElements(results, []string{"bar"})

			// resume bar server, check foo and bar is ok
			f.RunServer("", barServer)
			time.Sleep(2 * time.Second)
			results = doFooBarHTTPRequest(vhostPort, "example.com")
			framework.ExpectContainElements(results, []string{"foo", "bar"})
		})
	})
})

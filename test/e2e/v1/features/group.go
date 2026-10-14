package features

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/fatedier/frp/pkg/sdk/client"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/httpserver"
	"github.com/fatedier/frp/test/e2e/mock/server/streamserver"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

func waitForProxyStatus(proxyClient *client.Client, proxyName, want string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	var lastStatus string
	var lastErr error
	for {
		status, err := proxyClient.GetProxyStatus(ctx, proxyName)
		if err == nil {
			lastErr = nil
			lastStatus = status.Status
			if status.Status == want {
				return nil
			}
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("timeout waiting for proxy %q status %q: last error: %w", proxyName, want, lastErr)
			}
			return fmt.Errorf("timeout waiting for proxy %q status %q: last status %q", proxyName, want, lastStatus)
		case <-ticker.C:
		}
	}
}

func waitForSignal(signal <-chan struct{}) error {
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-signal:
		return nil
	case <-timer.C:
		return fmt.Errorf("timeout waiting for backend health-check signal")
	}
}

// Keep this longer than client/proxy.statusCheckInterval (3s). The wrapper's
// health notification is deliberately non-blocking, so the E2E assertion must
// also cover the fallback poll plus scheduling margin while recovery is gated.
const proxyFallbackObservationWindow = 4 * time.Second

func waitForServerProxyStatus(port int, proxyName, want string, timeout time.Duration) error {
	return waitForLifecycleCondition(timeout, func() error {
		body, err := getLifecycleEndpoint(port, "/api/proxies/"+url.PathEscape(proxyName))
		if err != nil {
			return err
		}
		var status struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal([]byte(body), &status); err != nil {
			return err
		}
		if status.Status != want {
			return fmt.Errorf("frps proxy %q status %q, want %q", proxyName, status.Status, want)
		}
		return nil
	})
}

type httpHealthStage struct {
	healthy bool
	signal  chan<- struct{}
	release <-chan struct{}
}

var _ = ginkgo.Describe("[Feature: Group]", func() {
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
		for range 10 {
			wait.Go(func() {
				expectFn()
			})
		}

		wait.Wait()
		return results
	}

	ginkgo.Describe("Load Balancing", func() {
		ginkgo.It("TCP", func() {
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
			[[proxies]]
			name = "foo"
			type = "tcp"
			localPort = %d
			remotePort = %d
			loadBalancer.group = "test"
			loadBalancer.groupKey = "123"

			[[proxies]]
			name = "bar"
			type = "tcp"
			localPort = %d
			remotePort = %d
			loadBalancer.group = "test"
			loadBalancer.groupKey = "123"
			`, fooPort, remotePort, barPort, remotePort)

			f.RunProcesses(serverConf, []string{clientConf})

			fooCount := 0
			barCount := 0
			for i := range 10 {
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

		ginkgo.It("HTTPS", func() {
			vhostHTTPSPort := f.AllocPort()
			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			vhostHTTPSPort = %d
			`, vhostHTTPSPort)
			clientConf := consts.DefaultClientConfig

			tlsConfig, err := transport.NewServerTLSConfig("", "", "")
			framework.ExpectNoError(err)

			fooPort := f.AllocPort()
			fooServer := httpserver.New(
				httpserver.WithBindPort(fooPort),
				httpserver.WithHandler(framework.SpecifiedHTTPBodyHandler([]byte("foo"))),
				httpserver.WithTLSConfig(tlsConfig),
			)
			f.RunServer("", fooServer)

			barPort := f.AllocPort()
			barServer := httpserver.New(
				httpserver.WithBindPort(barPort),
				httpserver.WithHandler(framework.SpecifiedHTTPBodyHandler([]byte("bar"))),
				httpserver.WithTLSConfig(tlsConfig),
			)
			f.RunServer("", barServer)

			clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "foo"
			type = "https"
			localPort = %d
			customDomains = ["example.com"]
			loadBalancer.group = "test"
			loadBalancer.groupKey = "123"

			[[proxies]]
			name = "bar"
			type = "https"
			localPort = %d
			customDomains = ["example.com"]
			loadBalancer.group = "test"
			loadBalancer.groupKey = "123"
			`, fooPort, barPort)

			f.RunProcesses(serverConf, []string{clientConf})

			fooCount := 0
			barCount := 0
			for i := range 10 {
				framework.NewRequestExpect(f).
					Explain("times " + strconv.Itoa(i)).
					Port(vhostHTTPSPort).
					RequestModify(func(r *request.Request) {
						r.HTTPS().HTTPHost("example.com").TLSConfig(&tls.Config{
							ServerName:         "example.com",
							InsecureSkipVerify: true,
						})
					}).
					Ensure(func(resp *request.Response) bool {
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

		ginkgo.It("TCPMux httpconnect", func() {
			vhostPort := f.AllocPort()
			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			tcpmuxHTTPConnectPort = %d
			`, vhostPort)
			clientConf := consts.DefaultClientConfig

			fooPort := f.AllocPort()
			fooServer := streamserver.New(streamserver.TCP, streamserver.WithBindPort(fooPort), streamserver.WithRespContent([]byte("foo")))
			f.RunServer("", fooServer)

			barPort := f.AllocPort()
			barServer := streamserver.New(streamserver.TCP, streamserver.WithBindPort(barPort), streamserver.WithRespContent([]byte("bar")))
			f.RunServer("", barServer)

			clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "foo"
			type = "tcpmux"
			multiplexer = "httpconnect"
			localPort = %d
			customDomains = ["tcpmux-group.example.com"]
			loadBalancer.group = "test"
			loadBalancer.groupKey = "123"

			[[proxies]]
			name = "bar"
			type = "tcpmux"
			multiplexer = "httpconnect"
			localPort = %d
			customDomains = ["tcpmux-group.example.com"]
			loadBalancer.group = "test"
			loadBalancer.groupKey = "123"
			`, fooPort, barPort)

			f.RunProcesses(serverConf, []string{clientConf})

			proxyURL := fmt.Sprintf("http://127.0.0.1:%d", vhostPort)
			fooCount := 0
			barCount := 0
			for i := range 10 {
				framework.NewRequestExpect(f).
					Explain("times " + strconv.Itoa(i)).
					RequestModify(func(r *request.Request) {
						r.Addr("tcpmux-group.example.com").Proxy(proxyURL)
					}).
					Ensure(func(resp *request.Response) bool {
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

	ginkgo.Describe("Health Check", func() {
		ginkgo.It("TCP", func() {
			dashboardPort := f.AllocPort()
			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			webServer.port = %d
			`, dashboardPort)
			clientConf := consts.DefaultClientConfig

			fooPort := f.AllocPort()
			fooServer := streamserver.New(streamserver.TCP, streamserver.WithBindPort(fooPort), streamserver.WithRespContent([]byte("foo")))
			f.RunServer("", fooServer)

			barPort := f.AllocPort()
			newBarServer := func() *streamserver.Server {
				return streamserver.New(streamserver.TCP, streamserver.WithBindPort(barPort), streamserver.WithRespContent([]byte("bar")))
			}
			barServer := newBarServer()
			f.RunServer("", barServer)

			remotePort := f.AllocPort()
			adminPort := f.AllocPort()
			clientConf += fmt.Sprintf(`
			webServer.port = %d

			[[proxies]]
			name = "foo"
			type = "tcp"
			localPort = %d
			remotePort = %d
			loadBalancer.group = "test"
			loadBalancer.groupKey = "123"
			healthCheck.type = "tcp"
			healthCheck.intervalSeconds = 1

			[[proxies]]
			name = "bar"
			type = "tcp"
			localPort = %d
			remotePort = %d
			loadBalancer.group = "test"
			loadBalancer.groupKey = "123"
			healthCheck.type = "tcp"
			healthCheck.intervalSeconds = 1
			healthCheck.maxFailed = 3
			`, adminPort, fooPort, remotePort, barPort, remotePort)

			f.RunProcesses(serverConf, []string{clientConf})
			proxyClient := f.APIClientForFrpc(adminPort)
			framework.ExpectNoError(waitForProxyStatus(proxyClient, "foo", "running"))
			framework.ExpectNoError(waitForProxyStatus(proxyClient, "bar", "running"))

			// Both requests traverse frps, the load-balancing group, frpc, and the
			// corresponding local backend.
			results := []string{}
			for range 10 {
				framework.NewRequestExpect(f).Port(remotePort).Ensure(validateFooBarResponse, func(resp *request.Response) bool {
					results = append(results, string(resp.Content))
					return true
				})
			}
			framework.ExpectContainElements(results, []string{"foo", "bar"})

			// frps removes the group listener before deleting the proxy from
			// its manager. Its offline status is the removal barrier; frpc's
			// local check-failed status alone does not acknowledge that work.
			for range 2 {
				framework.ExpectNoError(barServer.Close())
				framework.ExpectNoError(waitForProxyStatus(proxyClient, "bar", "check failed"))
				framework.ExpectNoError(waitForServerProxyStatus(dashboardPort, "bar", "offline", 5*time.Second))
				for range 10 {
					framework.NewRequestExpect(f).Port(remotePort).ExpectResp([]byte("foo")).Ensure()
				}

				barServer = newBarServer()
				f.RunServer("", barServer)
				framework.ExpectNoError(waitForProxyStatus(proxyClient, "bar", "running"))
				results = []string{}
				for range 10 {
					framework.NewRequestExpect(f).Port(remotePort).Ensure(validateFooBarResponse, func(resp *request.Response) bool {
						results = append(results, string(resp.Content))
						return true
					})
				}
				framework.ExpectContainElements(results, []string{"foo", "bar"})
			}
		})

		ginkgo.It("HTTP", func() {
			vhostPort := f.AllocPort()
			dashboardPort := f.AllocPort()
			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
			vhostHTTPPort = %d
			webServer.port = %d
			`, vhostPort, dashboardPort)
			clientConf := consts.DefaultClientConfig

			fooPort := f.AllocPort()
			fooServer := newHTTPServer(fooPort, "foo")
			f.RunServer("", fooServer)

			barPort := f.AllocPort()
			var stageMu sync.RWMutex
			stage := &httpHealthStage{healthy: true, signal: make(chan struct{}, 1)}
			barServer := httpserver.New(
				httpserver.WithBindPort(barPort),
				httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					stageMu.RLock()
					currentStage := stage
					stageMu.RUnlock()

					if r.URL.Path == "/healthz" {
						if !currentStage.healthy {
							w.WriteHeader(http.StatusServiceUnavailable)
						}
						select {
						case currentStage.signal <- struct{}{}:
						default:
						}
						if currentStage.release != nil {
							select {
							case <-currentStage.release:
							case <-r.Context().Done():
								return
							}
						}
						return
					}
					_, _ = w.Write([]byte("bar"))
				})),
			)
			f.RunServer("", barServer)

			adminPort := f.AllocPort()
			clientConf += fmt.Sprintf(`
			webServer.port = %d

			[[proxies]]
			name = "foo"
			type = "http"
			localPort = %d
			customDomains = ["example.com"]
			loadBalancer.group = "test"
			loadBalancer.groupKey = "123"
			healthCheck.type = "http"
			healthCheck.intervalSeconds = 1
			healthCheck.path = "/healthz"

			[[proxies]]
			name = "bar"
			type = "http"
			localPort = %d
			customDomains = ["example.com"]
			loadBalancer.group = "test"
			loadBalancer.groupKey = "123"
			healthCheck.type = "http"
			healthCheck.intervalSeconds = 1
			healthCheck.maxFailed = 3
			healthCheck.timeoutSeconds = 10
			healthCheck.path = "/healthz"
			`, adminPort, fooPort, barPort)

			f.RunProcesses(serverConf, []string{clientConf})
			proxyClient := f.APIClientForFrpc(adminPort)
			framework.ExpectNoError(waitForProxyStatus(proxyClient, "foo", "running"))
			framework.ExpectNoError(waitForProxyStatus(proxyClient, "bar", "running"))

			// Ordinary requests traverse the real HTTP proxy path.
			var contents []string
			framework.NewRequestExpect(f).Port(vhostPort).
				RequestModify(func(r *request.Request) {
					r.HTTP().HTTPHost("example.com")
				}).
				Ensure(func(resp *request.Response) bool {
					contents = append(contents, string(resp.Content))
					return true
				})
			framework.NewRequestExpect(f).Port(vhostPort).
				RequestModify(func(r *request.Request) {
					r.HTTP().HTTPHost("example.com")
				}).
				Ensure(func(resp *request.Response) bool {
					contents = append(contents, string(resp.Content))
					return true
				})
			framework.ExpectContainElements(contents, []string{"foo", "bar"})

			// Every failed health response is gated by the test. This gives each
			// stage a request-level barrier and avoids cumulative process logs.
			runFailureRecovery := func() {
				failureSignals := make(chan struct{}, 1)
				release := make(chan struct{}, 1)
				stageMu.Lock()
				stage = &httpHealthStage{healthy: false, signal: failureSignals, release: release}
				stageMu.Unlock()

				framework.ExpectNoError(waitForSignal(failureSignals))
				release <- struct{}{}
				framework.ExpectNoError(waitForSignal(failureSignals))

				recoverySignals := make(chan struct{}, 1)
				recoveryRelease := make(chan struct{})
				stageMu.Lock()
				stage = &httpHealthStage{healthy: true, signal: recoverySignals, release: recoveryRelease}
				stageMu.Unlock()
				release <- struct{}{}

				// The next request can only start after the worker consumed the
				// second failed response. Hold this success response until the
				// assertion finishes, so premature removal cannot be hidden by
				// recovery. Consistently also gives the proxy worker time to act
				// on an erroneous failure notification without retrying it away.
				framework.ExpectNoError(waitForSignal(recoverySignals))
				gomega.Consistently(func() string {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					status, err := proxyClient.GetProxyStatus(ctx, "bar")
					if err != nil {
						return err.Error()
					}
					return status.Status
				}, proxyFallbackObservationWindow, 25*time.Millisecond).Should(gomega.Equal("running"))
				close(recoveryRelease)
			}
			runFailureRecovery()
			results := doFooBarHTTPRequest(vhostPort, "example.com")
			framework.ExpectContainElements(results, []string{"foo", "bar"})

			// Repeat the two-failure window to verify that recovery cleared the
			// counter before another failure sequence began.
			runFailureRecovery()
			results = doFooBarHTTPRequest(vhostPort, "example.com")
			framework.ExpectContainElements(results, []string{"foo", "bar"})

			// Three failed responses reach MaxFailed and remove bar from the
			// group; a healthy response then re-registers it.
			failureSignals := make(chan struct{}, 1)
			release := make(chan struct{}, 1)
			stageMu.Lock()
			stage = &httpHealthStage{healthy: false, signal: failureSignals, release: release}
			stageMu.Unlock()
			for range 3 {
				framework.ExpectNoError(waitForSignal(failureSignals))
				release <- struct{}{}
			}
			framework.ExpectNoError(waitForProxyStatus(proxyClient, "bar", "check failed"))
			// HTTPProxy.Close unregisters the group route before frps reports
			// offline. Only then assert that fresh data-plane requests use foo.
			framework.ExpectNoError(waitForServerProxyStatus(dashboardPort, "bar", "offline", 5*time.Second))
			results = doFooBarHTTPRequest(vhostPort, "example.com")
			framework.ExpectContainElements(results, []string{"foo"})
			framework.ExpectNotContainElements(results, []string{"bar"})

			recoverySignals := make(chan struct{}, 1)
			stageMu.Lock()
			stage = &httpHealthStage{healthy: true, signal: recoverySignals}
			stageMu.Unlock()
			// Release any additional failed request already waiting while the
			// removal and data-plane assertions were in progress.
			close(release)
			framework.ExpectNoError(waitForSignal(recoverySignals))
			framework.ExpectNoError(waitForProxyStatus(proxyClient, "bar", "running"))
			results = doFooBarHTTPRequest(vhostPort, "example.com")
			framework.ExpectContainElements(results, []string{"foo", "bar"})
		})
	})
})

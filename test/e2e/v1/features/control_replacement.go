// Copyright 2026 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package features

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/relay"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

var _ = ginkgo.Describe("[Feature: ControlReplacement]", func() {
	f := framework.NewDefaultFramework()

	for _, wireProtocol := range []string{"v1", "v2"} {
		for _, tcpMux := range []bool{true, false} {
			ginkgo.It(fmt.Sprintf("recovers a %s control through a half-open relay with tcpMux=%t", wireProtocol, tcpMux), func() {
				runHalfOpenControlReplacement(f, wireProtocol, tcpMux)
			})
		}
	}
})

func runHalfOpenControlReplacement(f *framework.Framework, wireProtocol string, tcpMux bool) {
	serverPort := f.AllocPort()
	dashboardPort := f.AllocPort()
	remotePort := f.AllocPort()
	heartbeatTimeout := int64(-1)
	if !tcpMux {
		heartbeatTimeout = 3
	}

	serverConfig := fmt.Sprintf(`
bindAddr = "127.0.0.1"
bindPort = %d
log.level = "trace"
transport.tcpMux = %t
transport.tcpMuxKeepaliveInterval = 30
transport.heartbeatTimeout = %d
webServer.addr = "127.0.0.1"
webServer.port = %d
webServer.pprofEnable = true
enablePrometheus = true
`, serverPort, tcpMux, heartbeatTimeout, dashboardPort)
	serverConfigPath := f.WriteTempFile("issue-5391-frps.toml", serverConfig)
	serverProcess, _, err := f.StartFrps("-c", serverConfigPath)
	framework.ExpectNoError(err)
	framework.ExpectNoError(framework.WaitForTCPReady(fmt.Sprintf("127.0.0.1:%d", serverPort), 5*time.Second))

	halfOpenRelay := relay.New(fmt.Sprintf("127.0.0.1:%d", serverPort))
	f.RunServer("", halfOpenRelay)

	heartbeatInterval := int64(-1)
	clientHeartbeatTimeout := int64(-1)
	if !tcpMux {
		heartbeatInterval = 1
		clientHeartbeatTimeout = 3
	}
	clientConfig := fmt.Sprintf(`
serverAddr = "127.0.0.1"
serverPort = %d
clientID = "issue-5391"
loginFailExit = false
log.level = "trace"
transport.wireProtocol = %q
transport.tcpMux = %t
transport.tcpMuxKeepaliveInterval = 1
transport.heartbeatInterval = %d
transport.heartbeatTimeout = %d
transport.tls.enable = false

[[proxies]]
name = "issue-5391-tcp"
type = "tcp"
localPort = %d
remotePort = %d
`, halfOpenRelay.BindPort(), wireProtocol, tcpMux, heartbeatInterval, clientHeartbeatTimeout,
		f.PortByName(framework.TCPEchoServerPort), remotePort)
	clientConfigPath := f.WriteTempFile("issue-5391-frpc.toml", clientConfig)
	clientProcess, _, err := f.StartFrpc("-c", clientConfigPath)
	framework.ExpectNoError(err)
	framework.ExpectNoError(halfOpenRelay.WaitForConnections(1, 5*time.Second))
	framework.ExpectNoError(clientProcess.WaitForOutput("login to server success", 1, 10*time.Second))
	framework.ExpectNoError(clientProcess.WaitForOutput("[issue-5391-tcp] start proxy success", 1, 10*time.Second))
	framework.ExpectNoError(waitForReplacementState(dashboardPort, remotePort, 10*time.Second))

	replacementCount := 1
	if tcpMux {
		replacementCount = 3
	}
	for i := 0; i < replacementCount; i++ {
		connectionIndex := 1
		if tcpMux {
			connectionIndex = i + 1
		}
		framework.ExpectNoError(halfOpenRelay.Blackhole(connectionIndex))
		if tcpMux {
			framework.ExpectNoError(halfOpenRelay.WaitForConnections(connectionIndex+1, 15*time.Second))
		}
		framework.ExpectNoError(clientProcess.WaitForOutput("login to server success", i+2, 15*time.Second))
		framework.ExpectNoError(clientProcess.WaitForOutput("[issue-5391-tcp] start proxy success", i+2, 15*time.Second))
		framework.ExpectNoError(waitForReplacementState(dashboardPort, remotePort, 10*time.Second))
	}

	_ = clientProcess.Stop()
	select {
	case <-clientProcess.Done():
	case <-time.After(5 * time.Second):
		framework.Failf("frpc did not exit")
	}
	framework.ExpectNoError(waitForReplacementShutdown(dashboardPort, 10*time.Second))
	framework.ExpectNoError(halfOpenRelay.Close())
	framework.ExpectNoError(waitForNoHandoffWaiters(dashboardPort, 5*time.Second))

	_ = serverProcess.Stop()
	select {
	case <-serverProcess.Done():
	case <-time.After(5 * time.Second):
		framework.Failf("frps did not exit")
	}
}

func waitForReplacementState(dashboardPort, remotePort int, timeout time.Duration) error {
	return waitForLifecycleCondition(timeout, func() error {
		metricsBody, err := getLifecycleEndpoint(dashboardPort, "/metrics")
		if err != nil {
			return err
		}
		if err := expectMetricValue(metricsBody, "frp_server_client_counts", "", 1); err != nil {
			return err
		}
		if err := expectMetricValue(metricsBody, "frp_server_proxy_counts", `type="tcp"`, 1); err != nil {
			return err
		}
		clients, err := getOnlineLifecycleClients(dashboardPort)
		if err != nil {
			return err
		}
		if len(clients) != 1 || clients[0].ClientID != "issue-5391" {
			return fmt.Errorf("expected one online client, got %+v", clients)
		}
		profile, err := getLifecycleEndpoint(dashboardPort, "/debug/pprof/goroutine?debug=2")
		if err != nil {
			return err
		}
		if err := expectNoHandoffWaiter(profile, "after replacement"); err != nil {
			return err
		}
		resp, err := request.New().
			TCP().
			Port(remotePort).
			Timeout(time.Second).
			Body([]byte(consts.TestString)).
			Do()
		if err != nil {
			return err
		}
		if string(resp.Content) != consts.TestString {
			return fmt.Errorf("unexpected proxy response %q", resp.Content)
		}
		return nil
	})
}

func waitForReplacementShutdown(dashboardPort int, timeout time.Duration) error {
	return waitForLifecycleCondition(timeout, func() error {
		metricsBody, err := getLifecycleEndpoint(dashboardPort, "/metrics")
		if err != nil {
			return err
		}
		if err := expectMetricValue(metricsBody, "frp_server_client_counts", "", 0); err != nil {
			return err
		}
		if err := expectMetricValue(metricsBody, "frp_server_proxy_counts", `type="tcp"`, 0); err != nil {
			return err
		}
		clients, err := getOnlineLifecycleClients(dashboardPort)
		if err != nil {
			return err
		}
		if len(clients) != 0 {
			return fmt.Errorf("expected no online clients, got %+v", clients)
		}
		return nil
	})
}

func waitForNoHandoffWaiters(dashboardPort int, timeout time.Duration) error {
	return waitForLifecycleCondition(timeout, func() error {
		profile, err := getLifecycleEndpoint(dashboardPort, "/debug/pprof/goroutine?debug=2")
		if err != nil {
			return err
		}
		return expectNoHandoffWaiter(profile, "after relay shutdown")
	})
}

type lifecycleClient struct {
	ClientID string `json:"clientID"`
}

func expectNoHandoffWaiter(profile, phase string) error {
	if strings.Contains(profile, "(*Control).WaitForHandoff") {
		return fmt.Errorf("control handoff waiter remained %s", phase)
	}
	return nil
}

func getOnlineLifecycleClients(dashboardPort int) ([]lifecycleClient, error) {
	body, err := getLifecycleEndpoint(dashboardPort, "/api/clients?status=online")
	if err != nil {
		return nil, err
	}
	var clients []lifecycleClient
	if err := json.Unmarshal([]byte(body), &clients); err != nil {
		return nil, err
	}
	return clients, nil
}

func getLifecycleEndpoint(port int, path string) (string, error) {
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, path))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s returned %s", path, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func expectMetricValue(body, name, labels string, want float64) error {
	prefix := name
	if labels != "" {
		prefix += "{" + labels + "}"
	}
	for line := range strings.SplitSeq(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[0] != prefix {
			continue
		}
		got, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("metric %s = %v, want %v", prefix, got, want)
		}
		return nil
	}
	return fmt.Errorf("metric %s not found", prefix)
}

func waitForLifecycleCondition(timeout time.Duration, condition func() error) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		err := condition()
		if err == nil {
			return nil
		}
		lastErr = err
		select {
		case <-ticker.C:
		case <-timer.C:
			return fmt.Errorf("condition was not met: %w", lastErr)
		}
	}
}

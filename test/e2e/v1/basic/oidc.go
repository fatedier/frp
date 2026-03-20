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

package basic

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/oidcserver"
	"github.com/fatedier/frp/test/e2e/pkg/port"
)

var _ = ginkgo.Describe("[Feature: OIDC]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("should work with OIDC authentication", func() {
		oidcSrv := oidcserver.New(oidcserver.WithBindPort(f.AllocPort()))
		f.RunServer("", oidcSrv)

		portName := port.GenName("TCP")

		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
auth.method = "oidc"
auth.oidc.issuer = "%s"
auth.oidc.audience = "frps"
`, oidcSrv.Issuer())

		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
auth.method = "oidc"
auth.oidc.clientID = "test-client"
auth.oidc.clientSecret = "test-secret"
auth.oidc.tokenEndpointURL = "%s"

[[proxies]]
name = "tcp"
type = "tcp"
localPort = {{ .%s }}
remotePort = {{ .%s }}
`, oidcSrv.TokenEndpoint(), framework.TCPEchoServerPort, portName)

		f.RunProcesses(serverConf, []string{clientConf})
		framework.NewRequestExpect(f).PortName(portName).Ensure()
	})

	ginkgo.It("should authenticate heartbeats with OIDC", func() {
		oidcSrv := oidcserver.New(oidcserver.WithBindPort(f.AllocPort()))
		f.RunServer("", oidcSrv)

		serverPort := f.AllocPort()
		remotePort := f.AllocPort()

		serverConf := fmt.Sprintf(`
bindAddr = "0.0.0.0"
bindPort = %d
log.level = "trace"
auth.method = "oidc"
auth.additionalScopes = ["HeartBeats"]
auth.oidc.issuer = "%s"
auth.oidc.audience = "frps"
`, serverPort, oidcSrv.Issuer())

		clientConf := fmt.Sprintf(`
serverAddr = "127.0.0.1"
serverPort = %d
loginFailExit = false
log.level = "trace"
auth.method = "oidc"
auth.additionalScopes = ["HeartBeats"]
auth.oidc.clientID = "test-client"
auth.oidc.clientSecret = "test-secret"
auth.oidc.tokenEndpointURL = "%s"
transport.heartbeatInterval = 1

[[proxies]]
name = "tcp"
type = "tcp"
localPort = %d
remotePort = %d
`, serverPort, oidcSrv.TokenEndpoint(), f.PortByName(framework.TCPEchoServerPort), remotePort)

		serverConfigPath := f.GenerateConfigFile(serverConf)
		clientConfigPath := f.GenerateConfigFile(clientConf)

		_, _, err := f.RunFrps("-c", serverConfigPath)
		framework.ExpectNoError(err)
		clientProcess, _, err := f.RunFrpc("-c", clientConfigPath)
		framework.ExpectNoError(err)

		// Wait for several authenticated heartbeat cycles instead of a fixed sleep.
		err = clientProcess.WaitForOutput("send heartbeat to server", 3, 10*time.Second)
		framework.ExpectNoError(err)

		// Proxy should still work: heartbeat auth has not failed.
		framework.NewRequestExpect(f).Port(remotePort).Ensure()
	})

	ginkgo.It("should work when token has no expires_in", func() {
		oidcSrv := oidcserver.New(
			oidcserver.WithBindPort(f.AllocPort()),
			oidcserver.WithExpiresIn(0),
		)
		f.RunServer("", oidcSrv)

		portName := port.GenName("TCP")

		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
auth.method = "oidc"
auth.oidc.issuer = "%s"
auth.oidc.audience = "frps"
`, oidcSrv.Issuer())

		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
auth.method = "oidc"
auth.additionalScopes = ["HeartBeats"]
auth.oidc.clientID = "test-client"
auth.oidc.clientSecret = "test-secret"
auth.oidc.tokenEndpointURL = "%s"
transport.heartbeatInterval = 1

[[proxies]]
name = "tcp"
type = "tcp"
localPort = {{ .%s }}
remotePort = {{ .%s }}
`, oidcSrv.TokenEndpoint(), framework.TCPEchoServerPort, portName)

		_, clientProcesses := f.RunProcesses(serverConf, []string{clientConf})
		framework.NewRequestExpect(f).PortName(portName).Ensure()

		countAfterLogin := oidcSrv.TokenRequestCount()

		// Wait for several heartbeat cycles instead of a fixed sleep.
		// Each heartbeat fetches a fresh token in non-caching mode.
		err := clientProcesses[0].WaitForOutput("send heartbeat to server", 3, 10*time.Second)
		framework.ExpectNoError(err)

		framework.NewRequestExpect(f).PortName(portName).Ensure()

		// Each heartbeat should have fetched a new token (non-caching mode).
		countAfterHeartbeats := oidcSrv.TokenRequestCount()
		framework.ExpectTrue(
			countAfterHeartbeats > countAfterLogin,
			"expected additional token requests for heartbeats, got %d before and %d after",
			countAfterLogin, countAfterHeartbeats,
		)
	})

	ginkgo.It("should reject invalid OIDC credentials", func() {
		oidcSrv := oidcserver.New(oidcserver.WithBindPort(f.AllocPort()))
		f.RunServer("", oidcSrv)

		portName := port.GenName("TCP")

		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
auth.method = "oidc"
auth.oidc.issuer = "%s"
auth.oidc.audience = "frps"
`, oidcSrv.Issuer())

		clientConf := consts.DefaultClientConfig + fmt.Sprintf(`
auth.method = "oidc"
auth.oidc.clientID = "test-client"
auth.oidc.clientSecret = "wrong-secret"
auth.oidc.tokenEndpointURL = "%s"

[[proxies]]
name = "tcp"
type = "tcp"
localPort = {{ .%s }}
remotePort = {{ .%s }}
`, oidcSrv.TokenEndpoint(), framework.TCPEchoServerPort, portName)

		f.RunProcesses(serverConf, []string{clientConf})
		framework.NewRequestExpect(f).PortName(portName).ExpectError(true).Ensure()
	})
})

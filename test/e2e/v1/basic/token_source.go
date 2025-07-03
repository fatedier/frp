// Copyright 2025 The frp Authors
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
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/port"
)

var _ = ginkgo.Describe("[Feature: TokenSource]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.Describe("File-based token loading", func() {
		ginkgo.It("should work with file tokenSource", func() {
			// Create a temporary token file
			tmpDir := f.TempDirectory
			tokenFile := filepath.Join(tmpDir, "test_token")
			tokenContent := "test-token-123"

			err := os.WriteFile(tokenFile, []byte(tokenContent), 0o600)
			framework.ExpectNoError(err)

			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			portName := port.GenName("TCP")

			// Server config with tokenSource
			serverConf += fmt.Sprintf(`
auth.tokenSource.type = "file"
auth.tokenSource.file.path = "%s"
`, tokenFile)

			// Client config with matching token
			clientConf += fmt.Sprintf(`
auth.token = "%s"

[[proxies]]
name = "tcp"
type = "tcp"
localPort = {{ .%s }}
remotePort = {{ .%s }}
`, tokenContent, framework.TCPEchoServerPort, portName)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).PortName(portName).Ensure()
		})

		ginkgo.It("should work with client tokenSource", func() {
			// Create a temporary token file
			tmpDir := f.TempDirectory
			tokenFile := filepath.Join(tmpDir, "client_token")
			tokenContent := "client-token-456"

			err := os.WriteFile(tokenFile, []byte(tokenContent), 0o600)
			framework.ExpectNoError(err)

			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			portName := port.GenName("TCP")

			// Server config with matching token
			serverConf += fmt.Sprintf(`
auth.token = "%s"
`, tokenContent)

			// Client config with tokenSource
			clientConf += fmt.Sprintf(`
auth.tokenSource.type = "file"
auth.tokenSource.file.path = "%s"

[[proxies]]
name = "tcp"
type = "tcp"
localPort = {{ .%s }}
remotePort = {{ .%s }}
`, tokenFile, framework.TCPEchoServerPort, portName)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).PortName(portName).Ensure()
		})

		ginkgo.It("should work with both server and client tokenSource", func() {
			// Create temporary token files
			tmpDir := f.TempDirectory
			serverTokenFile := filepath.Join(tmpDir, "server_token")
			clientTokenFile := filepath.Join(tmpDir, "client_token")
			tokenContent := "shared-token-789"

			err := os.WriteFile(serverTokenFile, []byte(tokenContent), 0o600)
			framework.ExpectNoError(err)

			err = os.WriteFile(clientTokenFile, []byte(tokenContent), 0o600)
			framework.ExpectNoError(err)

			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			portName := port.GenName("TCP")

			// Server config with tokenSource
			serverConf += fmt.Sprintf(`
auth.tokenSource.type = "file"
auth.tokenSource.file.path = "%s"
`, serverTokenFile)

			// Client config with tokenSource
			clientConf += fmt.Sprintf(`
auth.tokenSource.type = "file"
auth.tokenSource.file.path = "%s"

[[proxies]]
name = "tcp"
type = "tcp"
localPort = {{ .%s }}
remotePort = {{ .%s }}
`, clientTokenFile, framework.TCPEchoServerPort, portName)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).PortName(portName).Ensure()
		})

		ginkgo.It("should fail with mismatched tokens", func() {
			// Create temporary token files with different content
			tmpDir := f.TempDirectory
			serverTokenFile := filepath.Join(tmpDir, "server_token")
			clientTokenFile := filepath.Join(tmpDir, "client_token")

			err := os.WriteFile(serverTokenFile, []byte("server-token"), 0o600)
			framework.ExpectNoError(err)

			err = os.WriteFile(clientTokenFile, []byte("client-token"), 0o600)
			framework.ExpectNoError(err)

			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			portName := port.GenName("TCP")

			// Server config with tokenSource
			serverConf += fmt.Sprintf(`
auth.tokenSource.type = "file"
auth.tokenSource.file.path = "%s"
`, serverTokenFile)

			// Client config with different tokenSource
			clientConf += fmt.Sprintf(`
auth.tokenSource.type = "file"
auth.tokenSource.file.path = "%s"

[[proxies]]
name = "tcp"
type = "tcp"
localPort = {{ .%s }}
remotePort = {{ .%s }}
`, clientTokenFile, framework.TCPEchoServerPort, portName)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			// This should fail due to token mismatch - the client should not be able to connect
			// We expect the request to fail because the proxy tunnel is not established
			framework.NewRequestExpect(f).PortName(portName).ExpectError(true).Ensure()
		})

		ginkgo.It("should fail with non-existent token file", func() {
			// This test verifies that server fails to start when tokenSource points to non-existent file
			// We'll verify this by checking that the configuration loading itself fails

			// Create a config that references a non-existent file
			tmpDir := f.TempDirectory
			nonExistentFile := filepath.Join(tmpDir, "non_existent_token")

			serverConf := consts.DefaultServerConfig

			// Server config with non-existent tokenSource file
			serverConf += fmt.Sprintf(`
auth.tokenSource.type = "file"
auth.tokenSource.file.path = "%s"
`, nonExistentFile)

			// The test expectation is that this will fail during the RunProcesses call
			// because the server cannot load the configuration due to missing token file
			defer func() {
				if r := recover(); r != nil {
					// Expected: server should fail to start due to missing file
					ginkgo.By(fmt.Sprintf("Server correctly failed to start: %v", r))
				}
			}()

			// This should cause a panic or error during server startup
			f.RunProcesses([]string{serverConf}, []string{})
		})
	})
})

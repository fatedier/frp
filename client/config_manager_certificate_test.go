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

//go:build !frps

package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/client/visitor"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/policy/security"
	"github.com/fatedier/frp/pkg/transport"
)

type certificateReloadSender struct{ started chan string }

func (s *certificateReloadSender) Send(m msg.Message) error {
	if start, ok := m.(*msg.NewProxy); ok {
		s.started <- start.ProxyName
	}
	return nil
}

func writeReloadCertificate(t *testing.T, crtPath, keyPath string) []byte {
	t.Helper()
	cfg, err := transport.NewServerTLSConfig("", "", "")
	require.NoError(t, err)
	cert := cfg.Certificates[0]
	key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(crtPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0o600))
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), 0o600))
	return cert.Certificate[0]
}

func TestFileReloadCertificatesAndStoreMutations(t *testing.T) {
	for _, pluginType := range []string{v1.PluginHTTPS2HTTP, v1.PluginHTTPS2HTTPS} {
		t.Run(pluginType, func(t *testing.T) {
			dir := t.TempDir()
			crtPath, keyPath := filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key")
			original := writeReloadCertificate(t, crtPath, keyPath)
			configPath := filepath.Join(dir, "frpc.toml")
			content := fmt.Sprintf(`[[proxies]]
name = "https"
type = "tcp"
remotePort = 8443
[proxies.plugin]
type = %q
localAddr = "127.0.0.1:80"
crtPath = %q
keyPath = %q
`, pluginType, crtPath, keyPath)
			require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))
			result, err := config.LoadClientConfigResult(configPath, true)
			require.NoError(t, err)
			configSource := source.NewConfigSource()
			storeSource, err := source.NewStoreSource(source.StoreSourceConfig{Path: filepath.Join(dir, "store.json")})
			require.NoError(t, err)
			aggregator := source.NewAggregator(configSource)
			aggregator.SetStoreSource(storeSource)
			sender := &certificateReloadSender{started: make(chan string, 4)}
			transporter := transport.NewMessageTransporter(sender)
			ctx := context.Background()
			pm := proxy.NewManager(ctx, result.Common, nil, transporter, nil, "")
			vm := visitor.NewManager(ctx, "", result.Common, nil, transporter, nil)
			t.Cleanup(pm.Close)
			t.Cleanup(vm.Close)
			svr := &Service{
				common:         result.Common,
				configFilePath: configPath,
				unsafeFeatures: security.NewUnsafeFeatures(nil),
				aggregator:     aggregator,
				configSource:   configSource,
				storeSource:    storeSource,
				ctl:            &Control{pm: pm, vm: vm},
			}
			require.NoError(t, svr.UpdateConfigSource(result.Common, result.Proxies, nil))
			select {
			case name := <-sender.started:
				require.Equal(t, "https", name)
			case <-time.After(5 * time.Second):
				t.Fatal("proxy did not request startup")
			}
			require.NoError(t, pm.StartProxy("https", ":8443", ""))
			manager := &serviceConfigManager{svr: svr}

			connect := func(wantCert []byte) *tls.Conn {
				t.Helper()
				clientRaw, serverRaw := net.Pipe()
				t.Cleanup(func() { _ = clientRaw.Close(); _ = serverRaw.Close() })
				require.NoError(t, clientRaw.SetDeadline(time.Now().Add(5*time.Second)))
				pm.HandleWorkConn("https", serverRaw, &msg.StartWorkConn{})
				conn := tls.Client(clientRaw, &tls.Config{ServerName: "localhost", InsecureSkipVerify: true})
				require.NoError(t, conn.Handshake())
				require.Equal(t, wantCert, conn.ConnectionState().PeerCertificates[0].Raw)
				return conn
			}
			oldConn := connect(original)
			reader := bufio.NewReader(oldConn)
			checkExistingConnection := func() {
				t.Helper()
				require.NoError(t, oldConn.SetDeadline(time.Now().Add(5*time.Second)))
				// A mismatched Host is handled locally, so no upstream network is needed.
				_, err := io.WriteString(oldConn, "GET / HTTP/1.1\r\nHost: other.example\r\n\r\n")
				require.NoError(t, err)
				resp, err := http.ReadResponse(reader, nil)
				require.NoError(t, err)
				require.Equal(t, http.StatusMisdirectedRequest, resp.StatusCode)
				_, err = io.Copy(io.Discard, resp.Body)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
			}
			checkExistingConnection()
			require.NoError(t, manager.ReloadFromFile(true))
			require.NoError(t, connect(original).NetConn().Close())

			next := writeReloadCertificate(t, crtPath, keyPath)
			// Both store mutation helpers must leave unchanged proxies' certificates alone.
			_, err = manager.CreateStoreProxy(result.Proxies[0].Clone())
			require.NoError(t, err)
			require.NoError(t, connect(original).NetConn().Close())
			_, err = manager.UpdateStoreProxy("https", result.Proxies[0].Clone())
			require.NoError(t, err)
			require.NoError(t, connect(original).NetConn().Close())
			require.NoError(t, manager.DeleteStoreProxy("https"))
			require.NoError(t, connect(original).NetConn().Close())
			require.NoError(t, manager.ReloadFromFile(true))
			require.NoError(t, connect(next).NetConn().Close())
			checkExistingConnection()

			for _, badFile := range []string{"missing", "invalid"} {
				saved, err := os.ReadFile(crtPath)
				require.NoError(t, err)
				if badFile == "missing" {
					require.NoError(t, os.Remove(crtPath))
				} else {
					require.NoError(t, os.WriteFile(crtPath, []byte("invalid"), 0o600))
				}
				require.NoError(t, manager.ReloadFromFile(true), "certificate errors must not fail reload")
				status, ok := pm.GetProxyStatus("https")
				require.True(t, ok)
				require.Equal(t, proxy.ProxyPhaseRunning, status.Phase)
				require.Contains(t, status.Err, "reload plugin ")
				require.NoError(t, connect(next).NetConn().Close())
				checkExistingConnection()
				require.NoError(t, os.WriteFile(crtPath, saved, 0o600))
				require.NoError(t, manager.ReloadFromFile(true))
				status, _ = pm.GetProxyStatus("https")
				require.Empty(t, status.Err, "restoring the active bytes must clear the reload error")
			}

			// A config change still replaces the proxy and uses normal startup errors.
			require.NoError(t, oldConn.NetConn().Close())
			require.NoError(t, os.WriteFile(crtPath, []byte("invalid"), 0o600))
			changedConfig := strings.Replace(content, "remotePort = 8443", "remotePort = 8444", 1)
			require.NoError(t, os.WriteFile(configPath, []byte(changedConfig), 0o600))
			require.NoError(t, manager.ReloadFromFile(true))
			select {
			case name := <-sender.started:
				require.Equal(t, "https", name)
			case <-time.After(5 * time.Second):
				t.Fatal("replacement proxy did not request startup")
			}
			require.Error(t, pm.StartProxy("https", ":8444", ""))
			status, ok := pm.GetProxyStatus("https")
			require.True(t, ok)
			require.Equal(t, proxy.ProxyPhaseStartErr, status.Phase)
			require.NotEmpty(t, status.Err)
		})
	}
}

func TestFileReloadCertificatesDuringControlHandoff(t *testing.T) {
	for _, pluginType := range []string{v1.PluginHTTPS2HTTP, v1.PluginHTTPS2HTTPS} {
		t.Run(pluginType, func(t *testing.T) {
			dir := t.TempDir()
			crtPath, keyPath := filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key")
			original := writeReloadCertificate(t, crtPath, keyPath)
			configPath := filepath.Join(dir, "frpc.toml")
			require.NoError(t, os.WriteFile(configPath, fmt.Appendf(nil, `[[proxies]]
name = "https"
type = "tcp"
remotePort = 8443
[proxies.plugin]
type = %q
localAddr = "127.0.0.1:80"
crtPath = %q
keyPath = %q
`, pluginType, crtPath, keyPath), 0o600))
			result, err := config.LoadClientConfigResult(configPath, true)
			require.NoError(t, err)
			configSource := source.NewConfigSource()
			svr := &Service{
				common:         result.Common,
				configFilePath: configPath,
				unsafeFeatures: security.NewUnsafeFeatures(nil),
				aggregator:     source.NewAggregator(configSource),
				configSource:   configSource,
			}
			require.NoError(t, svr.UpdateConfigSource(result.Common, result.Proxies, nil))
			sender := &certificateReloadSender{started: make(chan string, 1)}
			transporter := transport.NewMessageTransporter(sender)
			newControl := func() *Control {
				ctx := context.Background()
				pm := proxy.NewManager(ctx, result.Common, nil, transporter, nil, "")
				vm := visitor.NewManager(ctx, "", result.Common, nil, transporter, nil)
				t.Cleanup(pm.Close)
				t.Cleanup(vm.Close)
				return &Control{pm: pm, vm: vm}
			}
			previous := newControl()
			previous.pm.Close()
			svr.ctl = previous
			next := newControl()

			// Model the handoff transaction: start the new control's proxies while
			// the published control is still the closed, empty previous manager.
			svr.reloadMu.Lock()
			releaseTransaction := sync.OnceFunc(svr.reloadMu.Unlock)
			defer releaseTransaction()
			next.pm.UpdateAll(svr.proxyCfgs)
			select {
			case name := <-sender.started:
				require.Equal(t, "https", name)
			case <-time.After(5 * time.Second):
				t.Fatal("replacement proxy did not request startup")
			}
			require.NoError(t, next.pm.StartProxy("https", ":8443", ""))
			checkCertificate := func(want []byte) {
				t.Helper()
				clientRaw, serverRaw := net.Pipe()
				defer clientRaw.Close()
				defer serverRaw.Close()
				require.NoError(t, clientRaw.SetDeadline(time.Now().Add(5*time.Second)))
				next.pm.HandleWorkConn("https", serverRaw, &msg.StartWorkConn{})
				conn := tls.Client(clientRaw, &tls.Config{ServerName: "localhost", InsecureSkipVerify: true})
				require.NoError(t, conn.Handshake())
				require.Equal(t, want, conn.ConnectionState().PeerCertificates[0].Raw)
			}
			checkCertificate(original)
			renewed := writeReloadCertificate(t, crtPath, keyPath)

			svr.ctlMu.Lock()
			releaseControl := sync.OnceFunc(svr.ctlMu.Unlock)
			defer releaseControl()
			reloadStarted, reloadDone := make(chan struct{}), make(chan struct{})
			var reloadErr error
			go func() {
				close(reloadStarted)
				reloadErr = (&serviceConfigManager{svr: svr}).ReloadFromFile(true)
				close(reloadDone)
			}()
			t.Cleanup(func() {
				releaseTransaction()
				releaseControl()
				waitLifecycleSignal(t, reloadDone)
			})
			waitLifecycleSignal(t, reloadStarted)
			svr.ctl = next
			releaseTransaction()

			// Keep ctlMu as a read barrier after publication. Reload must acquire
			// reloadMu before selecting ctl, independently of goroutine scheduling.
			// The deadline only detects a broken lock order; it does not stage the race.
			deadline := time.NewTimer(5 * time.Second)
			defer deadline.Stop()
			for svr.reloadMu.TryLock() {
				svr.reloadMu.Unlock()
				select {
				case <-deadline.C:
					t.Fatal("reload selected ctl without first acquiring reloadMu")
				default:
					runtime.Gosched()
				}
			}
			releaseControl()
			waitLifecycleSignal(t, reloadDone)
			require.NoError(t, reloadErr)
			checkCertificate(renewed)
		})
	}
}

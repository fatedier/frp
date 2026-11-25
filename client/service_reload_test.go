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

package client

import (
	"context"
	"io"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
)

// Block proxy construction after the service has read its config snapshot, but
// before Control.Run returns and the replacement control is published.
type handoffProxyConfig struct {
	*v1.TCPProxyConfig
	started chan struct{}
	resume  chan struct{}
}

func init() {
	proxy.RegisterProxyFactory(reflect.TypeFor[*handoffProxyConfig](), func(base *proxy.BaseProxy, cfg v1.ProxyConfigurer) proxy.Proxy {
		blocking := cfg.(*handoffProxyConfig)
		close(blocking.started)
		<-blocking.resume
		return proxy.NewGeneralTCPProxy(base, blocking.TCPProxyConfig)
	})
}

func newReloadTestService(t *testing.T, beforeLogin func()) *Service {
	t.Helper()
	svr, err := NewService(ServiceOptions{
		Common: &v1.ClientCommonConfig{
			Transport: v1.ClientTransportConfig{HeartbeatInterval: -1, HeartbeatTimeout: -1},
		},
		ConfigSourceAggregator: source.NewAggregator(source.NewConfigSource()),
		ConnectorCreator: func(context.Context, *v1.ClientCommonConfig) Connector {
			clientConn, serverConn := net.Pipe()
			peerDone := make(chan struct{})
			t.Cleanup(func() {
				_ = clientConn.Close()
				_ = serverConn.Close()
				waitLifecycleSignal(t, peerDone)
			})
			go func() {
				defer close(peerDone)
				rw := msg.NewV1ReadWriter(serverConn)
				var login msg.Login
				if err := rw.ReadMsgInto(&login); err != nil {
					t.Errorf("read login: %v", err)
					return
				}
				if beforeLogin != nil {
					beforeLogin()
				}
				if err := rw.WriteMsg(&msg.LoginResp{RunID: "reload-test"}); err != nil {
					t.Errorf("write login response: %v", err)
					return
				}
				_, _ = io.Copy(io.Discard, serverConn)
			}()
			return &testConnector{conn: clientConn}
		},
	})
	require.NoError(t, err)
	svr.ctx, svr.cancel = context.WithCancelCause(context.Background())
	t.Cleanup(func() {
		svr.cancel(nil)
		if svr.ctl != nil {
			_ = svr.ctl.Close()
			waitLifecycleSignal(t, svr.ctl.Done())
		}
	})
	return svr
}

func TestServiceReloadDuringControlHandoff(t *testing.T) {
	for _, updateSource := range []bool{false, true} {
		name := "direct update"
		if updateSource {
			name = "config source reload"
		}
		t.Run(name, func(t *testing.T) {
			svr := newReloadTestService(t, nil)
			svr.loopLoginUntilSuccess(time.Second, true)
			previous := svr.ctl
			require.NotNil(t, previous)
			require.NoError(t, previous.Close())
			waitLifecycleSignal(t, previous.Done())

			oldProxy := &handoffProxyConfig{
				TCPProxyConfig: &v1.TCPProxyConfig{
					ProxyBaseConfig: v1.ProxyBaseConfig{Name: "proxy", Type: "tcp"},
					RemotePort:      18001,
				},
				started: make(chan struct{}),
				resume:  make(chan struct{}),
			}
			oldVisitor := &v1.STCPVisitorConfig{VisitorBaseConfig: v1.VisitorBaseConfig{
				Name: "visitor", Type: "stcp", BindPort: -1, ServerName: "old-target",
			}}
			// Seed the desired config for the next connection without restarting
			// proxies on the already closed control.
			svr.proxyCfgs = []v1.ProxyConfigurer{oldProxy}
			svr.visitorCfgs = []v1.VisitorConfigurer{oldVisitor}
			loginDone, updateDone := make(chan struct{}), make(chan struct{})
			resume := sync.OnceFunc(func() { close(oldProxy.resume) })
			t.Cleanup(func() {
				resume()
				waitLifecycleSignal(t, loginDone)
			})
			go func() {
				svr.loopLoginUntilSuccess(time.Second, true)
				close(loginDone)
			}()
			waitLifecycleSignal(t, oldProxy.started)
			svr.ctlMu.RLock()
			current := svr.ctl
			svr.ctlMu.RUnlock()
			require.Same(t, previous, current, "replacement must still be unpublished")
			// Check before starting the updater so it cannot be the lock owner.
			unlocked := svr.reloadMu.TryLock()
			if unlocked {
				svr.reloadMu.Unlock()
			}

			newProxy := &v1.TCPProxyConfig{
				ProxyBaseConfig: v1.ProxyBaseConfig{Name: "proxy", Type: "tcp"},
				RemotePort:      18002,
			}
			newVisitor := &v1.STCPVisitorConfig{VisitorBaseConfig: v1.VisitorBaseConfig{
				Name: "visitor", Type: "stcp", BindPort: -1, ServerName: "new-target",
			}}
			var updateErr error
			go func() {
				proxies, visitors := []v1.ProxyConfigurer{newProxy}, []v1.VisitorConfigurer{newVisitor}
				if updateSource {
					updateErr = svr.UpdateConfigSource(svr.common, proxies, visitors)
				} else {
					updateErr = svr.UpdateAllConfigurer(proxies, visitors)
				}
				close(updateDone)
			}()
			t.Cleanup(func() {
				resume()
				waitLifecycleSignal(t, updateDone)
			})

			// The gate fixes the handoff window. On the old implementation the
			// update completes against the retired control before this gate opens.
			select {
			case <-updateDone:
				t.Error("config update completed before control handoff")
			case <-time.After(20 * time.Millisecond):
			}
			require.False(t, unlocked, "control startup must hold the reload transaction until publication")
			resume()
			waitLifecycleSignal(t, loginDone)
			waitLifecycleSignal(t, updateDone)
			require.NoError(t, updateErr)
			require.NotSame(t, previous, svr.ctl)
			status, ok := svr.getProxyStatus("proxy")
			require.True(t, ok)
			require.Equal(t, 18002, status.Cfg.(*v1.TCPProxyConfig).RemotePort)
			visitor, ok := svr.getVisitorCfg("visitor")
			require.True(t, ok)
			require.Equal(t, "new-target", visitor.GetBaseConfig().ServerName)
		})
	}
}

func TestServiceReloadWhileLoginPending(t *testing.T) {
	loginStarted, allowLogin, loginDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	resume := sync.OnceFunc(func() { close(allowLogin) })
	svr := newReloadTestService(t, func() {
		close(loginStarted)
		<-allowLogin
	})
	t.Cleanup(func() {
		resume()
		waitLifecycleSignal(t, loginDone)
	})
	go func() {
		svr.loopLoginUntilSuccess(time.Second, true)
		close(loginDone)
	}()
	waitLifecycleSignal(t, loginStarted)

	proxyCfg := &v1.TCPProxyConfig{
		ProxyBaseConfig: v1.ProxyBaseConfig{Name: "during-login", Type: "tcp"},
		RemotePort:      18003,
	}
	updateDone := make(chan struct{})
	var updateErr error
	go func() {
		updateErr = svr.UpdateAllConfigurer([]v1.ProxyConfigurer{proxyCfg}, nil)
		close(updateDone)
	}()
	t.Cleanup(func() {
		resume()
		waitLifecycleSignal(t, updateDone)
	})
	// Config updates must complete even while the server has not replied to login.
	waitLifecycleSignal(t, updateDone)
	require.NoError(t, updateErr)
	resume()
	waitLifecycleSignal(t, loginDone)
	status, ok := svr.getProxyStatus("during-login")
	require.True(t, ok)
	require.Equal(t, 18003, status.Cfg.(*v1.TCPProxyConfig).RemotePort)
}

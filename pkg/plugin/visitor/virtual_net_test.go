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

package visitor

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/util/xlog"
)

const testVirtualNetVisitorName = "vnet-visitor"

type fakeClientRouteController struct {
	mu sync.Mutex

	routes map[string]io.Writer

	beforeRegister  func()
	registerCalls   int
	unregisterCalls int
}

func newFakeClientRouteController() *fakeClientRouteController {
	return &fakeClientRouteController{
		routes: make(map[string]io.Writer),
	}
}

func (c *fakeClientRouteController) RegisterClientRoute(
	_ context.Context,
	name string,
	_ []net.IPNet,
	conn io.ReadWriteCloser,
) {
	if c.beforeRegister != nil {
		c.beforeRegister()
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.registerCalls++
	c.routes[name] = conn
}

func (c *fakeClientRouteController) UnregisterClientRoute(name string, conn io.Writer) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.unregisterCalls++
	owner, ok := c.routes[name]
	if !ok || owner != conn {
		return false
	}
	delete(c.routes, name)
	return true
}

func (c *fakeClientRouteController) owner() io.Writer {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.routes[testVirtualNetVisitorName]
}

func (c *fakeClientRouteController) callCounts() (register, unregister int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.registerCalls, c.unregisterCalls
}

type trackedConn struct {
	net.Conn
	closed atomic.Bool
}

func (c *trackedConn) Close() error {
	c.closed.Store(true)
	return c.Conn.Close()
}

func newTrackedPipe(t *testing.T) (*trackedConn, *trackedConn) {
	t.Helper()
	left, right := net.Pipe()
	trackedLeft := &trackedConn{Conn: left}
	trackedRight := &trackedConn{Conn: right}
	t.Cleanup(func() {
		_ = trackedLeft.Close()
		_ = trackedRight.Close()
	})
	return trackedLeft, trackedRight
}

func newTestVirtualNetPlugin(t *testing.T, controller *fakeClientRouteController) *VirtualNetPlugin {
	t.Helper()
	pluginCtx := context.Background()
	ctx, cancel := context.WithCancel(pluginCtx)
	p := &VirtualNetPlugin{
		pluginCtx: PluginContext{
			Name: testVirtualNetVisitorName,
			Ctx:  pluginCtx,
		},
		routeController: controller,
		routes: []net.IPNet{{
			IP:   net.ParseIP("10.1.0.1"),
			Mask: net.CIDRMask(32, 32),
		}},
		ctx:    ctx,
		cancel: cancel,
	}
	t.Cleanup(func() {
		_ = p.Close()
	})
	return p
}

func waitResult[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case result := <-ch:
		return result
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for concurrent operation")
		var zero T
		return zero
	}
}

// TestVirtualNetReconnectDelay verifies the documented exponential backoff and
// ensures large error counts remain capped instead of overflowing to zero.
func TestVirtualNetReconnectDelay(t *testing.T) {
	tests := []struct {
		name              string
		consecutiveErrors int
		want              time.Duration
	}{
		{name: "first error", consecutiveErrors: 1, want: 60 * time.Second},
		{name: "second error", consecutiveErrors: 2, want: 120 * time.Second},
		{name: "third error", consecutiveErrors: 3, want: 240 * time.Second},
		{name: "fourth error", consecutiveErrors: 4, want: 300 * time.Second},
		{name: "shift width boundary", consecutiveErrors: 64, want: 300 * time.Second},
		{name: "observed retry storm", consecutiveErrors: 329769, want: 300 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, virtualNetReconnectDelay(tt.consecutiveErrors))
		})
	}
}

func TestVirtualNetPluginCloseBeforeRegisterDoesNotReplaceNewRoute(t *testing.T) {
	controller := newFakeClientRouteController()
	oldPlugin := newTestVirtualNetPlugin(t, controller)
	newPlugin := newTestVirtualNetPlugin(t, controller)
	oldControllerConn, oldPluginConn := newTrackedPipe(t)
	newControllerConn, newPluginConn := newTrackedPipe(t)

	allowOldRegister := make(chan struct{})
	oldRegisterResult := make(chan bool, 1)
	go func() {
		<-allowOldRegister
		oldRegisterResult <- oldPlugin.registerControllerConn(oldControllerConn, oldPluginConn)
	}()

	require.NoError(t, oldPlugin.Close())
	require.True(t, newPlugin.registerControllerConn(newControllerConn, newPluginConn))
	close(allowOldRegister)

	require.False(t, waitResult(t, oldRegisterResult))
	require.Same(t, newControllerConn, controller.owner())
	registerCalls, _ := controller.callCounts()
	require.Equal(t, 1, registerCalls)
	require.True(t, oldControllerConn.closed.Load())
	require.True(t, oldPluginConn.closed.Load())
}

func TestVirtualNetPluginRegisterBeforeCloseIsCleanedUp(t *testing.T) {
	controller := newFakeClientRouteController()
	p := newTestVirtualNetPlugin(t, controller)
	controllerConn, pluginConn := newTrackedPipe(t)
	registerEntered := make(chan struct{})
	var registerEnteredOnce sync.Once
	controller.beforeRegister = func() {
		registerEnteredOnce.Do(func() {
			close(registerEntered)
		})
		<-p.ctx.Done()
	}

	registerResult := make(chan bool, 1)
	go func() {
		registerResult <- p.registerControllerConn(controllerConn, pluginConn)
	}()
	waitResult(t, registerEntered)

	closeResult := make(chan error, 1)
	go func() {
		closeResult <- p.Close()
	}()

	require.NoError(t, waitResult(t, closeResult))
	require.True(t, waitResult(t, registerResult))
	require.Nil(t, controller.owner())
	registerCalls, unregisterCalls := controller.callCounts()
	require.Equal(t, 1, registerCalls)
	require.Equal(t, 1, unregisterCalls)
	require.True(t, controllerConn.closed.Load())
}

func TestVirtualNetPluginOldConnectionCleanupKeepsReplacementRoute(t *testing.T) {
	controller := newFakeClientRouteController()
	oldPlugin := newTestVirtualNetPlugin(t, controller)
	newPlugin := newTestVirtualNetPlugin(t, controller)
	oldControllerConn, oldPluginConn := newTrackedPipe(t)
	newControllerConn, newPluginConn := newTrackedPipe(t)

	require.True(t, oldPlugin.registerControllerConn(oldControllerConn, oldPluginConn))
	require.True(t, newPlugin.registerControllerConn(newControllerConn, newPluginConn))
	require.Same(t, newControllerConn, controller.owner())

	oldPlugin.cleanupControllerConn(xlog.FromContextSafe(oldPlugin.ctx), oldControllerConn)

	require.Same(t, newControllerConn, controller.owner())
	require.True(t, oldControllerConn.closed.Load())
}

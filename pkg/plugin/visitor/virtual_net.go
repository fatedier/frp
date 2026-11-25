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

//go:build !frps

package visitor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	netutil "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
)

func init() {
	Register(v1.VisitorPluginVirtualNet, NewVirtualNetPlugin)
}

type clientRouteController interface {
	RegisterClientRoute(context.Context, string, []net.IPNet, io.ReadWriteCloser)
	UnregisterClientRoute(string, io.Writer) bool
}

type VirtualNetPlugin struct {
	pluginCtx PluginContext

	routeController clientRouteController
	routes          []net.IPNet

	mu             sync.Mutex
	controllerConn net.Conn
	closeSignal    chan struct{}

	consecutiveErrors int // Tracks consecutive connection errors for exponential backoff

	ctx    context.Context
	cancel context.CancelFunc
}

const (
	virtualNetReconnectBaseDelay = 60 * time.Second
	virtualNetReconnectMaxDelay  = 300 * time.Second
)

func NewVirtualNetPlugin(pluginCtx PluginContext, options v1.VisitorPluginOptions) (Plugin, error) {
	opts := options.(*v1.VirtualNetVisitorPluginOptions)

	p := &VirtualNetPlugin{
		pluginCtx: pluginCtx,
		routes:    make([]net.IPNet, 0),
	}
	if pluginCtx.VnetController != nil {
		p.routeController = pluginCtx.VnetController
	}

	p.ctx, p.cancel = context.WithCancel(pluginCtx.Ctx)

	if opts.DestinationIP == "" {
		return nil, errors.New("destinationIP is required")
	}

	// Parse DestinationIP and create a host route.
	ip := net.ParseIP(opts.DestinationIP)
	if ip == nil {
		return nil, fmt.Errorf("invalid destination IP address [%s]", opts.DestinationIP)
	}

	var mask net.IPMask
	if ip.To4() != nil {
		mask = net.CIDRMask(32, 32) // /32 for IPv4
	} else {
		mask = net.CIDRMask(128, 128) // /128 for IPv6
	}
	p.routes = append(p.routes, net.IPNet{IP: ip, Mask: mask})

	return p, nil
}

func (p *VirtualNetPlugin) Name() string {
	return v1.VisitorPluginVirtualNet
}

func (p *VirtualNetPlugin) Start() {
	xl := xlog.FromContextSafe(p.pluginCtx.Ctx)
	if p.routeController == nil {
		return
	}

	routeStr := "unknown"
	if len(p.routes) > 0 {
		routeStr = p.routes[0].String()
	}
	xl.Infof("starting VirtualNetPlugin for visitor [%s], attempting to register routes for %s", p.pluginCtx.Name, routeStr)

	go p.run()
}

func (p *VirtualNetPlugin) run() {
	xl := xlog.FromContextSafe(p.ctx)

	for {
		currentCloseSignal := make(chan struct{})

		p.mu.Lock()
		p.closeSignal = currentCloseSignal
		p.mu.Unlock()

		select {
		case <-p.ctx.Done():
			xl.Infof("VirtualNetPlugin run loop for visitor [%s] stopping (context cancelled before pipe creation).", p.pluginCtx.Name)
			p.cleanupCurrentControllerConn(xl)
			return
		default:
		}

		controllerConn, pluginConn := net.Pipe()
		xl.Infof("attempting to register client route for visitor [%s]", p.pluginCtx.Name)
		if !p.registerControllerConn(controllerConn, pluginConn) {
			xl.Infof("VirtualNetPlugin run loop for visitor [%s] stopping (context cancelled before route registration).", p.pluginCtx.Name)
			return
		}

		// Wrap with CloseNotifyConn which supports both close notification and error recording
		var closeErr error
		pluginNotifyConn := netutil.WrapCloseNotifyConn(pluginConn, func(err error) {
			closeErr = err
			close(currentCloseSignal) // Signal the run loop on close.
		})

		xl.Infof("successfully registered client route for visitor [%s]. Starting connection handler with CloseNotifyConn.", p.pluginCtx.Name)

		// Pass the CloseNotifyConn to the visitor for handling.
		// The visitor can call CloseWithError to record the failure reason.
		p.pluginCtx.SendConnToVisitor(pluginNotifyConn)

		// Wait for context cancellation or connection close.
		select {
		case <-p.ctx.Done():
			xl.Infof("VirtualNetPlugin run loop stopping for visitor [%s] (context cancelled while waiting).", p.pluginCtx.Name)
			p.cleanupControllerConn(xl, controllerConn)
			return
		case <-currentCloseSignal:
			// Determine reconnect delay based on error with exponential backoff
			var reconnectDelay time.Duration
			if closeErr != nil {
				p.consecutiveErrors++
				xl.Warnf("connection closed with error for visitor [%s] (consecutive errors: %d): %v",
					p.pluginCtx.Name, p.consecutiveErrors, closeErr)

				// Exponential backoff: 60s, 120s, 240s, 300s (capped)
				reconnectDelay = virtualNetReconnectDelay(p.consecutiveErrors)
			} else {
				// Reset consecutive errors on successful connection
				if p.consecutiveErrors > 0 {
					xl.Infof("connection closed normally for visitor [%s], resetting error counter (was %d)",
						p.pluginCtx.Name, p.consecutiveErrors)
					p.consecutiveErrors = 0
				} else {
					xl.Infof("connection closed normally for visitor [%s]", p.pluginCtx.Name)
				}
				reconnectDelay = 10 * time.Second
			}

			// The visitor closed the plugin side. Close the controller side.
			p.cleanupControllerConn(xl, controllerConn)

			xl.Infof("waiting %v before attempting reconnection for visitor [%s]...", reconnectDelay, p.pluginCtx.Name)
			select {
			case <-time.After(reconnectDelay):
			case <-p.ctx.Done():
				xl.Infof("VirtualNetPlugin reconnection delay interrupted for visitor [%s]", p.pluginCtx.Name)
				return
			}
		}

		xl.Infof("re-establishing virtual connection for visitor [%s]...", p.pluginCtx.Name)
	}
}

// registerControllerConn publishes and registers controllerConn atomically with
// respect to Close. A canceled plugin cannot register a new route.
func (p *VirtualNetPlugin) registerControllerConn(controllerConn, pluginConn net.Conn) bool {
	p.mu.Lock()
	if p.ctx.Err() != nil || p.routeController == nil {
		p.mu.Unlock()
		_ = controllerConn.Close()
		_ = pluginConn.Close()
		return false
	}

	p.controllerConn = controllerConn
	p.routeController.RegisterClientRoute(p.ctx, p.pluginCtx.Name, p.routes, controllerConn)
	p.mu.Unlock()
	return true
}

// virtualNetReconnectDelay returns a bounded reconnect delay without allowing
// the exponential shift to overflow for large consecutive error counts.
func virtualNetReconnectDelay(consecutiveErrors int) time.Duration {
	if consecutiveErrors <= 1 {
		return virtualNetReconnectBaseDelay
	}
	if consecutiveErrors >= 4 {
		return virtualNetReconnectMaxDelay
	}
	return virtualNetReconnectBaseDelay * time.Duration(1<<uint(consecutiveErrors-1))
}

// cleanupControllerConn unregisters and closes one connection round without
// affecting a replacement route owned by another connection.
func (p *VirtualNetPlugin) cleanupControllerConn(xl *xlog.Logger, controllerConn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cleanupControllerConnLocked(xl, controllerConn)
}

func (p *VirtualNetPlugin) cleanupCurrentControllerConn(xl *xlog.Logger) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cleanupControllerConnLocked(xl, p.controllerConn)
}

// cleanupControllerConnLocked must be called with p.mu held.
func (p *VirtualNetPlugin) cleanupControllerConnLocked(xl *xlog.Logger, controllerConn net.Conn) {
	if controllerConn == nil {
		p.closeSignal = nil
		return
	}

	if p.routeController != nil &&
		p.routeController.UnregisterClientRoute(p.pluginCtx.Name, controllerConn) {
		xl.Infof("unregistered client route for visitor [%s]", p.pluginCtx.Name)
	}
	xl.Debugf("cleaning up controllerConn for visitor [%s]", p.pluginCtx.Name)
	_ = controllerConn.Close()
	if p.controllerConn == controllerConn {
		p.controllerConn = nil
		p.closeSignal = nil
	}
}

// Close initiates the plugin shutdown.
func (p *VirtualNetPlugin) Close() error {
	xl := xlog.FromContextSafe(p.pluginCtx.Ctx)
	xl.Infof("closing VirtualNetPlugin for visitor [%s]", p.pluginCtx.Name)

	// Signal the run loop goroutine to stop.
	p.cancel()

	// Unregister and close the current connection while holding the same lock
	// used to check cancellation and register a route in run.
	p.cleanupCurrentControllerConn(xl)
	xl.Infof("finished cleaning up connections during close for visitor [%s]", p.pluginCtx.Name)

	return nil
}

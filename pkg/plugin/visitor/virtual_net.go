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

type VirtualNetPlugin struct {
	pluginCtx PluginContext

	routes []net.IPNet

	mu             sync.Mutex
	controllerConn net.Conn
	closeSignal    chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
}

func NewVirtualNetPlugin(pluginCtx PluginContext, options v1.VisitorPluginOptions) (Plugin, error) {
	opts := options.(*v1.VirtualNetVisitorPluginOptions)

	p := &VirtualNetPlugin{
		pluginCtx: pluginCtx,
		routes:    make([]net.IPNet, 0),
	}

	p.ctx, p.cancel = context.WithCancel(pluginCtx.Ctx)

	if opts.DestinationIP == "" {
		return nil, errors.New("destinationIP is required")
	}

	// Parse DestinationIP as a single IP and create a host route
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
	if p.pluginCtx.VnetController == nil {
		return
	}

	routeStr := "unknown"
	if len(p.routes) > 0 {
		routeStr = p.routes[0].String()
	}
	xl.Infof("Starting VirtualNetPlugin for visitor [%s], attempting to register routes for %s", p.pluginCtx.Name, routeStr)

	go p.run()
}

func (p *VirtualNetPlugin) run() {
	xl := xlog.FromContextSafe(p.ctx)
	reconnectDelay := 10 * time.Second

	for {
		// Create a signal channel for this connection attempt
		currentCloseSignal := make(chan struct{})

		// Store the signal channel under lock
		p.mu.Lock()
		p.closeSignal = currentCloseSignal
		p.mu.Unlock()

		select {
		case <-p.ctx.Done():
			xl.Infof("VirtualNetPlugin run loop for visitor [%s] stopping (context cancelled before pipe creation).", p.pluginCtx.Name)
			// Ensure controllerConn from previous loop is cleaned up if necessary
			p.cleanupControllerConn(xl)
			return
		default:
		}

		controllerConn, pluginConn := net.Pipe()

		// Store controllerConn under lock for cleanup purposes
		p.mu.Lock()
		p.controllerConn = controllerConn
		p.mu.Unlock()

		// Wrap pluginConn using CloseNotifyConn
		pluginNotifyConn := netutil.WrapCloseNotifyConn(pluginConn, func() {
			close(currentCloseSignal) // Signal the run loop
		})

		xl.Infof("Attempting to register client route for visitor [%s]", p.pluginCtx.Name)
		err := p.pluginCtx.VnetController.RegisterClientRoute(p.ctx, p.pluginCtx.Name, p.routes, controllerConn)
		if err != nil {
			xl.Errorf("Failed to register client route for visitor [%s]: %v. Retrying after %v", p.pluginCtx.Name, err, reconnectDelay)
			p.cleanupPipePair(xl, controllerConn, pluginConn) // Close both ends on registration failure

			// Wait before retrying registration, unless context is cancelled
			select {
			case <-time.After(reconnectDelay):
				continue // Retry the loop
			case <-p.ctx.Done():
				xl.Infof("VirtualNetPlugin registration retry wait interrupted for visitor [%s]", p.pluginCtx.Name)
				return // Exit loop if context is cancelled during wait
			}
		}

		xl.Infof("Successfully registered client route for visitor [%s]. Starting connection handler with CloseNotifyConn.", p.pluginCtx.Name)

		// Pass the CloseNotifyConn to HandleConn.
		// HandleConn is responsible for calling Close() on pluginNotifyConn.
		p.pluginCtx.HandleConn(pluginNotifyConn)

		// Wait for either the plugin context to be cancelled or the wrapper's Close() to be called via the signal channel.
		select {
		case <-p.ctx.Done():
			xl.Infof("VirtualNetPlugin run loop stopping for visitor [%s] (context cancelled while waiting).", p.pluginCtx.Name)
			// Context cancelled, ensure controller side is closed if HandleConn didn't close its side yet.
			p.cleanupControllerConn(xl)
			return
		case <-currentCloseSignal:
			xl.Infof("Detected connection closed via CloseNotifyConn for visitor [%s].", p.pluginCtx.Name)
			// HandleConn closed the plugin side (pluginNotifyConn). The closeFn was called, closing currentCloseSignal.
			// We still need to close the controller side.
			p.cleanupControllerConn(xl)

			// Add a delay before attempting to reconnect, respecting context cancellation.
			xl.Infof("Waiting %v before attempting reconnection for visitor [%s]...", reconnectDelay, p.pluginCtx.Name)
			select {
			case <-time.After(reconnectDelay):
				// Delay completed, loop will continue.
			case <-p.ctx.Done():
				xl.Infof("VirtualNetPlugin reconnection delay interrupted for visitor [%s]", p.pluginCtx.Name)
				return // Exit loop if context is cancelled during wait
			}
			// Loop will continue to reconnect.
		}

		// Loop will restart, context check at the beginning of the loop is sufficient.
		xl.Infof("Re-establishing virtual connection for visitor [%s]...", p.pluginCtx.Name)
	}
}

// cleanupControllerConn closes the current controllerConn (if it exists) under lock.
func (p *VirtualNetPlugin) cleanupControllerConn(xl *xlog.Logger) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.controllerConn != nil {
		xl.Debugf("Cleaning up controllerConn for visitor [%s]", p.pluginCtx.Name)
		p.controllerConn.Close()
		p.controllerConn = nil
	}
	// Also clear the closeSignal reference for the completed/cancelled connection attempt
	p.closeSignal = nil
}

// cleanupPipePair closes both ends of a pipe, used typically when registration fails.
func (p *VirtualNetPlugin) cleanupPipePair(xl *xlog.Logger, controllerConn, pluginConn net.Conn) {
	xl.Debugf("Cleaning up pipe pair for visitor [%s] after registration failure", p.pluginCtx.Name)
	controllerConn.Close()
	pluginConn.Close()
	p.mu.Lock()
	p.controllerConn = nil // Ensure field is nil if it was briefly set
	p.closeSignal = nil    // Ensure field is nil if it was briefly set
	p.mu.Unlock()
}

// Close initiates the plugin shutdown.
func (p *VirtualNetPlugin) Close() error {
	xl := xlog.FromContextSafe(p.pluginCtx.Ctx) // Use base context for close logging
	xl.Infof("Closing VirtualNetPlugin for visitor [%s]", p.pluginCtx.Name)

	// 1. Signal the run loop goroutine to stop via context cancellation.
	p.cancel()

	// 2. Unregister the route from the controller.
	// This might implicitly cause the VnetController to close its end of the pipe (controllerConn).
	if p.pluginCtx.VnetController != nil {
		p.pluginCtx.VnetController.UnregisterClientRoute(p.pluginCtx.Name)
		xl.Infof("Unregistered client route for visitor [%s]", p.pluginCtx.Name)
	} else {
		xl.Warnf("VnetController is nil during close for visitor [%s], cannot unregister route", p.pluginCtx.Name)
	}

	// 3. Explicitly close the controller side of the pipe managed by this plugin.
	// This ensures the pipe is broken even if the run loop is stuck or HandleConn hasn't closed its end.
	p.cleanupControllerConn(xl)
	xl.Infof("Finished cleaning up connections during close for visitor [%s]", p.pluginCtx.Name)

	return nil
}

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

package client

import (
	"context"
	"io"
	"sync"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func init() {
	Register(v1.PluginVirtualNet, NewVirtualNetPlugin)
}

type VirtualNetPlugin struct {
	pluginCtx PluginContext
	opts      *v1.VirtualNetPluginOptions
	mu        sync.Mutex
	conns     map[io.ReadWriteCloser]struct{}
}

func NewVirtualNetPlugin(pluginCtx PluginContext, options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.VirtualNetPluginOptions)

	p := &VirtualNetPlugin{
		pluginCtx: pluginCtx,
		opts:      opts,
	}
	return p, nil
}

func (p *VirtualNetPlugin) Handle(ctx context.Context, connInfo *ConnectionInfo) {
	// Verify if virtual network controller is available
	if p.pluginCtx.VnetController == nil {
		return
	}

	// Add the connection before starting the read loop to avoid race condition
	// where RemoveConn might be called before the connection is added.
	p.mu.Lock()
	if p.conns == nil {
		p.conns = make(map[io.ReadWriteCloser]struct{})
	}
	p.conns[connInfo.Conn] = struct{}{}
	p.mu.Unlock()

	// Register the connection with the controller and pass the cleanup function
	p.pluginCtx.VnetController.StartServerConnReadLoop(ctx, connInfo.Conn, func() {
		p.RemoveConn(connInfo.Conn)
	})
}

func (p *VirtualNetPlugin) RemoveConn(conn io.ReadWriteCloser) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Check if the map exists, as Close might have set it to nil concurrently
	if p.conns != nil {
		delete(p.conns, conn)
	}
}

func (p *VirtualNetPlugin) Name() string {
	return v1.PluginVirtualNet
}

func (p *VirtualNetPlugin) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close any remaining connections
	for conn := range p.conns {
		_ = conn.Close()
	}
	p.conns = nil
	return nil
}

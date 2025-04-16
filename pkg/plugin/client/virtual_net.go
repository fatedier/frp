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

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/xlog"
)

func init() {
	Register(v1.PluginVirtualNet, NewVirtualNetPlugin)
}

type VirtualNetPlugin struct {
	pluginCtx PluginContext
	opts      *v1.VirtualNetPluginOptions
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
	xl := xlog.FromContextSafe(ctx)

	// Verify if virtual network controller is available
	if p.pluginCtx.VnetController == nil {
		return
	}

	// Register the connection with the controller
	routeName := p.pluginCtx.Name
	err := p.pluginCtx.VnetController.RegisterServerConn(ctx, routeName, connInfo.Conn)
	if err != nil {
		xl.Errorf("virtual net failed to register server connection: %v", err)
		return
	}
}

func (p *VirtualNetPlugin) Name() string {
	return v1.PluginVirtualNet
}

func (p *VirtualNetPlugin) Close() error {
	if p.pluginCtx.VnetController != nil {
		p.pluginCtx.VnetController.UnregisterServerConn(p.pluginCtx.Name)
	}
	return nil
}

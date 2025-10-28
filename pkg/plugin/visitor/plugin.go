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

package visitor

import (
	"context"
	"fmt"
	"net"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/vnet"
)

// PluginContext provides the necessary context and callbacks for visitor plugins.
type PluginContext struct {
	// Name is the unique identifier for this visitor, used for logging and routing.
	Name string

	// Ctx manages the plugin's lifecycle and carries the logger for structured logging.
	Ctx context.Context

	// VnetController manages TUN device routing. May be nil if virtual networking is disabled.
	VnetController *vnet.Controller

	// SendConnToVisitor sends a connection to the visitor's internal processing queue.
	// Does not return error; failures are handled by closing the connection.
	SendConnToVisitor func(net.Conn)
}

// Creators is used for create plugins to handle connections.
var creators = make(map[string]CreatorFn)

type CreatorFn func(pluginCtx PluginContext, options v1.VisitorPluginOptions) (Plugin, error)

func Register(name string, fn CreatorFn) {
	if _, exist := creators[name]; exist {
		panic(fmt.Sprintf("plugin [%s] is already registered", name))
	}
	creators[name] = fn
}

func Create(pluginName string, pluginCtx PluginContext, options v1.VisitorPluginOptions) (p Plugin, err error) {
	if fn, ok := creators[pluginName]; ok {
		p, err = fn(pluginCtx, options)
	} else {
		err = fmt.Errorf("plugin [%s] is not registered", pluginName)
	}
	return
}

type Plugin interface {
	Name() string
	Start()
	Close() error
}

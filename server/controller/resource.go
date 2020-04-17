// Copyright 2019 fatedier, fatedier@gmail.com
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

package controller

import (
	"github.com/fatedier/frp/models/nathole"
	plugin "github.com/fatedier/frp/models/plugin/server"
	"github.com/fatedier/frp/server/group"
	"github.com/fatedier/frp/server/ports"
	"github.com/fatedier/frp/utils/tcpmux"
	"github.com/fatedier/frp/utils/vhost"
)

// All resource managers and controllers
type ResourceController struct {
	// Manage all visitor listeners
	VisitorManager *VisitorManager

	// Tcp Group Controller
	TcpGroupCtl *group.TcpGroupCtl

	// HTTP Group Controller
	HTTPGroupCtl *group.HTTPGroupController

	// TCP Mux Group Controller
	TcpMuxGroupCtl *group.TcpMuxGroupCtl

	// Manage all tcp ports
	TcpPortManager *ports.PortManager

	// Manage all udp ports
	UdpPortManager *ports.PortManager

	// For http proxies, forwarding http requests
	HttpReverseProxy *vhost.HttpReverseProxy

	// For https proxies, route requests to different clients by hostname and other information
	VhostHttpsMuxer *vhost.HttpsMuxer

	// Controller for nat hole connections
	NatHoleController *nathole.NatHoleController

	// TcpMux HTTP CONNECT multiplexer
	TcpMuxHttpConnectMuxer *tcpmux.HttpConnectTcpMuxer

	// All server manager plugin
	PluginManager *plugin.Manager
}

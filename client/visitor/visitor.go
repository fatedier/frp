// Copyright 2017 fatedier, fatedier@gmail.com
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
	"net"
	"sync"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/transport"
	utilnet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
)

// Helper wrapps some functions for visitor to use.
type Helper interface {
	// ConnectServer directly connects to the frp server.
	ConnectServer() (net.Conn, error)
	// TransferConn transfers the connection to another visitor.
	TransferConn(string, net.Conn) error
	// MsgTransporter returns the message transporter that is used to send and receive messages
	// to the frp server through the controller.
	MsgTransporter() transport.MessageTransporter
	// RunID returns the run id of current controller.
	RunID() string
}

// Visitor is used for forward traffics from local port tot remote service.
type Visitor interface {
	Run() error
	AcceptConn(conn net.Conn) error
	Close()
}

func NewVisitor(
	ctx context.Context,
	cfg config.VisitorConf,
	clientCfg config.ClientCommonConf,
	helper Helper,
) (visitor Visitor) {
	xl := xlog.FromContextSafe(ctx).Spawn().AppendPrefix(cfg.GetBaseConfig().ProxyName)
	baseVisitor := BaseVisitor{
		clientCfg:  clientCfg,
		helper:     helper,
		ctx:        xlog.NewContext(ctx, xl),
		internalLn: utilnet.NewInternalListener(),
	}
	switch cfg := cfg.(type) {
	case *config.STCPVisitorConf:
		visitor = &STCPVisitor{
			BaseVisitor: &baseVisitor,
			cfg:         cfg,
		}
	case *config.XTCPVisitorConf:
		visitor = &XTCPVisitor{
			BaseVisitor:   &baseVisitor,
			cfg:           cfg,
			startTunnelCh: make(chan struct{}),
		}
	case *config.SUDPVisitorConf:
		visitor = &SUDPVisitor{
			BaseVisitor:  &baseVisitor,
			cfg:          cfg,
			checkCloseCh: make(chan struct{}),
		}
	}
	return
}

type BaseVisitor struct {
	clientCfg  config.ClientCommonConf
	helper     Helper
	l          net.Listener
	internalLn *utilnet.InternalListener

	mu  sync.RWMutex
	ctx context.Context
}

func (v *BaseVisitor) AcceptConn(conn net.Conn) error {
	return v.internalLn.PutConn(conn)
}

func (v *BaseVisitor) Close() {
	if v.l != nil {
		v.l.Close()
	}
	if v.internalLn != nil {
		v.internalLn.Close()
	}
}

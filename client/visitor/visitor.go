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
	"github.com/fatedier/frp/pkg/util/xlog"
)

// Visitor is used for forward traffics from local port tot remote service.
type Visitor interface {
	Run() error
	Close()
}

func NewVisitor(
	ctx context.Context,
	cfg config.VisitorConf,
	clientCfg config.ClientCommonConf,
	connectServer func() (net.Conn, error),
	msgTransporter transport.MessageTransporter,
) (visitor Visitor) {
	xl := xlog.FromContextSafe(ctx).Spawn().AppendPrefix(cfg.GetBaseInfo().ProxyName)
	baseVisitor := BaseVisitor{
		clientCfg:      clientCfg,
		connectServer:  connectServer,
		msgTransporter: msgTransporter,
		ctx:            xlog.NewContext(ctx, xl),
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
	clientCfg      config.ClientCommonConf
	connectServer  func() (net.Conn, error)
	msgTransporter transport.MessageTransporter
	l              net.Listener

	mu  sync.RWMutex
	ctx context.Context
}

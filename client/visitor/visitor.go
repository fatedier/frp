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
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	libio "github.com/fatedier/golib/io"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/naming"
	plugin "github.com/fatedier/frp/pkg/plugin/visitor"
	"github.com/fatedier/frp/pkg/transport"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/pkg/vnet"
)

// Helper wraps some functions for visitor to use.
type Helper interface {
	// ConnectServer directly connects to the frp server.
	ConnectServer() (net.Conn, error)
	// TransferConn transfers the connection to another visitor.
	TransferConn(string, net.Conn) error
	// MsgTransporter returns the message transporter that is used to send and receive messages
	// to the frp server through the controller.
	MsgTransporter() transport.MessageTransporter
	// VNetController returns the vnet controller that is used to manage the virtual network.
	VNetController() *vnet.Controller
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
	cfg v1.VisitorConfigurer,
	clientCfg *v1.ClientCommonConfig,
	helper Helper,
) (Visitor, error) {
	xl := xlog.FromContextSafe(ctx).Spawn().AppendPrefix(cfg.GetBaseConfig().Name)
	ctx = xlog.NewContext(ctx, xl)
	var visitor Visitor
	baseVisitor := BaseVisitor{
		clientCfg:  clientCfg,
		helper:     helper,
		ctx:        ctx,
		internalLn: netpkg.NewInternalListener(),
	}
	if cfg.GetBaseConfig().Plugin.Type != "" {
		p, err := plugin.Create(
			cfg.GetBaseConfig().Plugin.Type,
			plugin.PluginContext{
				Name:           cfg.GetBaseConfig().Name,
				Ctx:            ctx,
				VnetController: helper.VNetController(),
				SendConnToVisitor: func(conn net.Conn) {
					_ = baseVisitor.AcceptConn(conn)
				},
			},
			cfg.GetBaseConfig().Plugin.VisitorPluginOptions,
		)
		if err != nil {
			return nil, err
		}
		baseVisitor.plugin = p
	}
	switch cfg := cfg.(type) {
	case *v1.STCPVisitorConfig:
		visitor = &STCPVisitor{
			BaseVisitor: &baseVisitor,
			cfg:         cfg,
		}
	case *v1.XTCPVisitorConfig:
		visitor = &XTCPVisitor{
			BaseVisitor:   &baseVisitor,
			cfg:           cfg,
			startTunnelCh: make(chan struct{}),
		}
	case *v1.SUDPVisitorConfig:
		visitor = &SUDPVisitor{
			BaseVisitor:  &baseVisitor,
			cfg:          cfg,
			checkCloseCh: make(chan struct{}),
		}
	}
	return visitor, nil
}

type BaseVisitor struct {
	clientCfg  *v1.ClientCommonConfig
	helper     Helper
	l          net.Listener
	internalLn *netpkg.InternalListener
	plugin     plugin.Plugin

	mu  sync.RWMutex
	ctx context.Context
}

func (v *BaseVisitor) AcceptConn(conn net.Conn) error {
	return v.internalLn.PutConn(conn)
}

func (v *BaseVisitor) acceptLoop(l net.Listener, name string, handleConn func(net.Conn)) {
	xl := xlog.FromContextSafe(v.ctx)
	for {
		conn, err := l.Accept()
		if err != nil {
			xl.Warnf("%s listener closed", name)
			return
		}
		go handleConn(conn)
	}
}

func (v *BaseVisitor) Close() {
	if v.l != nil {
		v.l.Close()
	}
	if v.internalLn != nil {
		v.internalLn.Close()
	}
	if v.plugin != nil {
		v.plugin.Close()
	}
}

func (v *BaseVisitor) dialRawVisitorConn(cfg *v1.VisitorBaseConfig) (net.Conn, error) {
	visitorConn, err := v.helper.ConnectServer()
	if err != nil {
		return nil, fmt.Errorf("connect to server error: %v", err)
	}

	now := time.Now().Unix()
	targetProxyName := naming.BuildTargetServerProxyName(v.clientCfg.User, cfg.ServerUser, cfg.ServerName)
	newVisitorConnMsg := &msg.NewVisitorConn{
		RunID:          v.helper.RunID(),
		ProxyName:      targetProxyName,
		SignKey:        util.GetAuthKey(cfg.SecretKey, now),
		Timestamp:      now,
		UseEncryption:  cfg.Transport.UseEncryption,
		UseCompression: cfg.Transport.UseCompression,
	}
	err = msg.WriteMsg(visitorConn, newVisitorConnMsg)
	if err != nil {
		visitorConn.Close()
		return nil, fmt.Errorf("send newVisitorConnMsg to server error: %v", err)
	}

	var newVisitorConnRespMsg msg.NewVisitorConnResp
	_ = visitorConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	err = msg.ReadMsgInto(visitorConn, &newVisitorConnRespMsg)
	if err != nil {
		visitorConn.Close()
		return nil, fmt.Errorf("read newVisitorConnRespMsg error: %v", err)
	}
	_ = visitorConn.SetReadDeadline(time.Time{})

	if newVisitorConnRespMsg.Error != "" {
		visitorConn.Close()
		return nil, fmt.Errorf("start new visitor connection error: %s", newVisitorConnRespMsg.Error)
	}
	return visitorConn, nil
}

func wrapVisitorConn(conn io.ReadWriteCloser, cfg *v1.VisitorBaseConfig) (io.ReadWriteCloser, func(), error) {
	rwc := conn
	if cfg.Transport.UseEncryption {
		var err error
		rwc, err = libio.WithEncryption(rwc, []byte(cfg.SecretKey))
		if err != nil {
			return nil, func() {}, fmt.Errorf("create encryption stream error: %v", err)
		}
	}
	recycleFn := func() {}
	if cfg.Transport.UseCompression {
		rwc, recycleFn = libio.WithCompressionFromPool(rwc)
	}
	return rwc, recycleFn, nil
}

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

package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	libio "github.com/fatedier/golib/io"
	libnet "github.com/fatedier/golib/net"
	"golang.org/x/time/rate"

	"github.com/fatedier/frp/pkg/config/types"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	plugin "github.com/fatedier/frp/pkg/plugin/client"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/limit"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/pkg/vnet"
)

var proxyFactoryRegistry = map[reflect.Type]func(*BaseProxy, v1.ProxyConfigurer) Proxy{}

func RegisterProxyFactory(proxyConfType reflect.Type, factory func(*BaseProxy, v1.ProxyConfigurer) Proxy) {
	proxyFactoryRegistry[proxyConfType] = factory
}

// Proxy defines how to handle work connections for different proxy type.
type Proxy interface {
	Run() error
	// InWorkConn accept work connections registered to server.
	InWorkConn(net.Conn, *msg.StartWorkConn)
	SetInWorkConnCallback(func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) /* continue */ bool)
	Close()
}

func NewProxy(
	ctx context.Context,
	pxyConf v1.ProxyConfigurer,
	clientCfg *v1.ClientCommonConfig,
	encryptionKey []byte,
	msgTransporter transport.MessageTransporter,
	vnetController *vnet.Controller,
) (pxy Proxy) {
	var limiter *rate.Limiter
	limitBytes := pxyConf.GetBaseConfig().Transport.BandwidthLimit.Bytes()
	if limitBytes > 0 && pxyConf.GetBaseConfig().Transport.BandwidthLimitMode == types.BandwidthLimitModeClient {
		limiter = rate.NewLimiter(rate.Limit(float64(limitBytes)), int(limitBytes))
	}

	baseProxy := BaseProxy{
		baseCfg:        pxyConf.GetBaseConfig(),
		clientCfg:      clientCfg,
		encryptionKey:  encryptionKey,
		limiter:        limiter,
		msgTransporter: msgTransporter,
		vnetController: vnetController,
		xl:             xlog.FromContextSafe(ctx),
		ctx:            ctx,
	}

	factory := proxyFactoryRegistry[reflect.TypeOf(pxyConf)]
	if factory == nil {
		return nil
	}
	return factory(&baseProxy, pxyConf)
}

type BaseProxy struct {
	baseCfg        *v1.ProxyBaseConfig
	clientCfg      *v1.ClientCommonConfig
	encryptionKey  []byte
	msgTransporter transport.MessageTransporter
	vnetController *vnet.Controller
	limiter        *rate.Limiter
	// proxyPlugin is used to handle connections instead of dialing to local service.
	// It's only validate for TCP protocol now.
	proxyPlugin        plugin.Plugin
	inWorkConnCallback func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) /* continue */ bool

	mu  sync.RWMutex
	xl  *xlog.Logger
	ctx context.Context
}

func (pxy *BaseProxy) Run() error {
	if pxy.baseCfg.Plugin.Type != "" {
		p, err := plugin.Create(pxy.baseCfg.Plugin.Type, plugin.PluginContext{
			Name:           pxy.baseCfg.Name,
			VnetController: pxy.vnetController,
		}, pxy.baseCfg.Plugin.ClientPluginOptions)
		if err != nil {
			return err
		}
		pxy.proxyPlugin = p
	}
	return nil
}

func (pxy *BaseProxy) Close() {
	if pxy.proxyPlugin != nil {
		pxy.proxyPlugin.Close()
	}
}

// wrapWorkConn applies rate limiting, encryption, and compression
// to a work connection based on the proxy's transport configuration.
// The returned recycle function should be called when the stream is no longer in use
// to return compression resources to the pool. It is safe to not call recycle,
// in which case resources will be garbage collected normally.
func (pxy *BaseProxy) wrapWorkConn(conn net.Conn, encKey []byte) (io.ReadWriteCloser, func(), error) {
	var rwc io.ReadWriteCloser = conn
	if pxy.limiter != nil {
		rwc = libio.WrapReadWriteCloser(limit.NewReader(conn, pxy.limiter), limit.NewWriter(conn, pxy.limiter), func() error {
			return conn.Close()
		})
	}
	if pxy.baseCfg.Transport.UseEncryption {
		var err error
		rwc, err = libio.WithEncryption(rwc, encKey)
		if err != nil {
			conn.Close()
			return nil, nil, fmt.Errorf("create encryption stream error: %w", err)
		}
	}
	var recycleFn func()
	if pxy.baseCfg.Transport.UseCompression {
		rwc, recycleFn = libio.WithCompressionFromPool(rwc)
	}
	return rwc, recycleFn, nil
}

func (pxy *BaseProxy) SetInWorkConnCallback(cb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool) {
	pxy.inWorkConnCallback = cb
}

func (pxy *BaseProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	if pxy.inWorkConnCallback != nil {
		if !pxy.inWorkConnCallback(pxy.baseCfg, conn, m) {
			return
		}
	}
	pxy.HandleTCPWorkConnection(conn, m, pxy.encryptionKey)
}

// Common handler for tcp work connections.
func (pxy *BaseProxy) HandleTCPWorkConnection(workConn net.Conn, m *msg.StartWorkConn, encKey []byte) {
	xl := pxy.xl
	baseCfg := pxy.baseCfg

	xl.Tracef("handle tcp work connection, useEncryption: %t, useCompression: %t",
		baseCfg.Transport.UseEncryption, baseCfg.Transport.UseCompression)

	remote, recycleFn, err := pxy.wrapWorkConn(workConn, encKey)
	if err != nil {
		xl.Errorf("wrap work connection: %v", err)
		return
	}

	// check if we need to send proxy protocol info
	var connInfo plugin.ConnectionInfo
	if m.SrcAddr != "" && m.SrcPort != 0 {
		if m.DstAddr == "" {
			m.DstAddr = "127.0.0.1"
		}
		srcAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(m.SrcAddr, strconv.Itoa(int(m.SrcPort))))
		dstAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(m.DstAddr, strconv.Itoa(int(m.DstPort))))
		connInfo.SrcAddr = srcAddr
		connInfo.DstAddr = dstAddr
	}

	if baseCfg.Transport.ProxyProtocolVersion != "" && m.SrcAddr != "" && m.SrcPort != 0 {
		header := netpkg.BuildProxyProtocolHeaderStruct(connInfo.SrcAddr, connInfo.DstAddr, baseCfg.Transport.ProxyProtocolVersion)
		connInfo.ProxyProtocolHeader = header
	}
	connInfo.Conn = remote
	connInfo.UnderlyingConn = workConn

	if pxy.proxyPlugin != nil {
		// if plugin is set, let plugin handle connection first
		// Don't recycle compression resources here because plugins may
		// retain the connection after Handle returns.
		xl.Debugf("handle by plugin: %s", pxy.proxyPlugin.Name())
		pxy.proxyPlugin.Handle(pxy.ctx, &connInfo)
		xl.Debugf("handle by plugin finished")
		return
	}

	if recycleFn != nil {
		defer recycleFn()
	}

	localConn, err := libnet.Dial(
		net.JoinHostPort(baseCfg.LocalIP, strconv.Itoa(baseCfg.LocalPort)),
		libnet.WithTimeout(10*time.Second),
	)
	if err != nil {
		workConn.Close()
		xl.Errorf("connect to local service [%s:%d] error: %v", baseCfg.LocalIP, baseCfg.LocalPort, err)
		return
	}

	xl.Debugf("join connections, localConn(l[%s] r[%s]) workConn(l[%s] r[%s])", localConn.LocalAddr().String(),
		localConn.RemoteAddr().String(), workConn.LocalAddr().String(), workConn.RemoteAddr().String())

	if connInfo.ProxyProtocolHeader != nil {
		if _, err := connInfo.ProxyProtocolHeader.WriteTo(localConn); err != nil {
			workConn.Close()
			localConn.Close()
			xl.Errorf("write proxy protocol header to local conn error: %v", err)
			return
		}
	}

	_, _, errs := libio.Join(localConn, remote)
	xl.Debugf("join connections closed")
	if len(errs) > 0 {
		xl.Tracef("join connections errors: %v", errs)
	}
}

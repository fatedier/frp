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
	"bytes"
	"context"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	libio "github.com/fatedier/golib/io"
	libdial "github.com/fatedier/golib/net/dial"
	pp "github.com/pires/go-proxyproto"
	"golang.org/x/time/rate"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/msg"
	plugin "github.com/fatedier/frp/pkg/plugin/client"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/limit"
	"github.com/fatedier/frp/pkg/util/xlog"
)

var proxyFactoryRegistry = map[reflect.Type]func(*BaseProxy, config.ProxyConf) Proxy{}

func RegisterProxyFactory(proxyConfType reflect.Type, factory func(*BaseProxy, config.ProxyConf) Proxy) {
	proxyFactoryRegistry[proxyConfType] = factory
}

// Proxy defines how to handle work connections for different proxy type.
type Proxy interface {
	Run() error

	// InWorkConn accept work connections registered to server.
	InWorkConn(net.Conn, *msg.StartWorkConn)

	Close()
}

func NewProxy(
	ctx context.Context,
	pxyConf config.ProxyConf,
	clientCfg config.ClientCommonConf,
	msgTransporter transport.MessageTransporter,
) (pxy Proxy) {
	var limiter *rate.Limiter
	limitBytes := pxyConf.GetBaseConfig().BandwidthLimit.Bytes()
	if limitBytes > 0 && pxyConf.GetBaseConfig().BandwidthLimitMode == config.BandwidthLimitModeClient {
		limiter = rate.NewLimiter(rate.Limit(float64(limitBytes)), int(limitBytes))
	}

	baseProxy := BaseProxy{
		baseProxyConfig: pxyConf.GetBaseConfig(),
		clientCfg:       clientCfg,
		limiter:         limiter,
		msgTransporter:  msgTransporter,
		xl:              xlog.FromContextSafe(ctx),
		ctx:             ctx,
	}

	factory := proxyFactoryRegistry[reflect.TypeOf(pxyConf)]
	if factory == nil {
		return nil
	}
	return factory(&baseProxy, pxyConf)
}

type BaseProxy struct {
	baseProxyConfig *config.BaseProxyConf
	clientCfg       config.ClientCommonConf
	msgTransporter  transport.MessageTransporter
	limiter         *rate.Limiter
	// proxyPlugin is used to handle connections instead of dialing to local service.
	// It's only validate for TCP protocol now.
	proxyPlugin plugin.Plugin

	mu  sync.RWMutex
	xl  *xlog.Logger
	ctx context.Context
}

func (pxy *BaseProxy) Run() error {
	if pxy.baseProxyConfig.Plugin != "" {
		p, err := plugin.Create(pxy.baseProxyConfig.Plugin, pxy.baseProxyConfig.PluginParams)
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

func (pxy *BaseProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	pxy.HandleTCPWorkConnection(conn, m, []byte(pxy.clientCfg.Token))
}

// Common handler for tcp work connections.
func (pxy *BaseProxy) HandleTCPWorkConnection(workConn net.Conn, m *msg.StartWorkConn, encKey []byte) {
	xl := pxy.xl
	baseConfig := pxy.baseProxyConfig
	var (
		remote io.ReadWriteCloser
		err    error
	)
	remote = workConn
	if pxy.limiter != nil {
		remote = libio.WrapReadWriteCloser(limit.NewReader(workConn, pxy.limiter), limit.NewWriter(workConn, pxy.limiter), func() error {
			return workConn.Close()
		})
	}

	xl.Trace("handle tcp work connection, use_encryption: %t, use_compression: %t",
		baseConfig.UseEncryption, baseConfig.UseCompression)
	if baseConfig.UseEncryption {
		remote, err = libio.WithEncryption(remote, encKey)
		if err != nil {
			workConn.Close()
			xl.Error("create encryption stream error: %v", err)
			return
		}
	}
	if baseConfig.UseCompression {
		remote = libio.WithCompression(remote)
	}

	// check if we need to send proxy protocol info
	var extraInfo []byte
	if baseConfig.ProxyProtocolVersion != "" {
		if m.SrcAddr != "" && m.SrcPort != 0 {
			if m.DstAddr == "" {
				m.DstAddr = "127.0.0.1"
			}
			srcAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(m.SrcAddr, strconv.Itoa(int(m.SrcPort))))
			dstAddr, _ := net.ResolveTCPAddr("tcp", net.JoinHostPort(m.DstAddr, strconv.Itoa(int(m.DstPort))))
			h := &pp.Header{
				Command:         pp.PROXY,
				SourceAddr:      srcAddr,
				DestinationAddr: dstAddr,
			}

			if strings.Contains(m.SrcAddr, ".") {
				h.TransportProtocol = pp.TCPv4
			} else {
				h.TransportProtocol = pp.TCPv6
			}

			if baseConfig.ProxyProtocolVersion == "v1" {
				h.Version = 1
			} else if baseConfig.ProxyProtocolVersion == "v2" {
				h.Version = 2
			}

			buf := bytes.NewBuffer(nil)
			_, _ = h.WriteTo(buf)
			extraInfo = buf.Bytes()
		}
	}

	if pxy.proxyPlugin != nil {
		// if plugin is set, let plugin handle connection first
		xl.Debug("handle by plugin: %s", pxy.proxyPlugin.Name())
		pxy.proxyPlugin.Handle(remote, workConn, extraInfo)
		xl.Debug("handle by plugin finished")
		return
	}

	localConn, err := libdial.Dial(
		net.JoinHostPort(baseConfig.LocalIP, strconv.Itoa(baseConfig.LocalPort)),
		libdial.WithTimeout(10*time.Second),
	)
	if err != nil {
		workConn.Close()
		xl.Error("connect to local service [%s:%d] error: %v", baseConfig.LocalIP, baseConfig.LocalPort, err)
		return
	}

	xl.Debug("join connections, localConn(l[%s] r[%s]) workConn(l[%s] r[%s])", localConn.LocalAddr().String(),
		localConn.RemoteAddr().String(), workConn.LocalAddr().String(), workConn.RemoteAddr().String())

	if len(extraInfo) > 0 {
		if _, err := localConn.Write(extraInfo); err != nil {
			workConn.Close()
			xl.Error("write extraInfo to local conn error: %v", err)
			return
		}
	}

	_, _, errs := libio.Join(localConn, remote)
	xl.Debug("join connections closed")
	if len(errs) > 0 {
		xl.Trace("join connections errors: %v", errs)
	}
}

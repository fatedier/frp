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
	"strconv"
	"strings"
	"sync"
	"time"

	frpIo "github.com/fatedier/golib/io"
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
	limitBytes := pxyConf.GetBaseInfo().BandwidthLimit.Bytes()
	if limitBytes > 0 && pxyConf.GetBaseInfo().BandwidthLimitMode == config.BandwidthLimitModeClient {
		limiter = rate.NewLimiter(rate.Limit(float64(limitBytes)), int(limitBytes))
	}

	baseProxy := BaseProxy{
		clientCfg:      clientCfg,
		limiter:        limiter,
		msgTransporter: msgTransporter,
		xl:             xlog.FromContextSafe(ctx),
		ctx:            ctx,
	}
	switch cfg := pxyConf.(type) {
	case *config.TCPProxyConf:
		pxy = &TCPProxy{
			BaseProxy: &baseProxy,
			cfg:       cfg,
		}
	case *config.TCPMuxProxyConf:
		pxy = &TCPMuxProxy{
			BaseProxy: &baseProxy,
			cfg:       cfg,
		}
	case *config.UDPProxyConf:
		pxy = &UDPProxy{
			BaseProxy: &baseProxy,
			cfg:       cfg,
		}
	case *config.HTTPProxyConf:
		pxy = &HTTPProxy{
			BaseProxy: &baseProxy,
			cfg:       cfg,
		}
	case *config.HTTPSProxyConf:
		pxy = &HTTPSProxy{
			BaseProxy: &baseProxy,
			cfg:       cfg,
		}
	case *config.STCPProxyConf:
		pxy = &STCPProxy{
			BaseProxy: &baseProxy,
			cfg:       cfg,
		}
	case *config.XTCPProxyConf:
		pxy = &XTCPProxy{
			BaseProxy: &baseProxy,
			cfg:       cfg,
		}
	case *config.SUDPProxyConf:
		pxy = &SUDPProxy{
			BaseProxy: &baseProxy,
			cfg:       cfg,
			closeCh:   make(chan struct{}),
		}
	}
	return
}

type BaseProxy struct {
	closed         bool
	clientCfg      config.ClientCommonConf
	msgTransporter transport.MessageTransporter
	limiter        *rate.Limiter

	mu  sync.RWMutex
	xl  *xlog.Logger
	ctx context.Context
}

// TCP
type TCPProxy struct {
	*BaseProxy

	cfg         *config.TCPProxyConf
	proxyPlugin plugin.Plugin
}

func (pxy *TCPProxy) Run() (err error) {
	if pxy.cfg.Plugin != "" {
		pxy.proxyPlugin, err = plugin.Create(pxy.cfg.Plugin, pxy.cfg.PluginParams)
		if err != nil {
			return
		}
	}
	return
}

func (pxy *TCPProxy) Close() {
	if pxy.proxyPlugin != nil {
		pxy.proxyPlugin.Close()
	}
}

func (pxy *TCPProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	HandleTCPWorkConnection(pxy.ctx, &pxy.cfg.LocalSvrConf, pxy.proxyPlugin, pxy.cfg.GetBaseInfo(), pxy.limiter,
		conn, []byte(pxy.clientCfg.Token), m)
}

// TCP Multiplexer
type TCPMuxProxy struct {
	*BaseProxy

	cfg         *config.TCPMuxProxyConf
	proxyPlugin plugin.Plugin
}

func (pxy *TCPMuxProxy) Run() (err error) {
	if pxy.cfg.Plugin != "" {
		pxy.proxyPlugin, err = plugin.Create(pxy.cfg.Plugin, pxy.cfg.PluginParams)
		if err != nil {
			return
		}
	}
	return
}

func (pxy *TCPMuxProxy) Close() {
	if pxy.proxyPlugin != nil {
		pxy.proxyPlugin.Close()
	}
}

func (pxy *TCPMuxProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	HandleTCPWorkConnection(pxy.ctx, &pxy.cfg.LocalSvrConf, pxy.proxyPlugin, pxy.cfg.GetBaseInfo(), pxy.limiter,
		conn, []byte(pxy.clientCfg.Token), m)
}

// HTTP
type HTTPProxy struct {
	*BaseProxy

	cfg         *config.HTTPProxyConf
	proxyPlugin plugin.Plugin
}

func (pxy *HTTPProxy) Run() (err error) {
	if pxy.cfg.Plugin != "" {
		pxy.proxyPlugin, err = plugin.Create(pxy.cfg.Plugin, pxy.cfg.PluginParams)
		if err != nil {
			return
		}
	}
	return
}

func (pxy *HTTPProxy) Close() {
	if pxy.proxyPlugin != nil {
		pxy.proxyPlugin.Close()
	}
}

func (pxy *HTTPProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	HandleTCPWorkConnection(pxy.ctx, &pxy.cfg.LocalSvrConf, pxy.proxyPlugin, pxy.cfg.GetBaseInfo(), pxy.limiter,
		conn, []byte(pxy.clientCfg.Token), m)
}

// HTTPS
type HTTPSProxy struct {
	*BaseProxy

	cfg         *config.HTTPSProxyConf
	proxyPlugin plugin.Plugin
}

func (pxy *HTTPSProxy) Run() (err error) {
	if pxy.cfg.Plugin != "" {
		pxy.proxyPlugin, err = plugin.Create(pxy.cfg.Plugin, pxy.cfg.PluginParams)
		if err != nil {
			return
		}
	}
	return
}

func (pxy *HTTPSProxy) Close() {
	if pxy.proxyPlugin != nil {
		pxy.proxyPlugin.Close()
	}
}

func (pxy *HTTPSProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	HandleTCPWorkConnection(pxy.ctx, &pxy.cfg.LocalSvrConf, pxy.proxyPlugin, pxy.cfg.GetBaseInfo(), pxy.limiter,
		conn, []byte(pxy.clientCfg.Token), m)
}

// STCP
type STCPProxy struct {
	*BaseProxy

	cfg         *config.STCPProxyConf
	proxyPlugin plugin.Plugin
}

func (pxy *STCPProxy) Run() (err error) {
	if pxy.cfg.Plugin != "" {
		pxy.proxyPlugin, err = plugin.Create(pxy.cfg.Plugin, pxy.cfg.PluginParams)
		if err != nil {
			return
		}
	}
	return
}

func (pxy *STCPProxy) Close() {
	if pxy.proxyPlugin != nil {
		pxy.proxyPlugin.Close()
	}
}

func (pxy *STCPProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	HandleTCPWorkConnection(pxy.ctx, &pxy.cfg.LocalSvrConf, pxy.proxyPlugin, pxy.cfg.GetBaseInfo(), pxy.limiter,
		conn, []byte(pxy.clientCfg.Token), m)
}

// Common handler for tcp work connections.
func HandleTCPWorkConnection(ctx context.Context, localInfo *config.LocalSvrConf, proxyPlugin plugin.Plugin,
	baseInfo *config.BaseProxyConf, limiter *rate.Limiter, workConn net.Conn, encKey []byte, m *msg.StartWorkConn,
) {
	xl := xlog.FromContextSafe(ctx)
	var (
		remote io.ReadWriteCloser
		err    error
	)
	remote = workConn
	if limiter != nil {
		remote = frpIo.WrapReadWriteCloser(limit.NewReader(workConn, limiter), limit.NewWriter(workConn, limiter), func() error {
			return workConn.Close()
		})
	}

	xl.Trace("handle tcp work connection, use_encryption: %t, use_compression: %t",
		baseInfo.UseEncryption, baseInfo.UseCompression)
	if baseInfo.UseEncryption {
		remote, err = frpIo.WithEncryption(remote, encKey)
		if err != nil {
			workConn.Close()
			xl.Error("create encryption stream error: %v", err)
			return
		}
	}
	if baseInfo.UseCompression {
		remote = frpIo.WithCompression(remote)
	}

	// check if we need to send proxy protocol info
	var extraInfo []byte
	if baseInfo.ProxyProtocolVersion != "" {
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

			if baseInfo.ProxyProtocolVersion == "v1" {
				h.Version = 1
			} else if baseInfo.ProxyProtocolVersion == "v2" {
				h.Version = 2
			}

			buf := bytes.NewBuffer(nil)
			_, _ = h.WriteTo(buf)
			extraInfo = buf.Bytes()
		}
	}

	if proxyPlugin != nil {
		// if plugin is set, let plugin handle connections first
		xl.Debug("handle by plugin: %s", proxyPlugin.Name())
		proxyPlugin.Handle(remote, workConn, extraInfo)
		xl.Debug("handle by plugin finished")
		return
	}

	localConn, err := libdial.Dial(
		net.JoinHostPort(localInfo.LocalIP, strconv.Itoa(localInfo.LocalPort)),
		libdial.WithTimeout(10*time.Second),
	)
	if err != nil {
		workConn.Close()
		xl.Error("connect to local service [%s:%d] error: %v", localInfo.LocalIP, localInfo.LocalPort, err)
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

	_, _, errs := frpIo.Join(localConn, remote)
	xl.Debug("join connections closed")
	if len(errs) > 0 {
		xl.Trace("join connections errors: %v", errs)
	}
}

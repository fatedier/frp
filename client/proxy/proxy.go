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
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/msg"
	plugin "github.com/fatedier/frp/pkg/plugin/client"
	"github.com/fatedier/frp/pkg/proto/udp"
	"github.com/fatedier/frp/pkg/util/limit"
	frpNet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"

	"github.com/fatedier/golib/errors"
	frpIo "github.com/fatedier/golib/io"
	libdial "github.com/fatedier/golib/net/dial"
	"github.com/fatedier/golib/pool"
	fmux "github.com/hashicorp/yamux"
	pp "github.com/pires/go-proxyproto"
	"golang.org/x/time/rate"
)

// Proxy defines how to handle work connections for different proxy type.
type Proxy interface {
	Run() error

	// InWorkConn accept work connections registered to server.
	InWorkConn(net.Conn, *msg.StartWorkConn)

	Close()
}

func NewProxy(ctx context.Context, pxyConf config.ProxyConf, clientCfg config.ClientCommonConf, serverUDPPort int) (pxy Proxy) {
	var limiter *rate.Limiter
	limitBytes := pxyConf.GetBaseInfo().BandwidthLimit.Bytes()
	if limitBytes > 0 {
		limiter = rate.NewLimiter(rate.Limit(float64(limitBytes)), int(limitBytes))
	}

	baseProxy := BaseProxy{
		clientCfg:     clientCfg,
		serverUDPPort: serverUDPPort,
		limiter:       limiter,
		xl:            xlog.FromContextSafe(ctx),
		ctx:           ctx,
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
	closed        bool
	clientCfg     config.ClientCommonConf
	serverUDPPort int
	limiter       *rate.Limiter

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

// XTCP
type XTCPProxy struct {
	*BaseProxy

	cfg         *config.XTCPProxyConf
	proxyPlugin plugin.Plugin
}

func (pxy *XTCPProxy) Run() (err error) {
	if pxy.cfg.Plugin != "" {
		pxy.proxyPlugin, err = plugin.Create(pxy.cfg.Plugin, pxy.cfg.PluginParams)
		if err != nil {
			return
		}
	}
	return
}

func (pxy *XTCPProxy) Close() {
	if pxy.proxyPlugin != nil {
		pxy.proxyPlugin.Close()
	}
}

func (pxy *XTCPProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	xl := pxy.xl
	defer conn.Close()
	var natHoleSidMsg msg.NatHoleSid
	err := msg.ReadMsgInto(conn, &natHoleSidMsg)
	if err != nil {
		xl.Error("xtcp read from workConn error: %v", err)
		return
	}

	natHoleClientMsg := &msg.NatHoleClient{
		ProxyName: pxy.cfg.ProxyName,
		Sid:       natHoleSidMsg.Sid,
	}
	raddr, _ := net.ResolveUDPAddr("udp",
		fmt.Sprintf("%s:%d", pxy.clientCfg.ServerAddr, pxy.serverUDPPort))
	clientConn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		xl.Error("dial server udp addr error: %v", err)
		return
	}
	defer clientConn.Close()

	err = msg.WriteMsg(clientConn, natHoleClientMsg)
	if err != nil {
		xl.Error("send natHoleClientMsg to server error: %v", err)
		return
	}

	// Wait for client address at most 5 seconds.
	var natHoleRespMsg msg.NatHoleResp
	clientConn.SetReadDeadline(time.Now().Add(5 * time.Second))

	buf := pool.GetBuf(1024)
	n, err := clientConn.Read(buf)
	if err != nil {
		xl.Error("get natHoleRespMsg error: %v", err)
		return
	}
	err = msg.ReadMsgInto(bytes.NewReader(buf[:n]), &natHoleRespMsg)
	if err != nil {
		xl.Error("get natHoleRespMsg error: %v", err)
		return
	}
	clientConn.SetReadDeadline(time.Time{})
	clientConn.Close()

	if natHoleRespMsg.Error != "" {
		xl.Error("natHoleRespMsg get error info: %s", natHoleRespMsg.Error)
		return
	}

	xl.Trace("get natHoleRespMsg, sid [%s], client address [%s] visitor address [%s]", natHoleRespMsg.Sid, natHoleRespMsg.ClientAddr, natHoleRespMsg.VisitorAddr)

	// Send detect message
	host, portStr, err := net.SplitHostPort(natHoleRespMsg.VisitorAddr)
	if err != nil {
		xl.Error("get NatHoleResp visitor address [%s] error: %v", natHoleRespMsg.VisitorAddr, err)
	}
	laddr, _ := net.ResolveUDPAddr("udp", clientConn.LocalAddr().String())

	port, err := strconv.ParseInt(portStr, 10, 64)
	if err != nil {
		xl.Error("get natHoleResp visitor address error: %v", natHoleRespMsg.VisitorAddr)
		return
	}
	pxy.sendDetectMsg(host, int(port), laddr, []byte(natHoleRespMsg.Sid))
	xl.Trace("send all detect msg done")

	msg.WriteMsg(conn, &msg.NatHoleClientDetectOK{})

	// Listen for clientConn's address and wait for visitor connection
	lConn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		xl.Error("listen on visitorConn's local address error: %v", err)
		return
	}
	defer lConn.Close()

	lConn.SetReadDeadline(time.Now().Add(8 * time.Second))
	sidBuf := pool.GetBuf(1024)
	var uAddr *net.UDPAddr
	n, uAddr, err = lConn.ReadFromUDP(sidBuf)
	if err != nil {
		xl.Warn("get sid from visitor error: %v", err)
		return
	}
	lConn.SetReadDeadline(time.Time{})
	if string(sidBuf[:n]) != natHoleRespMsg.Sid {
		xl.Warn("incorrect sid from visitor")
		return
	}
	pool.PutBuf(sidBuf)
	xl.Info("nat hole connection make success, sid [%s]", natHoleRespMsg.Sid)

	lConn.WriteToUDP(sidBuf[:n], uAddr)

	kcpConn, err := frpNet.NewKCPConnFromUDP(lConn, false, uAddr.String())
	if err != nil {
		xl.Error("create kcp connection from udp connection error: %v", err)
		return
	}

	fmuxCfg := fmux.DefaultConfig()
	fmuxCfg.KeepAliveInterval = 5 * time.Second
	fmuxCfg.LogOutput = io.Discard
	sess, err := fmux.Server(kcpConn, fmuxCfg)
	if err != nil {
		xl.Error("create yamux server from kcp connection error: %v", err)
		return
	}
	defer sess.Close()
	muxConn, err := sess.Accept()
	if err != nil {
		xl.Error("accept for yamux connection error: %v", err)
		return
	}

	HandleTCPWorkConnection(pxy.ctx, &pxy.cfg.LocalSvrConf, pxy.proxyPlugin, pxy.cfg.GetBaseInfo(), pxy.limiter,
		muxConn, []byte(pxy.cfg.Sk), m)
}

func (pxy *XTCPProxy) sendDetectMsg(addr string, port int, laddr *net.UDPAddr, content []byte) (err error) {
	daddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return err
	}

	tConn, err := net.DialUDP("udp", laddr, daddr)
	if err != nil {
		return err
	}

	//uConn := ipv4.NewConn(tConn)
	//uConn.SetTTL(3)

	tConn.Write(content)
	tConn.Close()
	return nil
}

// UDP
type UDPProxy struct {
	*BaseProxy

	cfg *config.UDPProxyConf

	localAddr *net.UDPAddr
	readCh    chan *msg.UDPPacket

	// include msg.UDPPacket and msg.Ping
	sendCh   chan msg.Message
	workConn net.Conn
}

func (pxy *UDPProxy) Run() (err error) {
	pxy.localAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", pxy.cfg.LocalIP, pxy.cfg.LocalPort))
	if err != nil {
		return
	}
	return
}

func (pxy *UDPProxy) Close() {
	pxy.mu.Lock()
	defer pxy.mu.Unlock()

	if !pxy.closed {
		pxy.closed = true
		if pxy.workConn != nil {
			pxy.workConn.Close()
		}
		if pxy.readCh != nil {
			close(pxy.readCh)
		}
		if pxy.sendCh != nil {
			close(pxy.sendCh)
		}
	}
}

func (pxy *UDPProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	xl := pxy.xl
	xl.Info("incoming a new work connection for udp proxy, %s", conn.RemoteAddr().String())
	// close resources releated with old workConn
	pxy.Close()

	var rwc io.ReadWriteCloser = conn
	var err error
	if pxy.limiter != nil {
		rwc = frpIo.WrapReadWriteCloser(limit.NewReader(conn, pxy.limiter), limit.NewWriter(conn, pxy.limiter), func() error {
			return conn.Close()
		})
	}
	if pxy.cfg.UseEncryption {
		rwc, err = frpIo.WithEncryption(rwc, []byte(pxy.clientCfg.Token))
		if err != nil {
			conn.Close()
			xl.Error("create encryption stream error: %v", err)
			return
		}
	}
	if pxy.cfg.UseCompression {
		rwc = frpIo.WithCompression(rwc)
	}
	conn = frpNet.WrapReadWriteCloserToConn(rwc, conn)

	pxy.mu.Lock()
	pxy.workConn = conn
	pxy.readCh = make(chan *msg.UDPPacket, 1024)
	pxy.sendCh = make(chan msg.Message, 1024)
	pxy.closed = false
	pxy.mu.Unlock()

	workConnReaderFn := func(conn net.Conn, readCh chan *msg.UDPPacket) {
		for {
			var udpMsg msg.UDPPacket
			if errRet := msg.ReadMsgInto(conn, &udpMsg); errRet != nil {
				xl.Warn("read from workConn for udp error: %v", errRet)
				return
			}
			if errRet := errors.PanicToError(func() {
				xl.Trace("get udp package from workConn: %s", udpMsg.Content)
				readCh <- &udpMsg
			}); errRet != nil {
				xl.Info("reader goroutine for udp work connection closed: %v", errRet)
				return
			}
		}
	}
	workConnSenderFn := func(conn net.Conn, sendCh chan msg.Message) {
		defer func() {
			xl.Info("writer goroutine for udp work connection closed")
		}()
		var errRet error
		for rawMsg := range sendCh {
			switch m := rawMsg.(type) {
			case *msg.UDPPacket:
				xl.Trace("send udp package to workConn: %s", m.Content)
			case *msg.Ping:
				xl.Trace("send ping message to udp workConn")
			}
			if errRet = msg.WriteMsg(conn, rawMsg); errRet != nil {
				xl.Error("udp work write error: %v", errRet)
				return
			}
		}
	}
	heartbeatFn := func(conn net.Conn, sendCh chan msg.Message) {
		var errRet error
		for {
			time.Sleep(time.Duration(30) * time.Second)
			if errRet = errors.PanicToError(func() {
				sendCh <- &msg.Ping{}
			}); errRet != nil {
				xl.Trace("heartbeat goroutine for udp work connection closed")
				break
			}
		}
	}

	go workConnSenderFn(pxy.workConn, pxy.sendCh)
	go workConnReaderFn(pxy.workConn, pxy.readCh)
	go heartbeatFn(pxy.workConn, pxy.sendCh)
	udp.Forwarder(pxy.localAddr, pxy.readCh, pxy.sendCh, int(pxy.clientCfg.UDPPacketSize))
}

type SUDPProxy struct {
	*BaseProxy

	cfg *config.SUDPProxyConf

	localAddr *net.UDPAddr

	closeCh chan struct{}
}

func (pxy *SUDPProxy) Run() (err error) {
	pxy.localAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", pxy.cfg.LocalIP, pxy.cfg.LocalPort))
	if err != nil {
		return
	}
	return
}

func (pxy *SUDPProxy) Close() {
	pxy.mu.Lock()
	defer pxy.mu.Unlock()
	select {
	case <-pxy.closeCh:
		return
	default:
		close(pxy.closeCh)
	}
}

func (pxy *SUDPProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	xl := pxy.xl
	xl.Info("incoming a new work connection for sudp proxy, %s", conn.RemoteAddr().String())

	var rwc io.ReadWriteCloser = conn
	var err error
	if pxy.limiter != nil {
		rwc = frpIo.WrapReadWriteCloser(limit.NewReader(conn, pxy.limiter), limit.NewWriter(conn, pxy.limiter), func() error {
			return conn.Close()
		})
	}
	if pxy.cfg.UseEncryption {
		rwc, err = frpIo.WithEncryption(rwc, []byte(pxy.clientCfg.Token))
		if err != nil {
			conn.Close()
			xl.Error("create encryption stream error: %v", err)
			return
		}
	}
	if pxy.cfg.UseCompression {
		rwc = frpIo.WithCompression(rwc)
	}
	conn = frpNet.WrapReadWriteCloserToConn(rwc, conn)

	workConn := conn
	readCh := make(chan *msg.UDPPacket, 1024)
	sendCh := make(chan msg.Message, 1024)
	isClose := false

	mu := &sync.Mutex{}

	closeFn := func() {
		mu.Lock()
		defer mu.Unlock()
		if isClose {
			return
		}

		isClose = true
		if workConn != nil {
			workConn.Close()
		}
		close(readCh)
		close(sendCh)
	}

	// udp service <- frpc <- frps <- frpc visitor <- user
	workConnReaderFn := func(conn net.Conn, readCh chan *msg.UDPPacket) {
		defer closeFn()

		for {
			// first to check sudp proxy is closed or not
			select {
			case <-pxy.closeCh:
				xl.Trace("frpc sudp proxy is closed")
				return
			default:
			}

			var udpMsg msg.UDPPacket
			if errRet := msg.ReadMsgInto(conn, &udpMsg); errRet != nil {
				xl.Warn("read from workConn for sudp error: %v", errRet)
				return
			}

			if errRet := errors.PanicToError(func() {
				readCh <- &udpMsg
			}); errRet != nil {
				xl.Warn("reader goroutine for sudp work connection closed: %v", errRet)
				return
			}
		}
	}

	// udp service -> frpc -> frps -> frpc visitor -> user
	workConnSenderFn := func(conn net.Conn, sendCh chan msg.Message) {
		defer func() {
			closeFn()
			xl.Info("writer goroutine for sudp work connection closed")
		}()

		var errRet error
		for rawMsg := range sendCh {
			switch m := rawMsg.(type) {
			case *msg.UDPPacket:
				xl.Trace("frpc send udp package to frpc visitor, [udp local: %v, remote: %v], [tcp work conn local: %v, remote: %v]",
					m.LocalAddr.String(), m.RemoteAddr.String(), conn.LocalAddr().String(), conn.RemoteAddr().String())
			case *msg.Ping:
				xl.Trace("frpc send ping message to frpc visitor")
			}

			if errRet = msg.WriteMsg(conn, rawMsg); errRet != nil {
				xl.Error("sudp work write error: %v", errRet)
				return
			}
		}
	}

	heartbeatFn := func(conn net.Conn, sendCh chan msg.Message) {
		ticker := time.NewTicker(30 * time.Second)
		defer func() {
			ticker.Stop()
			closeFn()
		}()

		var errRet error
		for {
			select {
			case <-ticker.C:
				if errRet = errors.PanicToError(func() {
					sendCh <- &msg.Ping{}
				}); errRet != nil {
					xl.Warn("heartbeat goroutine for sudp work connection closed")
					return
				}
			case <-pxy.closeCh:
				xl.Trace("frpc sudp proxy is closed")
				return
			}
		}
	}

	go workConnSenderFn(workConn, sendCh)
	go workConnReaderFn(workConn, readCh)
	go heartbeatFn(workConn, sendCh)

	udp.Forwarder(pxy.localAddr, readCh, sendCh, int(pxy.clientCfg.UDPPacketSize))
}

// Common handler for tcp work connections.
func HandleTCPWorkConnection(ctx context.Context, localInfo *config.LocalSvrConf, proxyPlugin plugin.Plugin,
	baseInfo *config.BaseProxyConf, limiter *rate.Limiter, workConn net.Conn, encKey []byte, m *msg.StartWorkConn) {
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
			h.WriteTo(buf)
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
		localConn.Write(extraInfo)
	}

	frpIo.Join(localConn, remote)
	xl.Debug("join connections closed")
}

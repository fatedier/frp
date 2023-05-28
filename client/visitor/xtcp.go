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
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	frpIo "github.com/fatedier/golib/io"
	fmux "github.com/hashicorp/yamux"
	quic "github.com/quic-go/quic-go"
	"golang.org/x/time/rate"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/nathole"
	"github.com/fatedier/frp/pkg/transport"
	frpNet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/xlog"
)

var ErrNoTunnelSession = errors.New("no tunnel session")

type XTCPVisitor struct {
	*BaseVisitor
	session       TunnelSession
	startTunnelCh chan struct{}
	retryLimiter  *rate.Limiter
	cancel        context.CancelFunc

	cfg *config.XTCPVisitorConf
}

func (sv *XTCPVisitor) Run() (err error) {
	sv.ctx, sv.cancel = context.WithCancel(sv.ctx)

	if sv.cfg.Protocol == "kcp" {
		sv.session = NewKCPTunnelSession()
	} else {
		sv.session = NewQUICTunnelSession(&sv.clientCfg)
	}

	sv.l, err = net.Listen("tcp", net.JoinHostPort(sv.cfg.BindAddr, strconv.Itoa(sv.cfg.BindPort)))
	if err != nil {
		return
	}

	go sv.worker()
	go sv.processTunnelStartEvents()
	if sv.cfg.KeepTunnelOpen {
		sv.retryLimiter = rate.NewLimiter(rate.Every(time.Hour/time.Duration(sv.cfg.MaxRetriesAnHour)), sv.cfg.MaxRetriesAnHour)
		go sv.keepTunnelOpenWorker()
	}
	return
}

func (sv *XTCPVisitor) Close() {
	sv.l.Close()
	sv.cancel()
	if sv.session != nil {
		sv.session.Close()
	}
}

func (sv *XTCPVisitor) worker() {
	xl := xlog.FromContextSafe(sv.ctx)
	for {
		conn, err := sv.l.Accept()
		if err != nil {
			xl.Warn("xtcp local listener closed")
			return
		}

		go sv.handleConn(conn)
	}
}

func (sv *XTCPVisitor) processTunnelStartEvents() {
	for {
		select {
		case <-sv.ctx.Done():
			return
		case <-sv.startTunnelCh:
			start := time.Now()
			sv.makeNatHole()
			duration := time.Since(start)
			// avoid too frequently
			if duration < 10*time.Second {
				time.Sleep(10*time.Second - duration)
			}
		}
	}
}

func (sv *XTCPVisitor) keepTunnelOpenWorker() {
	xl := xlog.FromContextSafe(sv.ctx)
	ticker := time.NewTicker(time.Duration(sv.cfg.MinRetryInterval) * time.Second)
	defer ticker.Stop()

	sv.startTunnelCh <- struct{}{}
	for {
		select {
		case <-sv.ctx.Done():
			return
		case <-ticker.C:
			xl.Debug("keepTunnelOpenWorker try to check tunnel...")
			conn, err := sv.getTunnelConn()
			if err != nil {
				xl.Warn("keepTunnelOpenWorker get tunnel connection error: %v", err)
				_ = sv.retryLimiter.Wait(sv.ctx)
				continue
			}
			xl.Debug("keepTunnelOpenWorker check success")
			if conn != nil {
				conn.Close()
			}
		}
	}
}

func (sv *XTCPVisitor) handleConn(userConn net.Conn) {
	xl := xlog.FromContextSafe(sv.ctx)
	defer userConn.Close()

	xl.Debug("get a new xtcp user connection")

	// Open a tunnel connection to the server. If there is already a successful hole-punching connection,
	// it will be reused. Otherwise, it will block and wait for a successful hole-punching connection until timeout.
	tunnelConn, err := sv.openTunnel()
	if err != nil {
		xl.Error("open tunnel error: %v", err)
		return
	}

	var muxConnRWCloser io.ReadWriteCloser = tunnelConn
	if sv.cfg.UseEncryption {
		muxConnRWCloser, err = frpIo.WithEncryption(muxConnRWCloser, []byte(sv.cfg.Sk))
		if err != nil {
			xl.Error("create encryption stream error: %v", err)
			return
		}
	}
	if sv.cfg.UseCompression {
		muxConnRWCloser = frpIo.WithCompression(muxConnRWCloser)
	}

	_, _, errs := frpIo.Join(userConn, muxConnRWCloser)
	xl.Debug("join connections closed")
	if len(errs) > 0 {
		xl.Trace("join connections errors: %v", errs)
	}
}

// openTunnel will open a tunnel connection to the target server.
func (sv *XTCPVisitor) openTunnel() (conn net.Conn, err error) {
	xl := xlog.FromContextSafe(sv.ctx)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeoutC := time.After(20 * time.Second)
	immediateTrigger := make(chan struct{}, 1)
	defer close(immediateTrigger)
	immediateTrigger <- struct{}{}

	for {
		select {
		case <-sv.ctx.Done():
			return nil, sv.ctx.Err()
		case <-immediateTrigger:
			conn, err = sv.getTunnelConn()
		case <-ticker.C:
			conn, err = sv.getTunnelConn()
		case <-timeoutC:
			return nil, fmt.Errorf("open tunnel timeout")
		}

		if err != nil {
			if err != ErrNoTunnelSession {
				xl.Warn("get tunnel connection error: %v", err)
			}
			continue
		}
		return conn, nil
	}
}

func (sv *XTCPVisitor) getTunnelConn() (net.Conn, error) {
	conn, err := sv.session.OpenConn(sv.ctx)
	if err == nil {
		return conn, nil
	}
	sv.session.Close()

	select {
	case sv.startTunnelCh <- struct{}{}:
	default:
	}
	return nil, err
}

// 0. PreCheck
// 1. Prepare
// 2. ExchangeInfo
// 3. MakeNATHole
// 4. Create a tunnel session using an underlying UDP connection.
func (sv *XTCPVisitor) makeNatHole() {
	xl := xlog.FromContextSafe(sv.ctx)
	if err := nathole.PreCheck(sv.ctx, sv.msgTransporter, sv.cfg.ServerName, 5*time.Second); err != nil {
		xl.Warn("nathole precheck error: %v", err)
		return
	}

	prepareResult, err := nathole.Prepare([]string{sv.clientCfg.NatHoleSTUNServer})
	if err != nil {
		xl.Warn("nathole prepare error: %v", err)
		return
	}
	xl.Info("nathole prepare success, nat type: %s, behavior: %s, addresses: %v, assistedAddresses: %v",
		prepareResult.NatType, prepareResult.Behavior, prepareResult.Addrs, prepareResult.AssistedAddrs)

	listenConn := prepareResult.ListenConn

	// send NatHoleVisitor to server
	now := time.Now().Unix()
	transactionID := nathole.NewTransactionID()
	natHoleVisitorMsg := &msg.NatHoleVisitor{
		TransactionID: transactionID,
		ProxyName:     sv.cfg.ServerName,
		Protocol:      sv.cfg.Protocol,
		SignKey:       util.GetAuthKey(sv.cfg.Sk, now),
		Timestamp:     now,
		MappedAddrs:   prepareResult.Addrs,
		AssistedAddrs: prepareResult.AssistedAddrs,
	}

	natHoleRespMsg, err := nathole.ExchangeInfo(sv.ctx, sv.msgTransporter, transactionID, natHoleVisitorMsg, 5*time.Second)
	if err != nil {
		listenConn.Close()
		xl.Warn("nathole exchange info error: %v", err)
		return
	}

	xl.Info("get natHoleRespMsg, sid [%s], protocol [%s], candidate address %v, assisted address %v, detectBehavior: %+v",
		natHoleRespMsg.Sid, natHoleRespMsg.Protocol, natHoleRespMsg.CandidateAddrs,
		natHoleRespMsg.AssistedAddrs, natHoleRespMsg.DetectBehavior)

	newListenConn, raddr, err := nathole.MakeHole(sv.ctx, listenConn, natHoleRespMsg, []byte(sv.cfg.Sk))
	if err != nil {
		listenConn.Close()
		xl.Warn("make hole error: %v", err)
		return
	}
	listenConn = newListenConn
	xl.Info("establishing nat hole connection successful, sid [%s], remoteAddr [%s]", natHoleRespMsg.Sid, raddr)

	if err := sv.session.Init(listenConn, raddr); err != nil {
		listenConn.Close()
		xl.Warn("init tunnel session error: %v", err)
		return
	}
}

type TunnelSession interface {
	Init(listenConn *net.UDPConn, raddr *net.UDPAddr) error
	OpenConn(context.Context) (net.Conn, error)
	Close()
}

type KCPTunnelSession struct {
	session *fmux.Session
	lConn   *net.UDPConn
	mu      sync.RWMutex
}

func NewKCPTunnelSession() TunnelSession {
	return &KCPTunnelSession{}
}

func (ks *KCPTunnelSession) Init(listenConn *net.UDPConn, raddr *net.UDPAddr) error {
	listenConn.Close()
	laddr, _ := net.ResolveUDPAddr("udp", listenConn.LocalAddr().String())
	lConn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		return fmt.Errorf("dial udp error: %v", err)
	}
	remote, err := frpNet.NewKCPConnFromUDP(lConn, true, raddr.String())
	if err != nil {
		return fmt.Errorf("create kcp connection from udp connection error: %v", err)
	}

	fmuxCfg := fmux.DefaultConfig()
	fmuxCfg.KeepAliveInterval = 10 * time.Second
	fmuxCfg.MaxStreamWindowSize = 2 * 1024 * 1024
	fmuxCfg.LogOutput = io.Discard
	session, err := fmux.Client(remote, fmuxCfg)
	if err != nil {
		remote.Close()
		return fmt.Errorf("initial client session error: %v", err)
	}
	ks.mu.Lock()
	ks.session = session
	ks.lConn = lConn
	ks.mu.Unlock()
	return nil
}

func (ks *KCPTunnelSession) OpenConn(ctx context.Context) (net.Conn, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	session := ks.session
	if session == nil {
		return nil, ErrNoTunnelSession
	}
	return session.Open()
}

func (ks *KCPTunnelSession) Close() {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	if ks.session != nil {
		_ = ks.session.Close()
		ks.session = nil
	}
	if ks.lConn != nil {
		_ = ks.lConn.Close()
		ks.lConn = nil
	}
}

type QUICTunnelSession struct {
	session    quic.Connection
	listenConn *net.UDPConn
	mu         sync.RWMutex

	clientCfg *config.ClientCommonConf
}

func NewQUICTunnelSession(clientCfg *config.ClientCommonConf) TunnelSession {
	return &QUICTunnelSession{
		clientCfg: clientCfg,
	}
}

func (qs *QUICTunnelSession) Init(listenConn *net.UDPConn, raddr *net.UDPAddr) error {
	tlsConfig, err := transport.NewClientTLSConfig("", "", "", raddr.String())
	if err != nil {
		return fmt.Errorf("create tls config error: %v", err)
	}
	tlsConfig.NextProtos = []string{"frp"}
	quicConn, err := quic.Dial(listenConn, raddr, raddr.String(), tlsConfig,
		&quic.Config{
			MaxIdleTimeout:     time.Duration(qs.clientCfg.QUICMaxIdleTimeout) * time.Second,
			MaxIncomingStreams: int64(qs.clientCfg.QUICMaxIncomingStreams),
			KeepAlivePeriod:    time.Duration(qs.clientCfg.QUICKeepalivePeriod) * time.Second,
		})
	if err != nil {
		return fmt.Errorf("dial quic error: %v", err)
	}
	qs.mu.Lock()
	qs.session = quicConn
	qs.listenConn = listenConn
	qs.mu.Unlock()
	return nil
}

func (qs *QUICTunnelSession) OpenConn(ctx context.Context) (net.Conn, error) {
	qs.mu.RLock()
	defer qs.mu.RUnlock()
	session := qs.session
	if session == nil {
		return nil, ErrNoTunnelSession
	}
	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return frpNet.QuicStreamToNetConn(stream, session), nil
}

func (qs *QUICTunnelSession) Close() {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	if qs.session != nil {
		_ = qs.session.CloseWithError(0, "")
		qs.session = nil
	}
	if qs.listenConn != nil {
		_ = qs.listenConn.Close()
		qs.listenConn = nil
	}
}

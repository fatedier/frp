// Copyright 2024 The frp Authors
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
	"net"
	"strconv"
	"sync"
	"time"

	libio "github.com/fatedier/golib/io"
	quic "github.com/quic-go/quic-go"
	"golang.org/x/time/rate"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/naming"
	"github.com/fatedier/frp/pkg/nathole"
	"github.com/fatedier/frp/pkg/transport"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type XUDPVisitor struct {
	*BaseVisitor
	session       *xudpTunnelSession
	startTunnelCh chan struct{}
	retryLimiter  *rate.Limiter
	cancel        context.CancelFunc

	cfg *v1.XUDPVisitorConfig
}

func (sv *XUDPVisitor) Run() (err error) {
	sv.ctx, sv.cancel = context.WithCancel(sv.ctx)

	sv.session = newXUDPTunnelSession(sv.clientCfg)

	if sv.cfg.BindPort > 0 {
		sv.l, err = net.Listen("tcp", net.JoinHostPort(sv.cfg.BindAddr, strconv.Itoa(sv.cfg.BindPort)))
		if err != nil {
			return
		}
		go sv.acceptLoop(sv.l, "xudp local", sv.handleConn)
	}

	go sv.acceptLoop(sv.internalLn, "xudp internal", sv.handleConn)
	go sv.processTunnelStartEvents()
	if sv.cfg.KeepTunnelOpen {
		sv.retryLimiter = rate.NewLimiter(rate.Every(time.Hour/time.Duration(sv.cfg.MaxRetriesAnHour)), sv.cfg.MaxRetriesAnHour)
		go sv.keepTunnelOpenWorker()
	}

	if sv.plugin != nil {
		sv.plugin.Start()
	}
	return
}

func (sv *XUDPVisitor) Close() {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	sv.BaseVisitor.Close()
	if sv.cancel != nil {
		sv.cancel()
	}
	if sv.session != nil {
		sv.session.Close()
	}
}

func (sv *XUDPVisitor) processTunnelStartEvents() {
	for {
		select {
		case <-sv.ctx.Done():
			return
		case <-sv.startTunnelCh:
			start := time.Now()
			sv.makeNatHole()
			duration := time.Since(start)
			if duration < 10*time.Second {
				time.Sleep(10*time.Second - duration)
			}
		}
	}
}

func (sv *XUDPVisitor) keepTunnelOpenWorker() {
	xl := xlog.FromContextSafe(sv.ctx)
	ticker := time.NewTicker(time.Duration(sv.cfg.MinRetryInterval) * time.Second)
	defer ticker.Stop()

	select {
	case sv.startTunnelCh <- struct{}{}:
	case <-sv.ctx.Done():
		return
	}
	for {
		select {
		case <-sv.ctx.Done():
			return
		case <-ticker.C:
			xl.Debugf("xudp keepTunnelOpenWorker try to check tunnel...")
			conn, err := sv.getTunnelConn(sv.ctx)
			if err != nil {
				xl.Warnf("xudp keepTunnelOpenWorker get tunnel connection error: %v", err)
				_ = sv.retryLimiter.Wait(sv.ctx)
				continue
			}
			xl.Debugf("xudp keepTunnelOpenWorker check success")
			if conn != nil {
				conn.Close()
			}
		}
	}
}

func (sv *XUDPVisitor) handleConn(userConn net.Conn) {
	xl := xlog.FromContextSafe(sv.ctx)
	isConnTransferred := false
	var tunnelErr error
	defer func() {
		if !isConnTransferred {
			if tunnelErr != nil {
				if eConn, ok := userConn.(interface{ CloseWithError(error) error }); ok {
					_ = eConn.CloseWithError(tunnelErr)
					return
				}
			}
			userConn.Close()
		}
	}()

	xl.Debugf("get a new xudp user connection")

	ctx := sv.ctx
	if sv.cfg.FallbackTo != "" {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(sv.cfg.FallbackTimeoutMs)*time.Millisecond)
		defer cancel()
		ctx = timeoutCtx
	}
	tunnelConn, err := sv.openTunnel(ctx)
	if err != nil {
		xl.Errorf("xudp open tunnel error: %v", err)
		tunnelErr = err

		if sv.cfg.FallbackTo == "" {
			return
		}

		xl.Debugf("xudp try to transfer connection to visitor: %s", sv.cfg.FallbackTo)
		if err := sv.helper.TransferConn(sv.cfg.FallbackTo, userConn); err != nil {
			xl.Errorf("xudp transfer connection to visitor %s error: %v", sv.cfg.FallbackTo, err)
			return
		}
		isConnTransferred = true
		return
	}

	muxConnRWCloser, recycleFn, err := wrapVisitorConn(tunnelConn, sv.cfg.GetBaseConfig())
	if err != nil {
		xl.Errorf("xudp %v", err)
		tunnelConn.Close()
		tunnelErr = err
		return
	}
	defer recycleFn()

	_, _, errs := libio.Join(userConn, muxConnRWCloser)
	xl.Debugf("xudp join connections closed")
	if len(errs) > 0 {
		xl.Tracef("xudp join connections errors: %v", errs)
	}
}

func (sv *XUDPVisitor) openTunnel(ctx context.Context) (conn net.Conn, err error) {
	xl := xlog.FromContextSafe(sv.ctx)
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-sv.ctx.Done():
			return nil, sv.ctx.Err()
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("xudp open tunnel timeout")
			}
			return nil, ctx.Err()
		case <-timer.C:
			conn, err = sv.getTunnelConn(ctx)
			if err != nil {
				if !errors.Is(err, ErrNoTunnelSession) {
					xl.Warnf("xudp get tunnel connection error: %v", err)
				}
				timer.Reset(500 * time.Millisecond)
				continue
			}
			return conn, nil
		}
	}
}

func (sv *XUDPVisitor) getTunnelConn(ctx context.Context) (net.Conn, error) {
	conn, err := sv.session.OpenConn(ctx)
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

// makeNatHole performs the xudp NAT hole punching with multi-STUN prediction.
// 0. PreCheck
// 1. Multi-STUN Discover (concurrent queries for port prediction)
// 2. Prepare
// 3. Realm Rendezvous Exchange (server relays addresses then exits data flow)
// 4. XUDPMakeHole (with predicted ports)
// 5. Create decoupled QUIC tunnel session
func (sv *XUDPVisitor) makeNatHole() {
	xl := xlog.FromContextSafe(sv.ctx)
	targetProxyName := naming.BuildTargetServerProxyName(sv.clientCfg.User, sv.cfg.ServerUser, sv.cfg.ServerName)
	xl.Tracef("xudp makeNatHole start")

	if err := nathole.PreCheck(sv.ctx, sv.helper.MsgTransporter(), targetProxyName, 5*time.Second); err != nil {
		xl.Warnf("xudp nathole precheck error: %v", err)
		return
	}

	// Build STUN server list
	stunServers := []string{sv.clientCfg.NatHoleSTUNServer}
	if len(sv.cfg.STUNServers) > 0 {
		stunServers = append(stunServers, sv.cfg.STUNServers...)
	}

	// Multi-STUN discovery for port prediction
	var portDelta int
	var multiSTUNLocalAddr string
	if len(stunServers) > 1 {
		multiResult, err := nathole.DiscoverMultiSTUN(sv.ctx, stunServers, "")
		if err != nil {
			xl.Warnf("xudp multi-stun discover error (non-fatal): %v", err)
		} else {
			portDelta = multiResult.PortDelta
			if multiResult.LocalAddr != nil {
				multiSTUNLocalAddr = multiResult.LocalAddr.String()
			}
			xl.Infof("xudp multi-stun discovery: portDelta=%d, addrs=%v", portDelta, multiResult.Addrs)
		}
	}

	xl.Tracef("xudp nathole prepare start")

	// Prepare NAT traversal options
	var opts nathole.PrepareOptions
	if sv.cfg.NatTraversal != nil && sv.cfg.NatTraversal.DisableAssistedAddrs {
		opts.DisableAssistedAddrs = true
	}
	// Reuse the multi-STUN local address so portDelta corresponds to the punched socket
	opts.LocalAddr = multiSTUNLocalAddr

	prepareResult, err := nathole.Prepare(stunServers[:1], opts)
	if err != nil {
		xl.Warnf("xudp nathole prepare error: %v", err)
		return
	}

	xl.Infof("xudp nathole prepare success, nat type: %s, behavior: %s, addresses: %v, assistedAddresses: %v",
		prepareResult.NatType, prepareResult.Behavior, prepareResult.Addrs, prepareResult.AssistedAddrs)

	listenConn := prepareResult.ListenConn

	// Realm rendezvous exchange - server relays addresses then exits data flow
	now := time.Now().Unix()
	transactionID := nathole.NewTransactionID()
	natHoleVisitorMsg := &msg.NatHoleVisitor{
		TransactionID: transactionID,
		ProxyName:     targetProxyName,
		Protocol:      sv.cfg.Protocol,
		SignKey:       util.GetAuthKey(sv.cfg.SecretKey, now),
		Timestamp:     now,
		MappedAddrs:   prepareResult.Addrs,
		AssistedAddrs: prepareResult.AssistedAddrs,
	}

	xl.Tracef("xudp rendezvous exchange start")
	natHoleRespMsg, err := nathole.XUDPRendezvousExchange(sv.ctx, sv.helper.MsgTransporter(), transactionID, natHoleVisitorMsg, 5*time.Second)
	if err != nil {
		listenConn.Close()
		xl.Warnf("xudp rendezvous exchange error: %v", err)
		return
	}

	xl.Infof("xudp get natHoleRespMsg, sid [%s], protocol [%s], candidate address %v, assisted address %v, detectBehavior: %+v",
		natHoleRespMsg.Sid, natHoleRespMsg.Protocol, natHoleRespMsg.CandidateAddrs,
		natHoleRespMsg.AssistedAddrs, natHoleRespMsg.DetectBehavior)

	// XUDPMakeHole with port prediction
	newListenConn, raddr, err := nathole.XUDPMakeHole(sv.ctx, listenConn, natHoleRespMsg, []byte(sv.cfg.SecretKey), portDelta)
	if err != nil {
		listenConn.Close()
		xl.Warnf("xudp make hole error: %v", err)
		return
	}
	listenConn = newListenConn
	xl.Infof("xudp establishing nat hole connection successful, sid [%s], remoteAddr [%s]", natHoleRespMsg.Sid, raddr)

	// Initialize the decoupled tunnel session
	if err := sv.session.Init(listenConn, raddr); err != nil {
		listenConn.Close()
		xl.Warnf("xudp init tunnel session error: %v", err)
		return
	}
}

// xudpTunnelSession implements a decoupled QUIC tunnel session for xudp.
// It runs in an isolated goroutine that survives control channel drops.
type xudpTunnelSession struct {
	session    *quic.Conn
	listenConn *net.UDPConn
	mu         sync.RWMutex

	clientCfg *v1.ClientCommonConfig
}

func newXUDPTunnelSession(clientCfg *v1.ClientCommonConfig) *xudpTunnelSession {
	return &xudpTunnelSession{
		clientCfg: clientCfg,
	}
}

func (xs *xudpTunnelSession) Init(listenConn *net.UDPConn, raddr *net.UDPAddr) error {
	tlsConfig, err := transport.NewClientTLSConfig("", "", "", raddr.String())
	if err != nil {
		return fmt.Errorf("xudp create tls config error: %v", err)
	}
	tlsConfig.NextProtos = []string{"frp-xudp"}

	// Use a background context so the QUIC session is decoupled from the control channel
	quicConn, err := quic.Dial(context.Background(), listenConn, raddr, tlsConfig,
		&quic.Config{
			MaxIdleTimeout:     time.Duration(xs.clientCfg.Transport.QUIC.MaxIdleTimeout) * time.Second,
			MaxIncomingStreams: int64(xs.clientCfg.Transport.QUIC.MaxIncomingStreams),
			KeepAlivePeriod:    time.Duration(xs.clientCfg.Transport.QUIC.KeepalivePeriod) * time.Second,
		})
	if err != nil {
		return fmt.Errorf("xudp dial quic error: %v", err)
	}
	xs.mu.Lock()
	xs.session = quicConn
	xs.listenConn = listenConn
	xs.mu.Unlock()
	return nil
}

func (xs *xudpTunnelSession) OpenConn(ctx context.Context) (net.Conn, error) {
	xs.mu.RLock()
	defer xs.mu.RUnlock()
	session := xs.session
	if session == nil {
		return nil, ErrNoTunnelSession
	}
	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return netpkg.QuicStreamToNetConn(stream, session), nil
}

func (xs *xudpTunnelSession) Close() {
	xs.mu.Lock()
	defer xs.mu.Unlock()
	if xs.session != nil {
		_ = xs.session.CloseWithError(0, "")
		xs.session = nil
	}
	if xs.listenConn != nil {
		_ = xs.listenConn.Close()
		xs.listenConn = nil
	}
}

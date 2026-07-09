// Copyright 2025 The frp Authors
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

	liberrors "github.com/fatedier/golib/errors"
	libio "github.com/fatedier/golib/io"
	"golang.org/x/time/rate"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/naming"
	"github.com/fatedier/frp/pkg/nathole"
	"github.com/fatedier/frp/pkg/proto/udp"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/xlog"
)

// Stream tags carried as the first byte of every tunnel stream so the provider
// can route it to the TCP or UDP local service. Both ends of the combined proxy
// are new code, so this framing is private to xtcpxudp and does not affect xtcp.
const (
	xtcpxudpStreamTagTCP byte = 0x01
	xtcpxudpStreamTagUDP byte = 0x02
)

// XTCPXUDPVisitor carries BOTH TCP and UDP to a provider over a SINGLE
// hole-punched tunnel. It binds a local TCP listener and a local UDP listener on
// the same BindAddr:BindPort; each TCP connection and the UDP packet channel ride
// their own tagged stream on the shared quic/kcp session.
type XTCPXUDPVisitor struct {
	*BaseVisitor

	session       TunnelSession
	startTunnelCh chan struct{}
	retryLimiter  *rate.Limiter
	cancel        context.CancelFunc

	// UDP local edge
	checkCloseCh chan struct{}
	udpConn      *net.UDPConn
	readCh       chan *msg.UDPPacket
	sendCh       chan *msg.UDPPacket

	cfg *v1.XTCPXUDPVisitorConfig
}

func (sv *XTCPXUDPVisitor) Run() (err error) {
	sv.ctx, sv.cancel = context.WithCancel(sv.ctx)

	if sv.cfg.Protocol == "kcp" {
		sv.session = NewKCPTunnelSession()
	} else {
		sv.session = NewQUICTunnelSession(sv.clientCfg)
	}

	xl := xlog.FromContextSafe(sv.ctx)
	if sv.cfg.BindPort > 0 {
		// local TCP listener
		sv.l, err = net.Listen("tcp", net.JoinHostPort(sv.cfg.BindAddr, strconv.Itoa(sv.cfg.BindPort)))
		if err != nil {
			return
		}
		go sv.acceptLoop(sv.l, "xtcpxudp tcp local", sv.handleTCPConn)

		// local UDP listener (same addr:port)
		var addr *net.UDPAddr
		addr, err = net.ResolveUDPAddr("udp", net.JoinHostPort(sv.cfg.BindAddr, strconv.Itoa(sv.cfg.BindPort)))
		if err != nil {
			return fmt.Errorf("xtcpxudp ResolveUDPAddr error: %v", err)
		}
		sv.udpConn, err = net.ListenUDP("udp", addr)
		if err != nil {
			return fmt.Errorf("xtcpxudp listen udp port %s error: %v", addr.String(), err)
		}
		sv.sendCh = make(chan *msg.UDPPacket, 1024)
		sv.readCh = make(chan *msg.UDPPacket, 1024)
		go sv.udpDispatcher()
		go udp.ForwardUserConn(sv.udpConn, sv.readCh, sv.sendCh, int(sv.clientCfg.UDPPacketSize))

		xl.Infof("xtcpxudp start to work, listen on %s (tcp+udp)", addr)
	}

	// TCP connections redirected from other visitors / plugins
	go sv.acceptLoop(sv.internalLn, "xtcpxudp internal", sv.handleTCPConn)

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

func (sv *XTCPXUDPVisitor) Close() {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	if sv.checkCloseCh != nil {
		select {
		case <-sv.checkCloseCh:
		default:
			close(sv.checkCloseCh)
		}
	}
	sv.BaseVisitor.Close()
	if sv.cancel != nil {
		sv.cancel()
	}
	if sv.session != nil {
		sv.session.Close()
	}
	if sv.udpConn != nil {
		sv.udpConn.Close()
	}
	if sv.readCh != nil {
		close(sv.readCh)
	}
	if sv.sendCh != nil {
		close(sv.sendCh)
	}
}

// ---------------- TCP path (tag 0x01) ----------------

func (sv *XTCPXUDPVisitor) handleTCPConn(userConn net.Conn) {
	xl := xlog.FromContextSafe(sv.ctx)
	defer userConn.Close()

	tunnelConn, err := sv.openTunnel(sv.ctx)
	if err != nil {
		xl.Errorf("open tunnel error: %v", err)
		return
	}
	// tag this stream as TCP before any encryption wrapping
	if _, err := tunnelConn.Write([]byte{xtcpxudpStreamTagTCP}); err != nil {
		xl.Errorf("write tcp stream tag error: %v", err)
		tunnelConn.Close()
		return
	}

	muxConnRWCloser, recycleFn, err := wrapVisitorConn(tunnelConn, sv.cfg.GetBaseConfig())
	if err != nil {
		xl.Errorf("%v", err)
		tunnelConn.Close()
		return
	}
	defer recycleFn()

	_, _, errs := libio.Join(userConn, muxConnRWCloser)
	xl.Debugf("join connections closed")
	if len(errs) > 0 {
		xl.Tracef("join connections errors: %v", errs)
	}
}

// ---------------- UDP path (tag 0x02) ----------------

func (sv *XTCPXUDPVisitor) udpDispatcher() {
	xl := xlog.FromContextSafe(sv.ctx)

	var firstPacket *msg.UDPPacket
	for {
		select {
		case firstPacket = <-sv.sendCh:
			if firstPacket == nil {
				return
			}
		case <-sv.checkCloseCh:
			return
		}

		visitorConn, recycleFn, err := sv.getNewUDPVisitorConn()
		if err != nil {
			xl.Warnf("open xtcpxudp udp tunnel connection error: %v", err)
			continue
		}

		func() {
			defer recycleFn()
			sv.udpWorker(visitorConn, firstPacket)
		}()

		select {
		case <-sv.checkCloseCh:
			return
		default:
		}
	}
}

func (sv *XTCPXUDPVisitor) getNewUDPVisitorConn() (net.Conn, func(), error) {
	tunnelConn, err := sv.openTunnel(sv.ctx)
	if err != nil {
		return nil, func() {}, err
	}
	// tag this stream as UDP before any encryption wrapping
	if _, err := tunnelConn.Write([]byte{xtcpxudpStreamTagUDP}); err != nil {
		tunnelConn.Close()
		return nil, func() {}, err
	}
	rwc, recycleFn, err := wrapVisitorConn(tunnelConn, sv.cfg.GetBaseConfig())
	if err != nil {
		tunnelConn.Close()
		return nil, func() {}, err
	}
	return netpkg.WrapReadWriteCloserToConn(rwc, tunnelConn), recycleFn, nil
}

func (sv *XTCPXUDPVisitor) udpWorker(workConn net.Conn, firstPacket *msg.UDPPacket) {
	xl := xlog.FromContextSafe(sv.ctx)
	xl.Debugf("starting xtcpxudp udp proxy worker")
	payloadConn := msg.NewConn(workConn, msg.NewReadWriter(workConn, sv.clientCfg.Transport.WireProtocol))

	wg := &sync.WaitGroup{}
	wg.Add(2)
	closeCh := make(chan struct{})

	workConnReaderFn := func(payloadConn *msg.Conn) {
		defer func() {
			payloadConn.Close()
			close(closeCh)
			wg.Done()
		}()

		for {
			var (
				rawMsg msg.Message
				errRet error
			)
			_ = payloadConn.SetReadDeadline(time.Now().Add(60 * time.Second))
			if rawMsg, errRet = payloadConn.ReadMsg(); errRet != nil {
				xl.Warnf("read from workconn for user udp conn error: %v", errRet)
				return
			}
			_ = payloadConn.SetReadDeadline(time.Time{})
			switch m := rawMsg.(type) {
			case *msg.Ping:
				continue
			case *msg.UDPPacket:
				if errRet := liberrors.PanicToError(func() {
					sv.readCh <- m
				}); errRet != nil {
					xl.Infof("reader goroutine for udp work connection closed")
					return
				}
			}
		}
	}

	workConnSenderFn := func(payloadConn *msg.Conn) {
		defer func() {
			payloadConn.Close()
			wg.Done()
		}()

		var errRet error
		if firstPacket != nil {
			if errRet = payloadConn.WriteMsg(firstPacket); errRet != nil {
				return
			}
		}
		for {
			select {
			case udpMsg, ok := <-sv.sendCh:
				if !ok {
					return
				}
				if errRet = payloadConn.WriteMsg(udpMsg); errRet != nil {
					return
				}
			case <-closeCh:
				return
			}
		}
	}

	go workConnReaderFn(payloadConn)
	go workConnSenderFn(payloadConn)

	wg.Wait()
	xl.Infof("xtcpxudp udp worker is closed")
}

// ---------------- shared hole-punch / tunnel session (mirrors xtcp/xudp) ----------------

func (sv *XTCPXUDPVisitor) processTunnelStartEvents() {
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

func (sv *XTCPXUDPVisitor) keepTunnelOpenWorker() {
	xl := xlog.FromContextSafe(sv.ctx)
	ticker := time.NewTicker(time.Duration(sv.cfg.MinRetryInterval) * time.Second)
	defer ticker.Stop()

	sv.startTunnelCh <- struct{}{}
	for {
		select {
		case <-sv.ctx.Done():
			return
		case <-ticker.C:
			xl.Debugf("keepTunnelOpenWorker try to check tunnel...")
			conn, err := sv.getTunnelConn(sv.ctx)
			if err != nil {
				xl.Warnf("keepTunnelOpenWorker get tunnel connection error: %v", err)
				_ = sv.retryLimiter.Wait(sv.ctx)
				continue
			}
			xl.Debugf("keepTunnelOpenWorker check success")
			if conn != nil {
				conn.Close()
			}
		}
	}
}

func (sv *XTCPXUDPVisitor) openTunnel(ctx context.Context) (conn net.Conn, err error) {
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
				return nil, fmt.Errorf("open tunnel timeout")
			}
			return nil, ctx.Err()
		case <-timer.C:
			conn, err = sv.getTunnelConn(ctx)
			if err != nil {
				if !errors.Is(err, ErrNoTunnelSession) {
					xl.Warnf("get tunnel connection error: %v", err)
				}
				timer.Reset(500 * time.Millisecond)
				continue
			}
			return conn, nil
		}
	}
}

func (sv *XTCPXUDPVisitor) getTunnelConn(ctx context.Context) (net.Conn, error) {
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

func (sv *XTCPXUDPVisitor) makeNatHole() {
	xl := xlog.FromContextSafe(sv.ctx)
	targetProxyName := naming.BuildTargetServerProxyName(sv.clientCfg.User, sv.cfg.ServerUser, sv.cfg.ServerName)
	xl.Tracef("makeNatHole start")
	if err := nathole.PreCheck(sv.ctx, sv.helper.MsgTransporter(), targetProxyName, 5*time.Second); err != nil {
		xl.Warnf("nathole precheck error: %v", err)
		return
	}

	xl.Tracef("nathole prepare start")
	var opts nathole.PrepareOptions
	if sv.cfg.NatTraversal != nil && sv.cfg.NatTraversal.DisableAssistedAddrs {
		opts.DisableAssistedAddrs = true
	}

	prepareResult, err := nathole.Prepare([]string{sv.clientCfg.NatHoleSTUNServer}, opts)
	if err != nil {
		xl.Warnf("nathole prepare error: %v", err)
		return
	}
	xl.Infof("nathole prepare success, nat type: %s, behavior: %s, addresses: %v, assistedAddresses: %v",
		prepareResult.NatType, prepareResult.Behavior, prepareResult.Addrs, prepareResult.AssistedAddrs)

	listenConn := prepareResult.ListenConn

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

	xl.Tracef("nathole exchange info start")
	natHoleRespMsg, err := nathole.ExchangeInfo(sv.ctx, sv.helper.MsgTransporter(), transactionID, natHoleVisitorMsg, 5*time.Second)
	if err != nil {
		listenConn.Close()
		xl.Warnf("nathole exchange info error: %v", err)
		return
	}

	xl.Infof("get natHoleRespMsg, sid [%s], protocol [%s], candidate address %v, assisted address %v, detectBehavior: %+v",
		natHoleRespMsg.Sid, natHoleRespMsg.Protocol, natHoleRespMsg.CandidateAddrs,
		natHoleRespMsg.AssistedAddrs, natHoleRespMsg.DetectBehavior)

	newListenConn, raddr, err := nathole.MakeHole(sv.ctx, listenConn, natHoleRespMsg, []byte(sv.cfg.SecretKey))
	if err != nil {
		listenConn.Close()
		xl.Warnf("make hole error: %v", err)
		return
	}
	listenConn = newListenConn
	xl.Infof("establishing nat hole connection successful, sid [%s], remoteAddr [%s]", natHoleRespMsg.Sid, raddr)

	if err := sv.session.Init(listenConn, raddr); err != nil {
		listenConn.Close()
		xl.Warnf("init tunnel session error: %v", err)
		return
	}
}

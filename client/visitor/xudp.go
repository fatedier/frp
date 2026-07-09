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

// XUDPVisitor is the UDP counterpart of XTCPVisitor: it reaches the provider via
// NAT hole punching (nathole + a reliable quic/kcp TunnelSession over the punched
// UDP hole), and bridges a local UDP listener to that tunnel by framing datagrams
// as msg.UDPPacket (reusing the sudp packet machinery).
//
// The hole-punching / session code is shared verbatim with XTCPVisitor; only the
// local edge is UDP instead of a TCP listener + stream splice.
type XUDPVisitor struct {
	*BaseVisitor

	session       TunnelSession
	startTunnelCh chan struct{}
	retryLimiter  *rate.Limiter
	cancel        context.CancelFunc

	checkCloseCh chan struct{}
	// udpConn is the local udp listener for user datagrams.
	udpConn *net.UDPConn
	readCh  chan *msg.UDPPacket
	sendCh  chan *msg.UDPPacket

	cfg *v1.XUDPVisitorConfig
}

func (sv *XUDPVisitor) Run() (err error) {
	sv.ctx, sv.cancel = context.WithCancel(sv.ctx)

	if sv.cfg.Protocol == "kcp" {
		sv.session = NewKCPTunnelSession()
	} else {
		sv.session = NewQUICTunnelSession(sv.clientCfg)
	}

	xl := xlog.FromContextSafe(sv.ctx)
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(sv.cfg.BindAddr, strconv.Itoa(sv.cfg.BindPort)))
	if err != nil {
		return fmt.Errorf("xudp ResolveUDPAddr error: %v", err)
	}
	sv.udpConn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("xudp listen udp port %s error: %v", addr.String(), err)
	}

	sv.sendCh = make(chan *msg.UDPPacket, 1024)
	sv.readCh = make(chan *msg.UDPPacket, 1024)

	xl.Infof("xudp start to work, listen on %s", addr)

	go sv.dispatcher()
	go udp.ForwardUserConn(sv.udpConn, sv.readCh, sv.sendCh, int(sv.clientCfg.UDPPacketSize), nil)

	go sv.processTunnelStartEvents()
	if sv.cfg.KeepTunnelOpen {
		sv.retryLimiter = rate.NewLimiter(rate.Every(time.Hour/time.Duration(sv.cfg.MaxRetriesAnHour)), sv.cfg.MaxRetriesAnHour)
		go sv.keepTunnelOpenWorker()
	}
	return
}

func (sv *XUDPVisitor) Close() {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	select {
	case <-sv.checkCloseCh:
		return
	default:
		close(sv.checkCloseCh)
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
	close(sv.readCh)
	close(sv.sendCh)
}

// dispatcher waits for the first user datagram, opens (or reuses) a hole-punched
// tunnel stream, and pumps all datagrams through it via a single worker. If the
// tunnel drops it loops and re-establishes on the next datagram.
func (sv *XUDPVisitor) dispatcher() {
	xl := xlog.FromContextSafe(sv.ctx)

	var firstPacket *msg.UDPPacket
	for {
		select {
		case firstPacket = <-sv.sendCh:
			if firstPacket == nil {
				xl.Infof("frpc xudp visitor proxy is closed")
				return
			}
		case <-sv.checkCloseCh:
			xl.Infof("frpc xudp visitor proxy is closed")
			return
		}

		visitorConn, recycleFn, err := sv.getNewVisitorConn()
		if err != nil {
			xl.Warnf("open xudp tunnel connection error: %v", err)
			continue
		}

		// visitorConn always be closed when worker done.
		func() {
			defer recycleFn()
			sv.worker(visitorConn, firstPacket)
		}()

		select {
		case <-sv.checkCloseCh:
			return
		default:
		}
	}
}

func (sv *XUDPVisitor) worker(workConn net.Conn, firstPacket *msg.UDPPacket) {
	xl := xlog.FromContextSafe(sv.ctx)
	xl.Debugf("starting xudp proxy worker")
	payloadConn := msg.NewConn(workConn, msg.NewReadWriter(workConn, sv.clientCfg.Transport.WireProtocol))

	wg := &sync.WaitGroup{}
	wg.Add(2)
	closeCh := make(chan struct{})

	// udp service -> frpc(provider) -> hole -> frpc visitor -> user
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

			// provider frpc sends heartbeat over the tunnel to keep it alive
			_ = payloadConn.SetReadDeadline(time.Now().Add(60 * time.Second))
			if rawMsg, errRet = payloadConn.ReadMsg(); errRet != nil {
				xl.Warnf("read from workconn for user udp conn error: %v", errRet)
				return
			}

			_ = payloadConn.SetReadDeadline(time.Time{})
			switch m := rawMsg.(type) {
			case *msg.Ping:
				xl.Debugf("frpc xudp visitor get ping message from provider")
				continue
			case *msg.UDPPacket:
				if errRet := liberrors.PanicToError(func() {
					sv.readCh <- m
					xl.Tracef("frpc xudp visitor get udp packet from workConn, len: %d", len(m.Content))
				}); errRet != nil {
					xl.Infof("reader goroutine for udp work connection closed")
					return
				}
			}
		}
	}

	// udp service <- frpc(provider) <- hole <- frpc visitor <- user
	workConnSenderFn := func(payloadConn *msg.Conn) {
		defer func() {
			payloadConn.Close()
			wg.Done()
		}()

		var errRet error
		if firstPacket != nil {
			if errRet = payloadConn.WriteMsg(firstPacket); errRet != nil {
				xl.Warnf("sender goroutine for udp work connection closed: %v", errRet)
				return
			}
			xl.Tracef("send udp package to workConn, len: %d", len(firstPacket.Content))
		}

		for {
			select {
			case udpMsg, ok := <-sv.sendCh:
				if !ok {
					xl.Infof("sender goroutine for udp work connection closed")
					return
				}

				if errRet = payloadConn.WriteMsg(udpMsg); errRet != nil {
					xl.Warnf("sender goroutine for udp work connection closed: %v", errRet)
					return
				}
				xl.Tracef("send udp package to workConn, len: %d", len(udpMsg.Content))
			case <-closeCh:
				return
			}
		}
	}

	go workConnReaderFn(payloadConn)
	go workConnSenderFn(payloadConn)

	wg.Wait()
	xl.Infof("xudp worker is closed")
}

func (sv *XUDPVisitor) getNewVisitorConn() (net.Conn, func(), error) {
	tunnelConn, err := sv.openTunnel(sv.ctx)
	if err != nil {
		return nil, func() {}, err
	}
	rwc, recycleFn, err := wrapVisitorConn(tunnelConn, sv.cfg.GetBaseConfig())
	if err != nil {
		tunnelConn.Close()
		return nil, func() {}, err
	}
	return netpkg.WrapReadWriteCloserToConn(rwc, tunnelConn), recycleFn, nil
}

// ------------------------------------------------------------------------------
// The following hole-punching / tunnel-session helpers mirror XTCPVisitor. The
// only functional difference from XTCP is the local edge (UDP vs TCP), handled
// above; the NAT traversal path below is identical.
// ------------------------------------------------------------------------------

func (sv *XUDPVisitor) processTunnelStartEvents() {
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

func (sv *XUDPVisitor) keepTunnelOpenWorker() {
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

// openTunnel opens a tunnel stream to the provider. If there is already a
// successful hole-punching session it is reused, otherwise it blocks and waits
// for a successful hole-punching session until timeout.
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

// 0. PreCheck
// 1. Prepare
// 2. ExchangeInfo
// 3. MakeNATHole
// 4. Create a tunnel session using an underlying UDP connection.
func (sv *XUDPVisitor) makeNatHole() {
	xl := xlog.FromContextSafe(sv.ctx)
	targetProxyName := naming.BuildTargetServerProxyName(sv.clientCfg.User, sv.cfg.ServerUser, sv.cfg.ServerName)
	xl.Tracef("makeNatHole start")
	if err := nathole.PreCheck(sv.ctx, sv.helper.MsgTransporter(), targetProxyName, 5*time.Second); err != nil {
		xl.Warnf("nathole precheck error: %v", err)
		return
	}

	xl.Tracef("nathole prepare start")

	// Prepare NAT traversal options
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

	// send NatHoleVisitor to server
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

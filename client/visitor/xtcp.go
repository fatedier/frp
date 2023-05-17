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
	"strconv"
	"sync"
	"time"

	frpIo "github.com/fatedier/golib/io"
	fmux "github.com/hashicorp/yamux"
	quic "github.com/quic-go/quic-go"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/nathole"
	"github.com/fatedier/frp/pkg/transport"
	frpNet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type XTCPVisitor struct {
	*BaseVisitor
	kcpSession    *fmux.Session
	quicSession   quic.Connection
	startTunnelCh chan struct{}
	mu            sync.RWMutex
	cancel        context.CancelFunc

	cfg *config.XTCPVisitorConf
}

func (sv *XTCPVisitor) Run() (err error) {
	sv.ctx, sv.cancel = context.WithCancel(sv.ctx)

	sv.l, err = net.Listen("tcp", net.JoinHostPort(sv.cfg.BindAddr, strconv.Itoa(sv.cfg.BindPort)))
	if err != nil {
		return
	}

	go sv.worker()
	go sv.keepTunnelConnection()
	return
}

func (sv *XTCPVisitor) Close() {
	sv.l.Close()
	sv.cancel()
	if sv.kcpSession != nil {
		sv.kcpSession.Close()
	}
	if sv.quicSession != nil {
		_ = sv.quicSession.CloseWithError(0, "")
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

func (sv *XTCPVisitor) keepTunnelConnection() {
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
			xl.Warn("get tunnel connection error: %v", err)
			continue
		}
		// If there is no tunnel connection, we will continue to wait for a new one until timeout.
		if conn == nil {
			continue
		}
		return conn, nil
	}
}

func (sv *XTCPVisitor) getTunnelConn() (conn net.Conn, err error) {
	var (
		kcpSession  *fmux.Session
		quicSession quic.Connection
	)
	sv.mu.RLock()
	kcpSession = sv.kcpSession
	quicSession = sv.quicSession
	sv.mu.RUnlock()

	if sv.cfg.Protocol == "kcp" && kcpSession != nil {
		conn, err = kcpSession.Open()
		if err != nil {
			sv.mu.Lock()
			if sv.kcpSession != nil {
				sv.kcpSession.Close()
				sv.kcpSession = nil
			}
			sv.mu.Unlock()
			return nil, err
		}
		return
	} else if quicSession != nil {
		stream, err := quicSession.OpenStreamSync(sv.ctx)
		if err != nil {
			sv.mu.Lock()
			if sv.quicSession != nil {
				_ = sv.quicSession.CloseWithError(0, "")
				sv.quicSession = nil
			}
			sv.mu.Unlock()
			return nil, err
		}
		conn = frpNet.QuicStreamToNetConn(stream, quicSession)
		return conn, err
	}

	select {
	case sv.startTunnelCh <- struct{}{}:
	default:
	}
	return nil, nil
}

// 1. Prepare
// 2. ExchangeInfo
// 3. MakeHole
// 4. Create a QUIC or KCP session using an underlying UDP connection.
func (sv *XTCPVisitor) makeNatHole() {
	xl := xlog.FromContextSafe(sv.ctx)
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

	xl.Info("get natHoleRespMsg, sid [%s], candidate address %v, assisted address %v, detectBehavior: %+v",
		natHoleRespMsg.Sid, natHoleRespMsg.CandidateAddrs, natHoleRespMsg.AssistedAddrs, natHoleRespMsg.DetectBehavior)

	newListenConn, raddr, err := nathole.MakeHole(sv.ctx, listenConn, natHoleRespMsg, []byte(sv.cfg.Sk))
	if err != nil {
		listenConn.Close()
		xl.Warn("make hole error: %v", err)
		return
	}
	listenConn = newListenConn
	xl.Info("establishing nat hole connection successful, sid [%s], remoteAddr [%s]", natHoleRespMsg.Sid, raddr)

	if sv.cfg.Protocol == "kcp" {
		kcpSession, err := sv.createKCPSession(listenConn, raddr)
		if err != nil {
			xl.Warn("create kcp session error: %v", err)
			listenConn.Close()
			return
		}
		sv.mu.Lock()
		sv.kcpSession = kcpSession
		sv.mu.Unlock()
		return
	}

	// default is quic
	quicSession, err := sv.createQUICSession(listenConn, raddr)
	if err != nil {
		xl.Warn("create quic session error: %v", err)
		listenConn.Close()
		return
	}
	sv.mu.Lock()
	sv.quicSession = quicSession
	sv.mu.Unlock()
}

func (sv *XTCPVisitor) createKCPSession(listenConn *net.UDPConn, raddr *net.UDPAddr) (*fmux.Session, error) {
	listenConn.Close()
	laddr, _ := net.ResolveUDPAddr("udp", listenConn.LocalAddr().String())
	lConn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		return nil, fmt.Errorf("dial udp error: %v", err)
	}
	remote, err := frpNet.NewKCPConnFromUDP(lConn, true, raddr.String())
	if err != nil {
		return nil, fmt.Errorf("create kcp connection from udp connection error: %v", err)
	}

	fmuxCfg := fmux.DefaultConfig()
	fmuxCfg.KeepAliveInterval = 10 * time.Second
	fmuxCfg.MaxStreamWindowSize = 1024 * 1024
	fmuxCfg.LogOutput = io.Discard
	session, err := fmux.Client(remote, fmuxCfg)
	if err != nil {
		remote.Close()
		return nil, fmt.Errorf("initial client session error: %v", err)
	}
	return session, nil
}

func (sv *XTCPVisitor) createQUICSession(listenConn *net.UDPConn, raddr *net.UDPAddr) (quic.Connection, error) {
	tlsConfig, err := transport.NewClientTLSConfig("", "", "", raddr.String())
	if err != nil {
		return nil, fmt.Errorf("create tls config error: %v", err)
	}
	tlsConfig.NextProtos = []string{"frp"}
	quicConn, err := quic.Dial(listenConn, raddr, raddr.String(), tlsConfig,
		&quic.Config{
			MaxIdleTimeout:     time.Duration(sv.clientCfg.QUICMaxIdleTimeout) * time.Second,
			MaxIncomingStreams: int64(sv.clientCfg.QUICMaxIncomingStreams),
			KeepAlivePeriod:    time.Duration(sv.clientCfg.QUICKeepalivePeriod) * time.Second,
		})
	if err != nil {
		return nil, fmt.Errorf("dial quic error: %v", err)
	}
	return quicConn, nil
}

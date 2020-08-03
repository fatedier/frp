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

package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/models/proto/udp"
	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/util"
	"github.com/fatedier/frp/utils/xlog"

	"github.com/fatedier/golib/errors"
	frpIo "github.com/fatedier/golib/io"
	"github.com/fatedier/golib/pool"
	fmux "github.com/hashicorp/yamux"
)

// Visitor is used for forward traffics from local port tot remote service.
type Visitor interface {
	Run() error
	Close()
}

func NewVisitor(ctx context.Context, ctl *Control, cfg config.VisitorConf) (visitor Visitor) {
	xl := xlog.FromContextSafe(ctx).Spawn().AppendPrefix(cfg.GetBaseInfo().ProxyName)
	baseVisitor := BaseVisitor{
		ctl: ctl,
		ctx: xlog.NewContext(ctx, xl),
	}
	switch cfg := cfg.(type) {
	case *config.StcpVisitorConf:
		visitor = &StcpVisitor{
			BaseVisitor: &baseVisitor,
			cfg:         cfg,
		}
	case *config.XtcpVisitorConf:
		visitor = &XtcpVisitor{
			BaseVisitor: &baseVisitor,
			cfg:         cfg,
		}
	case *config.SudpVisitorConf:
		visitor = &SudpVisitor{
			BaseVisitor:  &baseVisitor,
			cfg:          cfg,
			checkCloseCh: make(chan struct{}),
		}
	}
	return
}

type BaseVisitor struct {
	ctl    *Control
	l      net.Listener
	closed bool

	mu  sync.RWMutex
	ctx context.Context
}

type StcpVisitor struct {
	*BaseVisitor

	cfg *config.StcpVisitorConf
}

func (sv *StcpVisitor) Run() (err error) {
	sv.l, err = net.Listen("tcp", fmt.Sprintf("%s:%d", sv.cfg.BindAddr, sv.cfg.BindPort))
	if err != nil {
		return
	}

	go sv.worker()
	return
}

func (sv *StcpVisitor) Close() {
	sv.l.Close()
}

func (sv *StcpVisitor) worker() {
	xl := xlog.FromContextSafe(sv.ctx)
	for {
		conn, err := sv.l.Accept()
		if err != nil {
			xl.Warn("stcp local listener closed")
			return
		}

		go sv.handleConn(conn)
	}
}

func (sv *StcpVisitor) handleConn(userConn net.Conn) {
	xl := xlog.FromContextSafe(sv.ctx)
	defer userConn.Close()

	xl.Debug("get a new stcp user connection")
	visitorConn, err := sv.ctl.connectServer()
	if err != nil {
		return
	}
	defer visitorConn.Close()

	now := time.Now().Unix()
	newVisitorConnMsg := &msg.NewVisitorConn{
		ProxyName:      sv.cfg.ServerName,
		SignKey:        util.GetAuthKey(sv.cfg.Sk, now),
		Timestamp:      now,
		UseEncryption:  sv.cfg.UseEncryption,
		UseCompression: sv.cfg.UseCompression,
	}
	err = msg.WriteMsg(visitorConn, newVisitorConnMsg)
	if err != nil {
		xl.Warn("send newVisitorConnMsg to server error: %v", err)
		return
	}

	var newVisitorConnRespMsg msg.NewVisitorConnResp
	visitorConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	err = msg.ReadMsgInto(visitorConn, &newVisitorConnRespMsg)
	if err != nil {
		xl.Warn("get newVisitorConnRespMsg error: %v", err)
		return
	}
	visitorConn.SetReadDeadline(time.Time{})

	if newVisitorConnRespMsg.Error != "" {
		xl.Warn("start new visitor connection error: %s", newVisitorConnRespMsg.Error)
		return
	}

	var remote io.ReadWriteCloser
	remote = visitorConn
	if sv.cfg.UseEncryption {
		remote, err = frpIo.WithEncryption(remote, []byte(sv.cfg.Sk))
		if err != nil {
			xl.Error("create encryption stream error: %v", err)
			return
		}
	}

	if sv.cfg.UseCompression {
		remote = frpIo.WithCompression(remote)
	}

	frpIo.Join(userConn, remote)
}

type XtcpVisitor struct {
	*BaseVisitor

	cfg *config.XtcpVisitorConf
}

func (sv *XtcpVisitor) Run() (err error) {
	sv.l, err = net.Listen("tcp", fmt.Sprintf("%s:%d", sv.cfg.BindAddr, sv.cfg.BindPort))
	if err != nil {
		return
	}

	go sv.worker()
	return
}

func (sv *XtcpVisitor) Close() {
	sv.l.Close()
}

func (sv *XtcpVisitor) worker() {
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

func (sv *XtcpVisitor) handleConn(userConn net.Conn) {
	xl := xlog.FromContextSafe(sv.ctx)
	defer userConn.Close()

	xl.Debug("get a new xtcp user connection")
	if sv.ctl.serverUDPPort == 0 {
		xl.Error("xtcp is not supported by server")
		return
	}

	raddr, err := net.ResolveUDPAddr("udp",
		fmt.Sprintf("%s:%d", sv.ctl.clientCfg.ServerAddr, sv.ctl.serverUDPPort))
	if err != nil {
		xl.Error("resolve server UDP addr error")
		return
	}

	visitorConn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		xl.Warn("dial server udp addr error: %v", err)
		return
	}
	defer visitorConn.Close()

	now := time.Now().Unix()
	natHoleVisitorMsg := &msg.NatHoleVisitor{
		ProxyName: sv.cfg.ServerName,
		SignKey:   util.GetAuthKey(sv.cfg.Sk, now),
		Timestamp: now,
	}
	err = msg.WriteMsg(visitorConn, natHoleVisitorMsg)
	if err != nil {
		xl.Warn("send natHoleVisitorMsg to server error: %v", err)
		return
	}

	// Wait for client address at most 10 seconds.
	var natHoleRespMsg msg.NatHoleResp
	visitorConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := pool.GetBuf(1024)
	n, err := visitorConn.Read(buf)
	if err != nil {
		xl.Warn("get natHoleRespMsg error: %v", err)
		return
	}

	err = msg.ReadMsgInto(bytes.NewReader(buf[:n]), &natHoleRespMsg)
	if err != nil {
		xl.Warn("get natHoleRespMsg error: %v", err)
		return
	}
	visitorConn.SetReadDeadline(time.Time{})
	pool.PutBuf(buf)

	if natHoleRespMsg.Error != "" {
		xl.Error("natHoleRespMsg get error info: %s", natHoleRespMsg.Error)
		return
	}

	xl.Trace("get natHoleRespMsg, sid [%s], client address [%s], visitor address [%s]", natHoleRespMsg.Sid, natHoleRespMsg.ClientAddr, natHoleRespMsg.VisitorAddr)

	// Close visitorConn, so we can use it's local address.
	visitorConn.Close()

	// send sid message to client
	laddr, _ := net.ResolveUDPAddr("udp", visitorConn.LocalAddr().String())
	daddr, err := net.ResolveUDPAddr("udp", natHoleRespMsg.ClientAddr)
	if err != nil {
		xl.Error("resolve client udp address error: %v", err)
		return
	}
	lConn, err := net.DialUDP("udp", laddr, daddr)
	if err != nil {
		xl.Error("dial client udp address error: %v", err)
		return
	}
	defer lConn.Close()

	lConn.Write([]byte(natHoleRespMsg.Sid))

	// read ack sid from client
	sidBuf := pool.GetBuf(1024)
	lConn.SetReadDeadline(time.Now().Add(8 * time.Second))
	n, err = lConn.Read(sidBuf)
	if err != nil {
		xl.Warn("get sid from client error: %v", err)
		return
	}
	lConn.SetReadDeadline(time.Time{})
	if string(sidBuf[:n]) != natHoleRespMsg.Sid {
		xl.Warn("incorrect sid from client")
		return
	}
	pool.PutBuf(sidBuf)

	xl.Info("nat hole connection make success, sid [%s]", natHoleRespMsg.Sid)

	// wrap kcp connection
	var remote io.ReadWriteCloser
	remote, err = frpNet.NewKcpConnFromUdp(lConn, true, natHoleRespMsg.ClientAddr)
	if err != nil {
		xl.Error("create kcp connection from udp connection error: %v", err)
		return
	}

	fmuxCfg := fmux.DefaultConfig()
	fmuxCfg.KeepAliveInterval = 5 * time.Second
	fmuxCfg.LogOutput = ioutil.Discard
	sess, err := fmux.Client(remote, fmuxCfg)
	if err != nil {
		xl.Error("create yamux session error: %v", err)
		return
	}
	defer sess.Close()
	muxConn, err := sess.Open()
	if err != nil {
		xl.Error("open yamux stream error: %v", err)
		return
	}

	var muxConnRWCloser io.ReadWriteCloser = muxConn
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

	frpIo.Join(userConn, muxConnRWCloser)
	xl.Debug("join connections closed")
}

type SudpVisitor struct {
	*BaseVisitor

	checkCloseCh chan struct{}
	// udpConn is the listener of udp packet
	udpConn *net.UDPConn
	readCh  chan *msg.UdpPacket
	sendCh  chan *msg.UdpPacket

	cfg *config.SudpVisitorConf
}

// SUDP Run start listen a udp port
func (sv *SudpVisitor) Run() (err error) {
	xl := xlog.FromContextSafe(sv.ctx)

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", sv.cfg.BindAddr, sv.cfg.BindPort))
	if err != nil {
		return fmt.Errorf("sudp ResolveUDPAddr error: %v", err)
	}

	sv.udpConn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("listen udp port %s error: %v", addr.String(), err)
	}

	sv.sendCh = make(chan *msg.UdpPacket, 1024)
	sv.readCh = make(chan *msg.UdpPacket, 1024)

	xl.Info("sudp start to work")

	go sv.dispatcher()
	go udp.ForwardUserConn(sv.udpConn, sv.readCh, sv.sendCh)

	return
}

func (sv *SudpVisitor) dispatcher() {
	xl := xlog.FromContextSafe(sv.ctx)

	for {
		// loop for get frpc to frps tcp conn
		// setup worker
		// wait worker to finished
		// retry or exit
		visitorConn, err := sv.getNewVisitorConn()
		if err != nil {
			// check if proxy is closed
			// if checkCloseCh is close, we will return, other case we will continue to reconnect
			select {
			case <-sv.checkCloseCh:
				xl.Info("frpc sudp visitor proxy is closed")
				return
			default:
			}

			time.Sleep(3 * time.Second)

			xl.Warn("newVisitorConn to frps error: %v, try to reconnect", err)
			continue
		}

		sv.worker(visitorConn)

		select {
		case <-sv.checkCloseCh:
			return
		default:
		}
	}
}

func (sv *SudpVisitor) worker(workConn net.Conn) {
	xl := xlog.FromContextSafe(sv.ctx)
	xl.Debug("starting sudp proxy worker")

	wg := &sync.WaitGroup{}
	wg.Add(2)
	closeCh := make(chan struct{})

	// udp service -> frpc -> frps -> frpc visitor -> user
	workConnReaderFn := func(conn net.Conn) {
		defer func() {
			conn.Close()
			close(closeCh)
			wg.Done()
		}()

		for {
			var (
				rawMsg msg.Message
				errRet error
			)

			// frpc will send heartbeat in workConn to frpc visitor for keeping alive
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			if rawMsg, errRet = msg.ReadMsg(conn); errRet != nil {
				xl.Warn("read from workconn for user udp conn error: %v", errRet)
				return
			}

			conn.SetReadDeadline(time.Time{})
			switch m := rawMsg.(type) {
			case *msg.Ping:
				xl.Debug("frpc visitor get ping message from frpc")
				continue
			case *msg.UdpPacket:
				if errRet := errors.PanicToError(func() {
					sv.readCh <- m
					xl.Trace("frpc visitor get udp packet from frpc")
				}); errRet != nil {
					xl.Info("reader goroutine for udp work connection closed")
					return
				}
			}
		}
	}

	// udp service <- frpc <- frps <- frpc visitor <- user
	workConnSenderFn := func(conn net.Conn) {
		defer func() {
			conn.Close()
			wg.Done()
		}()

		var errRet error
		for {
			select {
			case udpMsg, ok := <-sv.sendCh:
				if !ok {
					xl.Info("sender goroutine for udp work connection closed")
					return
				}

				if errRet = msg.WriteMsg(conn, udpMsg); errRet != nil {
					xl.Warn("sender goroutine for udp work connection closed: %v", errRet)
					return
				}
			case <-closeCh:
				return
			}
		}
	}

	go workConnReaderFn(workConn)
	go workConnSenderFn(workConn)

	wg.Wait()
	xl.Info("sudp worker is closed")
}

func (sv *SudpVisitor) getNewVisitorConn() (visitorConn net.Conn, err error) {
	visitorConn, err = sv.ctl.connectServer()
	if err != nil {
		return nil, fmt.Errorf("frpc connect frps error: %v", err)
	}

	now := time.Now().Unix()
	newVisitorConnMsg := &msg.NewVisitorConn{
		ProxyName:      sv.cfg.ServerName,
		SignKey:        util.GetAuthKey(sv.cfg.Sk, now),
		Timestamp:      now,
		UseEncryption:  sv.cfg.UseEncryption,
		UseCompression: sv.cfg.UseCompression,
	}
	err = msg.WriteMsg(visitorConn, newVisitorConnMsg)
	if err != nil {
		return nil, fmt.Errorf("frpc send newVisitorConnMsg to frps error: %v", err)
	}

	var newVisitorConnRespMsg msg.NewVisitorConnResp
	visitorConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	err = msg.ReadMsgInto(visitorConn, &newVisitorConnRespMsg)
	if err != nil {
		return nil, fmt.Errorf("frpc read newVisitorConnRespMsg error: %v", err)
	}
	visitorConn.SetReadDeadline(time.Time{})

	if newVisitorConnRespMsg.Error != "" {
		return nil, fmt.Errorf("start new visitor connection error: %s", newVisitorConnRespMsg.Error)
	}
	return
}

func (sv *SudpVisitor) Close() {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	select {
	case <-sv.checkCloseCh:
		return
	default:
		close(sv.checkCloseCh)
	}
	if sv.udpConn != nil {
		sv.udpConn.Close()
	}
	close(sv.readCh)
	close(sv.sendCh)
}

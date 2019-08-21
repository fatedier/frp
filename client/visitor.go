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
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/fatedier/frp/g"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/util"

	frpIo "github.com/fatedier/golib/io"
	"github.com/fatedier/golib/pool"
	fmux "github.com/hashicorp/yamux"
)

// Visitor is used for forward traffics from local port tot remote service.
type Visitor interface {
	Run() error
	Close()
	log.Logger
}

func NewVisitor(ctl *Control, cfg config.VisitorConf) (visitor Visitor) {
	baseVisitor := BaseVisitor{
		ctl:    ctl,
		Logger: log.NewPrefixLogger(cfg.GetBaseInfo().ProxyName),
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
	}
	return
}

type BaseVisitor struct {
	ctl    *Control
	l      frpNet.Listener
	closed bool
	mu     sync.RWMutex
	log.Logger
}

type StcpVisitor struct {
	*BaseVisitor

	cfg *config.StcpVisitorConf
}

func (sv *StcpVisitor) Run() (err error) {
	sv.l, err = frpNet.ListenTcp(sv.cfg.BindAddr, sv.cfg.BindPort)
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
	for {
		conn, err := sv.l.Accept()
		if err != nil {
			sv.Warn("stcp local listener closed")
			return
		}

		go sv.handleConn(conn)
	}
}

func (sv *StcpVisitor) handleConn(userConn frpNet.Conn) {
	defer userConn.Close()

	sv.Debug("get a new stcp user connection")
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
		sv.Warn("send newVisitorConnMsg to server error: %v", err)
		return
	}

	var newVisitorConnRespMsg msg.NewVisitorConnResp
	visitorConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	err = msg.ReadMsgInto(visitorConn, &newVisitorConnRespMsg)
	if err != nil {
		sv.Warn("get newVisitorConnRespMsg error: %v", err)
		return
	}
	visitorConn.SetReadDeadline(time.Time{})

	if newVisitorConnRespMsg.Error != "" {
		sv.Warn("start new visitor connection error: %s", newVisitorConnRespMsg.Error)
		return
	}

	var remote io.ReadWriteCloser
	remote = visitorConn
	if sv.cfg.UseEncryption {
		remote, err = frpIo.WithEncryption(remote, []byte(sv.cfg.Sk))
		if err != nil {
			sv.Error("create encryption stream error: %v", err)
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
	sv.l, err = frpNet.ListenTcp(sv.cfg.BindAddr, sv.cfg.BindPort)
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
	for {
		conn, err := sv.l.Accept()
		if err != nil {
			sv.Warn("xtcp local listener closed")
			return
		}

		go sv.handleConn(conn)
	}
}

func (sv *XtcpVisitor) handleConn(userConn frpNet.Conn) {
	defer userConn.Close()

	sv.Debug("get a new xtcp user connection")
	if g.GlbClientCfg.ServerUdpPort == 0 {
		sv.Error("xtcp is not supported by server")
		return
	}

	raddr, err := net.ResolveUDPAddr("udp",
		fmt.Sprintf("%s:%d", g.GlbClientCfg.ServerAddr, g.GlbClientCfg.ServerUdpPort))
	if err != nil {
		sv.Error("resolve server UDP addr error")
		return
	}

	visitorConn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		sv.Warn("dial server udp addr error: %v", err)
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
		sv.Warn("send natHoleVisitorMsg to server error: %v", err)
		return
	}

	// Wait for client address at most 10 seconds.
	var natHoleRespMsg msg.NatHoleResp
	visitorConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := pool.GetBuf(1024)
	n, err := visitorConn.Read(buf)
	if err != nil {
		sv.Warn("get natHoleRespMsg error: %v", err)
		return
	}

	err = msg.ReadMsgInto(bytes.NewReader(buf[:n]), &natHoleRespMsg)
	if err != nil {
		sv.Warn("get natHoleRespMsg error: %v", err)
		return
	}
	visitorConn.SetReadDeadline(time.Time{})
	pool.PutBuf(buf)

	if natHoleRespMsg.Error != "" {
		sv.Error("natHoleRespMsg get error info: %s", natHoleRespMsg.Error)
		return
	}

	sv.Trace("get natHoleRespMsg, sid [%s], client address [%s], visitor address [%s]", natHoleRespMsg.Sid, natHoleRespMsg.ClientAddr, natHoleRespMsg.VisitorAddr)

	// Close visitorConn, so we can use it's local address.
	visitorConn.Close()

	// send sid message to client
	laddr, _ := net.ResolveUDPAddr("udp", visitorConn.LocalAddr().String())
	daddr, err := net.ResolveUDPAddr("udp", natHoleRespMsg.ClientAddr)
	if err != nil {
		sv.Error("resolve client udp address error: %v", err)
		return
	}
	lConn, err := net.DialUDP("udp", laddr, daddr)
	if err != nil {
		sv.Error("dial client udp address error: %v", err)
		return
	}
	defer lConn.Close()

	lConn.Write([]byte(natHoleRespMsg.Sid))

	// read ack sid from client
	sidBuf := pool.GetBuf(1024)
	lConn.SetReadDeadline(time.Now().Add(8 * time.Second))
	n, err = lConn.Read(sidBuf)
	if err != nil {
		sv.Warn("get sid from client error: %v", err)
		return
	}
	lConn.SetReadDeadline(time.Time{})
	if string(sidBuf[:n]) != natHoleRespMsg.Sid {
		sv.Warn("incorrect sid from client")
		return
	}
	pool.PutBuf(sidBuf)

	sv.Info("nat hole connection make success, sid [%s]", natHoleRespMsg.Sid)

	// wrap kcp connection
	var remote io.ReadWriteCloser
	remote, err = frpNet.NewKcpConnFromUdp(lConn, true, natHoleRespMsg.ClientAddr)
	if err != nil {
		sv.Error("create kcp connection from udp connection error: %v", err)
		return
	}

	if sv.cfg.UseEncryption {
		remote, err = frpIo.WithEncryption(remote, []byte(sv.cfg.Sk))
		if err != nil {
			sv.Error("create encryption stream error: %v", err)
			return
		}
	}

	if sv.cfg.UseCompression {
		remote = frpIo.WithCompression(remote)
	}

	fmuxCfg := fmux.DefaultConfig()
	fmuxCfg.KeepAliveInterval = 5 * time.Second
	fmuxCfg.LogOutput = ioutil.Discard
	sess, err := fmux.Client(remote, fmuxCfg)
	if err != nil {
		sv.Error("create yamux session error: %v", err)
		return
	}
	defer sess.Close()
	muxConn, err := sess.Open()
	if err != nil {
		sv.Error("open yamux stream error: %v", err)
		return
	}

	frpIo.Join(userConn, muxConn)
	sv.Debug("join connections closed")
}

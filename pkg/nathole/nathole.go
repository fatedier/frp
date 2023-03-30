// Copyright 2023 The frp Authors
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

package nathole

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/fatedier/golib/crypto"
	"github.com/fatedier/golib/errors"
	"github.com/fatedier/golib/pool"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/util"
)

// NatHoleTimeout seconds.
var NatHoleTimeout int64 = 10

func NewTransactionID() string {
	id, _ := util.RandID()
	return fmt.Sprintf("%d%s", time.Now().Unix(), id)
}

type SidRequest struct {
	Sid      string
	NotifyCh chan struct{}
}

type Controller struct {
	listener *net.UDPConn

	clientCfgs map[string]*ClientCfg
	sessions   map[string]*Session

	encryptionKey []byte
	mu            sync.RWMutex
}

func NewController(udpBindAddr string, encryptionKey []byte) (nc *Controller, err error) {
	addr, err := net.ResolveUDPAddr("udp", udpBindAddr)
	if err != nil {
		return nil, err
	}
	lconn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	nc = &Controller{
		listener:      lconn,
		clientCfgs:    make(map[string]*ClientCfg),
		sessions:      make(map[string]*Session),
		encryptionKey: encryptionKey,
	}
	return nc, nil
}

func (nc *Controller) ListenClient(name string, sk string) (sidCh chan *SidRequest) {
	clientCfg := &ClientCfg{
		Name:  name,
		Sk:    sk,
		SidCh: make(chan *SidRequest),
	}
	nc.mu.Lock()
	nc.clientCfgs[name] = clientCfg
	nc.mu.Unlock()
	return clientCfg.SidCh
}

func (nc *Controller) CloseClient(name string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	delete(nc.clientCfgs, name)
}

func (nc *Controller) Run() {
	for {
		buf := pool.GetBuf(1024)
		n, raddr, err := nc.listener.ReadFromUDP(buf)
		if err != nil {
			log.Warn("nat hole listener read from udp error: %v", err)
			return
		}
		plain, err := crypto.Decode(buf[:n], nc.encryptionKey)
		if err != nil {
			log.Warn("nathole listener decode from %s error: %v", raddr.String(), err)
			continue
		}

		rawMsg, err := msg.ReadMsg(bytes.NewReader(plain))
		if err != nil {
			log.Warn("read nat hole message error: %v", err)
			continue
		}

		switch m := rawMsg.(type) {
		case *msg.NatHoleBinding:
			go nc.HandleBinding(m, raddr)
		case *msg.NatHoleVisitor:
			go nc.HandleVisitor(m, raddr)
		case *msg.NatHoleClient:
			go nc.HandleClient(m, raddr)
		default:
			log.Trace("unknown nat hole message type")
			continue
		}
		pool.PutBuf(buf)
	}
}

func (nc *Controller) GenSid() string {
	t := time.Now().Unix()
	id, _ := util.RandID()
	return fmt.Sprintf("%d%s", t, id)
}

func (nc *Controller) HandleBinding(m *msg.NatHoleBinding, raddr *net.UDPAddr) {
	log.Trace("handle binding message from %s", raddr.String())
	resp := &msg.NatHoleBindingResp{
		TransactionID: m.TransactionID,
		Address:       raddr.String(),
	}
	plain, err := msg.Pack(resp)
	if err != nil {
		log.Error("pack nat hole binding response error: %v", err)
		return
	}
	buf, err := crypto.Encode(plain, nc.encryptionKey)
	if err != nil {
		log.Error("encode nat hole binding response error: %v", err)
		return
	}
	_, err = nc.listener.WriteToUDP(buf, raddr)
	if err != nil {
		log.Error("write nat hole binding response to %s error: %v", raddr.String(), err)
		return
	}
}

func (nc *Controller) HandleVisitor(m *msg.NatHoleVisitor, raddr *net.UDPAddr) {
	sid := nc.GenSid()
	session := &Session{
		Sid:         sid,
		VisitorAddr: raddr,
		NotifyCh:    make(chan struct{}),
	}
	nc.mu.Lock()
	clientCfg, ok := nc.clientCfgs[m.ProxyName]
	if !ok {
		nc.mu.Unlock()
		errInfo := fmt.Sprintf("xtcp server for [%s] doesn't exist", m.ProxyName)
		log.Debug(errInfo)
		_, _ = nc.listener.WriteToUDP(nc.GenNatHoleResponse(nil, errInfo), raddr)
		return
	}
	if m.SignKey != util.GetAuthKey(clientCfg.Sk, m.Timestamp) {
		nc.mu.Unlock()
		errInfo := fmt.Sprintf("xtcp connection of [%s] auth failed", m.ProxyName)
		log.Debug(errInfo)
		_, _ = nc.listener.WriteToUDP(nc.GenNatHoleResponse(nil, errInfo), raddr)
		return
	}

	nc.sessions[sid] = session
	nc.mu.Unlock()
	log.Trace("handle visitor message, sid [%s]", sid)

	defer func() {
		nc.mu.Lock()
		delete(nc.sessions, sid)
		nc.mu.Unlock()
	}()

	err := errors.PanicToError(func() {
		clientCfg.SidCh <- &SidRequest{
			Sid:      sid,
			NotifyCh: session.NotifyCh,
		}
	})
	if err != nil {
		return
	}

	// Wait client connections.
	select {
	case <-session.NotifyCh:
		resp := nc.GenNatHoleResponse(session, "")
		log.Trace("send nat hole response to visitor")
		_, _ = nc.listener.WriteToUDP(resp, raddr)
	case <-time.After(time.Duration(NatHoleTimeout) * time.Second):
		return
	}
}

func (nc *Controller) HandleClient(m *msg.NatHoleClient, raddr *net.UDPAddr) {
	nc.mu.RLock()
	session, ok := nc.sessions[m.Sid]
	nc.mu.RUnlock()
	if !ok {
		return
	}
	log.Trace("handle client message, sid [%s]", session.Sid)
	session.ClientAddr = raddr

	resp := nc.GenNatHoleResponse(session, "")
	log.Trace("send nat hole response to client")
	_, _ = nc.listener.WriteToUDP(resp, raddr)
}

func (nc *Controller) GenNatHoleResponse(session *Session, errInfo string) []byte {
	var (
		sid         string
		visitorAddr string
		clientAddr  string
	)
	if session != nil {
		sid = session.Sid
		visitorAddr = session.VisitorAddr.String()
		clientAddr = session.ClientAddr.String()
	}
	m := &msg.NatHoleResp{
		Sid:         sid,
		VisitorAddr: visitorAddr,
		ClientAddr:  clientAddr,
		Error:       errInfo,
	}
	b := bytes.NewBuffer(nil)
	err := msg.WriteMsg(b, m)
	if err != nil {
		return []byte("")
	}
	return b.Bytes()
}

type Session struct {
	Sid         string
	VisitorAddr *net.UDPAddr
	ClientAddr  *net.UDPAddr

	NotifyCh chan struct{}
}

type ClientCfg struct {
	Name  string
	Sk    string
	SidCh chan *SidRequest
}

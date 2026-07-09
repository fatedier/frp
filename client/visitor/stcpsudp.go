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
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/fatedier/golib/errors"
	libio "github.com/fatedier/golib/io"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/udp"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
)

// Stream tags carried as the first byte of every relayed stream so the provider
// can route it to the TCP or UDP local service. MUST match
// client/proxy/stcpsudp.go.
const (
	stcpsudpStreamTagTCP byte = 0x01
	stcpsudpStreamTagUDP byte = 0x02
)

// STCPSUDPVisitor is the visitor side of the merged secret proxy. It binds BOTH a
// local TCP listener and a local UDP listener on BindAddr:BindPort and reaches the
// provider through the frps relay. Every relayed stream is prefixed with a 1-byte
// tag (outside encryption) so the provider routes it to the TCP or UDP service:
// the TCP path mirrors the stcp visitor, the UDP path mirrors the sudp visitor.
type STCPSUDPVisitor struct {
	*BaseVisitor

	cfg *v1.STCPSUDPVisitorConfig

	// UDP local edge (mirrors SUDPVisitor).
	checkCloseCh chan struct{}
	udpConn      *net.UDPConn
	readCh       chan *msg.UDPPacket
	sendCh       chan *msg.UDPPacket
}

func (sv *STCPSUDPVisitor) Run() (err error) {
	xl := xlog.FromContextSafe(sv.ctx)

	if sv.cfg.BindPort > 0 {
		// local TCP listener
		sv.l, err = net.Listen("tcp", net.JoinHostPort(sv.cfg.BindAddr, strconv.Itoa(sv.cfg.BindPort)))
		if err != nil {
			return
		}
		go sv.acceptLoop(sv.l, "stcp+sudp tcp local", sv.handleTCPConn)

		// local UDP listener (same addr:port)
		var addr *net.UDPAddr
		addr, err = net.ResolveUDPAddr("udp", net.JoinHostPort(sv.cfg.BindAddr, strconv.Itoa(sv.cfg.BindPort)))
		if err != nil {
			return fmt.Errorf("stcp+sudp ResolveUDPAddr error: %v", err)
		}
		sv.udpConn, err = net.ListenUDP("udp", addr)
		if err != nil {
			return fmt.Errorf("stcp+sudp listen udp port %s error: %v", addr.String(), err)
		}
		sv.sendCh = make(chan *msg.UDPPacket, 1024)
		sv.readCh = make(chan *msg.UDPPacket, 1024)
		go sv.udpDispatcher()
		go udp.ForwardUserConn(sv.udpConn, sv.readCh, sv.sendCh, int(sv.clientCfg.UDPPacketSize), nil)

		xl.Infof("stcp+sudp start to work, listen on %s (tcp+udp)", addr)
	}

	// TCP connections redirected from other visitors / plugins.
	go sv.acceptLoop(sv.internalLn, "stcp+sudp internal", sv.handleTCPConn)

	if sv.plugin != nil {
		sv.plugin.Start()
	}
	return
}

func (sv *STCPSUDPVisitor) Close() {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	// Early return makes Close idempotent (it can be called twice on reconnect):
	// falling through would close readCh/sendCh a second time and panic.
	select {
	case <-sv.checkCloseCh:
		return
	default:
		close(sv.checkCloseCh)
	}
	sv.BaseVisitor.Close()
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

// ---------------- TCP path (tag 0x01) — mirrors STCPVisitor ----------------

func (sv *STCPSUDPVisitor) handleTCPConn(userConn net.Conn) {
	xl := xlog.FromContextSafe(sv.ctx)
	defer userConn.Close()

	xl.Debugf("get a new stcp+sudp tcp user connection")
	visitorConn, err := sv.dialRawVisitorConn(sv.cfg.GetBaseConfig())
	if err != nil {
		xl.Warnf("dialRawVisitorConn error: %v", err)
		return
	}
	defer visitorConn.Close()

	// tag this stream as TCP before any encryption wrapping
	if _, err := visitorConn.Write([]byte{stcpsudpStreamTagTCP}); err != nil {
		xl.Warnf("write tcp stream tag error: %v", err)
		return
	}

	remote, recycleFn, err := wrapVisitorConn(visitorConn, sv.cfg.GetBaseConfig())
	if err != nil {
		xl.Warnf("wrapVisitorConn error: %v", err)
		return
	}
	defer recycleFn()

	libio.Join(userConn, remote)
}

// ---------------- UDP path (tag 0x02) — mirrors SUDPVisitor ----------------

func (sv *STCPSUDPVisitor) udpDispatcher() {
	xl := xlog.FromContextSafe(sv.ctx)

	var (
		visitorConn net.Conn
		recycleFn   func()
		err         error

		firstPacket *msg.UDPPacket
	)

	for {
		select {
		case firstPacket = <-sv.sendCh:
			if firstPacket == nil {
				xl.Infof("frpc stcp+sudp visitor proxy is closed")
				return
			}
		case <-sv.checkCloseCh:
			xl.Infof("frpc stcp+sudp visitor proxy is closed")
			return
		}

		visitorConn, recycleFn, err = sv.getNewUDPVisitorConn()
		if err != nil {
			xl.Warnf("newVisitorConn to frps error: %v, try to reconnect", err)
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

func (sv *STCPSUDPVisitor) getNewUDPVisitorConn() (net.Conn, func(), error) {
	rawConn, err := sv.dialRawVisitorConn(sv.cfg.GetBaseConfig())
	if err != nil {
		return nil, func() {}, err
	}
	// tag this stream as UDP before any encryption wrapping
	if _, err := rawConn.Write([]byte{stcpsudpStreamTagUDP}); err != nil {
		rawConn.Close()
		return nil, func() {}, err
	}
	rwc, recycleFn, err := wrapVisitorConn(rawConn, sv.cfg.GetBaseConfig())
	if err != nil {
		rawConn.Close()
		return nil, func() {}, err
	}
	return netpkg.WrapReadWriteCloserToConn(rwc, rawConn), recycleFn, nil
}

func (sv *STCPSUDPVisitor) udpWorker(workConn net.Conn, firstPacket *msg.UDPPacket) {
	xl := xlog.FromContextSafe(sv.ctx)
	xl.Debugf("starting stcp+sudp udp proxy worker")
	payloadConn := msg.NewConn(workConn, msg.NewReadWriter(workConn, sv.clientCfg.Transport.WireProtocol))

	wg := &sync.WaitGroup{}
	wg.Add(2)
	closeCh := make(chan struct{})

	// udp service -> frpc -> frps -> frpc visitor -> user
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
				xl.Debugf("frpc visitor get ping message from frpc")
				continue
			case *msg.UDPPacket:
				if errRet := errors.PanicToError(func() {
					sv.readCh <- m
				}); errRet != nil {
					xl.Infof("reader goroutine for udp work connection closed")
					return
				}
			}
		}
	}

	// udp service <- frpc <- frps <- frpc visitor <- user
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
			case <-closeCh:
				return
			}
		}
	}

	go workConnReaderFn(payloadConn)
	go workConnSenderFn(payloadConn)

	wg.Wait()
	xl.Infof("stcp+sudp udp worker is closed")
}

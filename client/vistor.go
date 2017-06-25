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
	"io"
	"sync"
	"time"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	frpIo "github.com/fatedier/frp/utils/io"
	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/util"
)

// Vistor is used for forward traffics from local port tot remote service.
type Vistor interface {
	Run() error
	Close()
	log.Logger
}

func NewVistor(ctl *Control, pxyConf config.ProxyConf) (vistor Vistor) {
	baseVistor := BaseVistor{
		ctl:    ctl,
		Logger: log.NewPrefixLogger(pxyConf.GetName()),
	}
	switch cfg := pxyConf.(type) {
	case *config.StcpProxyConf:
		vistor = &StcpVistor{
			BaseVistor: baseVistor,
			cfg:        cfg,
		}
	}
	return
}

type BaseVistor struct {
	ctl    *Control
	l      frpNet.Listener
	closed bool
	mu     sync.RWMutex
	log.Logger
}

type StcpVistor struct {
	BaseVistor

	cfg *config.StcpProxyConf
}

func (sv *StcpVistor) Run() (err error) {
	sv.l, err = frpNet.ListenTcp(sv.cfg.BindAddr, int64(sv.cfg.BindPort))
	if err != nil {
		return
	}

	go sv.worker()
	return
}

func (sv *StcpVistor) Close() {
	sv.l.Close()
}

func (sv *StcpVistor) worker() {
	for {
		conn, err := sv.l.Accept()
		if err != nil {
			sv.Warn("stcp local listener closed")
			return
		}

		go sv.handleConn(conn)
	}
}

func (sv *StcpVistor) handleConn(userConn frpNet.Conn) {
	defer userConn.Close()

	sv.Debug("get a new stcp user connection")
	vistorConn, err := sv.ctl.connectServer()
	if err != nil {
		return
	}
	defer vistorConn.Close()

	now := time.Now().Unix()
	newVistorConnMsg := &msg.NewVistorConn{
		ProxyName:      sv.cfg.ServerName,
		SignKey:        util.GetAuthKey(sv.cfg.Sk, now),
		Timestamp:      now,
		UseEncryption:  sv.cfg.UseEncryption,
		UseCompression: sv.cfg.UseCompression,
	}
	err = msg.WriteMsg(vistorConn, newVistorConnMsg)
	if err != nil {
		sv.Warn("send newVistorConnMsg to server error: %v", err)
		return
	}

	var newVistorConnRespMsg msg.NewVistorConnResp
	vistorConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	err = msg.ReadMsgInto(vistorConn, &newVistorConnRespMsg)
	if err != nil {
		sv.Warn("get newVistorConnRespMsg error: %v", err)
		return
	}
	vistorConn.SetReadDeadline(time.Time{})

	if newVistorConnRespMsg.Error != "" {
		sv.Warn("start new vistor connection error: %s", newVistorConnRespMsg.Error)
		return
	}

	var remote io.ReadWriteCloser
	remote = vistorConn
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

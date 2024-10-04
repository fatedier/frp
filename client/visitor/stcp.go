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
	"io"
	"net"
	"strconv"
	"time"

	libio "github.com/fatedier/golib/io"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type STCPVisitor struct {
	*BaseVisitor

	cfg *v1.STCPVisitorConfig
}

func (sv *STCPVisitor) Run() (err error) {
	if sv.cfg.BindPort > 0 {
		sv.l, err = net.Listen("tcp", net.JoinHostPort(sv.cfg.BindAddr, strconv.Itoa(sv.cfg.BindPort)))
		if err != nil {
			return
		}
		go sv.worker()
	}

	go sv.internalConnWorker()
	return
}

func (sv *STCPVisitor) Close() {
	sv.BaseVisitor.Close()
}

func (sv *STCPVisitor) worker() {
	xl := xlog.FromContextSafe(sv.ctx)
	for {
		conn, err := sv.l.Accept()
		if err != nil {
			xl.Warnf("STCP本地监听关闭")
			return
		}
		go sv.handleConn(conn)
	}
}

func (sv *STCPVisitor) internalConnWorker() {
	xl := xlog.FromContextSafe(sv.ctx)
	for {
		conn, err := sv.internalLn.Accept()
		if err != nil {
			xl.Warnf("STCP互联网监听关闭")
			return
		}
		go sv.handleConn(conn)
	}
}

func (sv *STCPVisitor) handleConn(userConn net.Conn) {
	xl := xlog.FromContextSafe(sv.ctx)
	defer userConn.Close()

	xl.Debugf("获取到一个新的STCP用户链接.")
	visitorConn, err := sv.helper.ConnectServer()
	if err != nil {
		return
	}
	defer visitorConn.Close()

	now := time.Now().Unix()
	newVisitorConnMsg := &msg.NewVisitorConn{
		RunID:          sv.helper.RunID(),
		ProxyName:      sv.cfg.ServerName,
		SignKey:        util.GetAuthKey(sv.cfg.SecretKey, now),
		Timestamp:      now,
		UseEncryption:  sv.cfg.Transport.UseEncryption,
		UseCompression: sv.cfg.Transport.UseCompression,
	}
	err = msg.WriteMsg(visitorConn, newVisitorConnMsg)
	if err != nil {
		xl.Warnf("发生新参与者链接信息到服务器时发生错误: %v", err)
		return
	}

	var newVisitorConnRespMsg msg.NewVisitorConnResp
	_ = visitorConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	err = msg.ReadMsgInto(visitorConn, &newVisitorConnRespMsg)
	if err != nil {
		xl.Warnf("获取新参与者链接信息时发生错误: %v", err)
		return
	}
	_ = visitorConn.SetReadDeadline(time.Time{})

	if newVisitorConnRespMsg.Error != "" {
		xl.Warnf("启动一个新的参与者链接时发生错误: %s", newVisitorConnRespMsg.Error)
		return
	}

	var remote io.ReadWriteCloser
	remote = visitorConn
	if sv.cfg.Transport.UseEncryption {
		remote, err = libio.WithEncryption(remote, []byte(sv.cfg.SecretKey))
		if err != nil {
			xl.Errorf("创建流线型加密连接时出现错误: %v", err)
			return
		}
	}

	if sv.cfg.Transport.UseCompression {
		var recycleFn func()
		remote, recycleFn = libio.WithCompressionFromPool(remote)
		defer recycleFn()
	}

	libio.Join(userConn, remote)
}

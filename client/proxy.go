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
	"fmt"
	"io"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/proto/tcp"
	"github.com/fatedier/frp/utils/net"
)

// Proxy defines how to work for different proxy type.
type Proxy interface {
	Run() error

	// InWorkConn accept work connections registered to server.
	InWorkConn(conn net.Conn)
	Close()
}

func NewProxy(ctl *Control, pxyConf config.ProxyConf) (pxy Proxy) {
	switch cfg := pxyConf.(type) {
	case *config.TcpProxyConf:
		pxy = &TcpProxy{
			cfg: cfg,
			ctl: ctl,
		}
	case *config.UdpProxyConf:
		pxy = &UdpProxy{
			cfg: cfg,
			ctl: ctl,
		}
	case *config.HttpProxyConf:
		pxy = &HttpProxy{
			cfg: cfg,
			ctl: ctl,
		}
	case *config.HttpsProxyConf:
		pxy = &HttpsProxy{
			cfg: cfg,
			ctl: ctl,
		}
	}
	return
}

// TCP
type TcpProxy struct {
	cfg *config.TcpProxyConf
	ctl *Control
}

func (pxy *TcpProxy) Run() (err error) {
	return
}

func (pxy *TcpProxy) Close() {
}

func (pxy *TcpProxy) InWorkConn(conn net.Conn) {
	defer conn.Close()
	HandleTcpWorkConnection(&pxy.cfg.LocalSvrConf, &pxy.cfg.BaseProxyConf, conn)
}

// HTTP
type HttpProxy struct {
	cfg *config.HttpProxyConf
	ctl *Control
}

func (pxy *HttpProxy) Run() (err error) {
	return
}

func (pxy *HttpProxy) Close() {
}

func (pxy *HttpProxy) InWorkConn(conn net.Conn) {
	defer conn.Close()
	HandleTcpWorkConnection(&pxy.cfg.LocalSvrConf, &pxy.cfg.BaseProxyConf, conn)
}

// HTTPS
type HttpsProxy struct {
	cfg *config.HttpsProxyConf
	ctl *Control
}

func (pxy *HttpsProxy) Run() (err error) {
	return
}

func (pxy *HttpsProxy) Close() {
}

func (pxy *HttpsProxy) InWorkConn(conn net.Conn) {
	defer conn.Close()
	HandleTcpWorkConnection(&pxy.cfg.LocalSvrConf, &pxy.cfg.BaseProxyConf, conn)
}

// UDP
type UdpProxy struct {
	cfg *config.UdpProxyConf
	ctl *Control
}

func (pxy *UdpProxy) Run() (err error) {
	return
}

func (pxy *UdpProxy) Close() {
}

func (pxy *UdpProxy) InWorkConn(conn net.Conn) {
	defer conn.Close()
}

// Common handler for tcp work connections.
func HandleTcpWorkConnection(localInfo *config.LocalSvrConf, baseInfo *config.BaseProxyConf, workConn net.Conn) {
	localConn, err := net.ConnectTcpServer(fmt.Sprintf("%s:%d", localInfo.LocalIp, localInfo.LocalPort))
	if err != nil {
		workConn.Error("connect to local service [%s:%d] error: %v", localInfo.LocalIp, localInfo.LocalPort, err)
		return
	}

	var remote io.ReadWriteCloser
	remote = workConn
	if baseInfo.UseEncryption {
		remote, err = tcp.WithEncryption(remote, []byte(config.ClientCommonCfg.PrivilegeToken))
		if err != nil {
			workConn.Error("create encryption stream error: %v", err)
			return
		}
	}
	if baseInfo.UseCompression {
		remote = tcp.WithCompression(remote)
	}
	workConn.Debug("join connections, localConn(l[%s] r[%s]) workConn(l[%s] r[%s])", localConn.LocalAddr().String(),
		localConn.RemoteAddr().String(), workConn.LocalAddr().String(), workConn.RemoteAddr().String())
	tcp.Join(localConn, remote)
	workConn.Debug("join connections closed")
}

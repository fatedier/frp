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

type Proxy interface {
	Run()
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

func (pxy *TcpProxy) Run() {
}

func (pxy *TcpProxy) Close() {
}

func (pxy *TcpProxy) InWorkConn(conn net.Conn) {
	defer conn.Close()
	localConn, err := net.ConnectTcpServer(fmt.Sprintf("%s:%d", pxy.cfg.LocalIp, pxy.cfg.LocalPort))
	if err != nil {
		conn.Error("connect to local service [%s:%d] error: %v", pxy.cfg.LocalIp, pxy.cfg.LocalPort, err)
		return
	}

	var remote io.ReadWriteCloser
	remote = conn
	if pxy.cfg.UseEncryption {
		remote, err = tcp.WithEncryption(remote, []byte(config.ClientCommonCfg.PrivilegeToken))
		if err != nil {
			conn.Error("create encryption stream error: %v", err)
			return
		}
	}
	if pxy.cfg.UseCompression {
		remote = tcp.WithCompression(remote)
	}
	conn.Debug("join connections")
	tcp.Join(localConn, remote)
	conn.Debug("join connections closed")
}

// UDP
type UdpProxy struct {
	cfg *config.UdpProxyConf
	ctl *Control
}

func (pxy *UdpProxy) Run() {
}

func (pxy *UdpProxy) Close() {
}

func (pxy *UdpProxy) InWorkConn(conn net.Conn) {
	defer conn.Close()
}

// HTTP
type HttpProxy struct {
	cfg *config.HttpProxyConf
	ctl *Control
}

func (pxy *HttpProxy) Run() {
}

func (pxy *HttpProxy) Close() {
}

func (pxy *HttpProxy) InWorkConn(conn net.Conn) {
	defer conn.Close()
}

// HTTPS
type HttpsProxy struct {
	cfg *config.HttpsProxyConf
	ctl *Control
}

func (pxy *HttpsProxy) Run() {
}

func (pxy *HttpsProxy) Close() {
}

func (pxy *HttpsProxy) InWorkConn(conn net.Conn) {
	defer conn.Close()
}

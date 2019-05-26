// Copyright 2019 fatedier, fatedier@gmail.com
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

package proxy

import (
	"fmt"

	"github.com/fatedier/frp/g"
	"github.com/fatedier/frp/models/config"
	frpNet "github.com/fatedier/frp/utils/net"
)

type TcpProxy struct {
	*BaseProxy
	cfg *config.TcpProxyConf

	realPort int
}

func (pxy *TcpProxy) Run() (remoteAddr string, err error) {
	if pxy.cfg.Group != "" {
		l, realPort, errRet := pxy.rc.TcpGroupCtl.Listen(pxy.name, pxy.cfg.Group, pxy.cfg.GroupKey, g.GlbServerCfg.ProxyBindAddr, pxy.cfg.RemotePort)
		if errRet != nil {
			err = errRet
			return
		}
		defer func() {
			if err != nil {
				l.Close()
			}
		}()
		pxy.realPort = realPort
		listener := frpNet.WrapLogListener(l)
		listener.AddLogPrefix(pxy.name)
		pxy.listeners = append(pxy.listeners, listener)
		pxy.Info("tcp proxy listen port [%d] in group [%s]", pxy.cfg.RemotePort, pxy.cfg.Group)
	} else {
		pxy.realPort, err = pxy.rc.TcpPortManager.Acquire(pxy.name, pxy.cfg.RemotePort)
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				pxy.rc.TcpPortManager.Release(pxy.realPort)
			}
		}()
		listener, errRet := frpNet.ListenTcp(g.GlbServerCfg.ProxyBindAddr, pxy.realPort)
		if errRet != nil {
			err = errRet
			return
		}
		listener.AddLogPrefix(pxy.name)
		pxy.listeners = append(pxy.listeners, listener)
		pxy.Info("tcp proxy listen port [%d]", pxy.cfg.RemotePort)
	}

	pxy.cfg.RemotePort = pxy.realPort
	remoteAddr = fmt.Sprintf(":%d", pxy.realPort)
	pxy.startListenHandler(pxy, HandleUserTcpConnection)
	return
}

func (pxy *TcpProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *TcpProxy) Close() {
	pxy.BaseProxy.Close()
	if pxy.cfg.Group == "" {
		pxy.rc.TcpPortManager.Release(pxy.realPort)
	}
}

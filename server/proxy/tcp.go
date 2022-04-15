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
	"net"
	"strconv"

	"github.com/fatedier/frp/pkg/config"
)

type TCPProxy struct {
	*BaseProxy
	cfg *config.TCPProxyConf

	realPort int
}

func (pxy *TCPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl
	if pxy.cfg.Group != "" {
		l, realPort, errRet := pxy.rc.TCPGroupCtl.Listen(pxy.name, pxy.cfg.Group, pxy.cfg.GroupKey, pxy.serverCfg.ProxyBindAddr, pxy.cfg.RemotePort)
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
		pxy.listeners = append(pxy.listeners, l)
		xl.Info("tcp proxy listen port [%d] in group [%s]", pxy.cfg.RemotePort, pxy.cfg.Group)
	} else {
		pxy.realPort, err = pxy.rc.TCPPortManager.Acquire(pxy.name, pxy.cfg.RemotePort)
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				pxy.rc.TCPPortManager.Release(pxy.realPort)
			}
		}()
		listener, errRet := net.Listen("tcp", net.JoinHostPort(pxy.serverCfg.ProxyBindAddr, strconv.Itoa(pxy.realPort)))
		if errRet != nil {
			err = errRet
			return
		}
		pxy.listeners = append(pxy.listeners, listener)
		xl.Info("tcp proxy listen port [%d]", pxy.cfg.RemotePort)
	}

	pxy.cfg.RemotePort = pxy.realPort
	remoteAddr = fmt.Sprintf(":%d", pxy.realPort)
	pxy.startListenHandler(pxy, HandleUserTCPConnection)
	return
}

func (pxy *TCPProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *TCPProxy) Close() {
	pxy.BaseProxy.Close()
	if pxy.cfg.Group == "" {
		pxy.rc.TCPPortManager.Release(pxy.realPort)
	}
}

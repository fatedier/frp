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
	"reflect"
	"strconv"

	"github.com/fatedier/frp/pkg/config"
)

func init() {
	RegisterProxyFactory(reflect.TypeOf(&config.TCPProxyConf{}), NewTCPProxy)
}

type TCPProxy struct {
	*BaseProxy
	cfg *config.TCPProxyConf

	realBindPort int
}

func NewTCPProxy(baseProxy *BaseProxy, cfg config.ProxyConf) Proxy {
	unwrapped, ok := cfg.(*config.TCPProxyConf)
	if !ok {
		return nil
	}
	baseProxy.usedPortsNum = 1
	return &TCPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

func (pxy *TCPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl
	if pxy.cfg.Group != "" {
		l, realBindPort, errRet := pxy.rc.TCPGroupCtl.Listen(pxy.name, pxy.cfg.Group, pxy.cfg.GroupKey, pxy.serverCfg.ProxyBindAddr, pxy.cfg.RemotePort)
		if errRet != nil {
			err = errRet
			return
		}
		defer func() {
			if err != nil {
				l.Close()
			}
		}()
		pxy.realBindPort = realBindPort
		pxy.listeners = append(pxy.listeners, l)
		xl.Info("tcp proxy listen port [%d] in group [%s]", pxy.cfg.RemotePort, pxy.cfg.Group)
	} else {
		pxy.realBindPort, err = pxy.rc.TCPPortManager.Acquire(pxy.name, pxy.cfg.RemotePort)
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				pxy.rc.TCPPortManager.Release(pxy.realBindPort)
			}
		}()
		listener, errRet := net.Listen("tcp", net.JoinHostPort(pxy.serverCfg.ProxyBindAddr, strconv.Itoa(pxy.realBindPort)))
		if errRet != nil {
			err = errRet
			return
		}
		pxy.listeners = append(pxy.listeners, listener)
		xl.Info("tcp proxy listen port [%d]", pxy.cfg.RemotePort)
	}

	pxy.cfg.RemotePort = pxy.realBindPort
	remoteAddr = fmt.Sprintf(":%d", pxy.realBindPort)
	pxy.startCommonTCPListenersHandler()
	return
}

func (pxy *TCPProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *TCPProxy) Close() {
	pxy.BaseProxy.Close()
	if pxy.cfg.Group == "" {
		pxy.rc.TCPPortManager.Release(pxy.realBindPort)
	}
}

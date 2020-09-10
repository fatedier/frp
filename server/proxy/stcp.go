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
	"github.com/fatedier/frp/models/config"
)

type StcpProxy struct {
	*BaseProxy
	cfg *config.StcpProxyConf
}

func (pxy *StcpProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl
	listener, errRet := pxy.rc.VisitorManager.Listen(pxy.GetName(), pxy.cfg.Sk)
	if errRet != nil {
		err = errRet
		return
	}
	pxy.listeners = append(pxy.listeners, listener)
	xl.Info("stcp proxy custom listen success")

	pxy.startListenHandler(pxy, HandleUserTcpConnection)
	return
}

func (pxy *StcpProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *StcpProxy) Close() {
	pxy.BaseProxy.Close()
	pxy.rc.VisitorManager.CloseListener(pxy.GetName())
}

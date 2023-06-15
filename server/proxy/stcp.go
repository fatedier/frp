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
	"reflect"

	"github.com/fatedier/frp/pkg/config"
)

func init() {
	RegisterProxyFactory(reflect.TypeOf(&config.STCPProxyConf{}), NewSTCPProxy)
}

type STCPProxy struct {
	*BaseProxy
	cfg *config.STCPProxyConf
}

func NewSTCPProxy(baseProxy *BaseProxy, cfg config.ProxyConf) Proxy {
	unwrapped, ok := cfg.(*config.STCPProxyConf)
	if !ok {
		return nil
	}
	return &STCPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

func (pxy *STCPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl
	allowUsers := pxy.cfg.AllowUsers
	// if allowUsers is empty, only allow same user from proxy
	if len(allowUsers) == 0 {
		allowUsers = []string{pxy.GetUserInfo().User}
	}
	listener, errRet := pxy.rc.VisitorManager.Listen(pxy.GetName(), pxy.cfg.Sk, allowUsers)
	if errRet != nil {
		err = errRet
		return
	}
	pxy.listeners = append(pxy.listeners, listener)
	xl.Info("stcp proxy custom listen success")

	pxy.startCommonTCPListenersHandler()
	return
}

func (pxy *STCPProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *STCPProxy) Close() {
	pxy.BaseProxy.Close()
	pxy.rc.VisitorManager.CloseListener(pxy.GetName())
}

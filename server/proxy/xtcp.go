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
	"reflect"

	"github.com/fatedier/golib/errors"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/msg"
)

func init() {
	RegisterProxyFactory(reflect.TypeOf(&config.XTCPProxyConf{}), NewXTCPProxy)
}

type XTCPProxy struct {
	*BaseProxy
	cfg *config.XTCPProxyConf

	closeCh chan struct{}
}

func NewXTCPProxy(baseProxy *BaseProxy, cfg config.ProxyConf) Proxy {
	unwrapped, ok := cfg.(*config.XTCPProxyConf)
	if !ok {
		return nil
	}
	return &XTCPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

func (pxy *XTCPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl

	if pxy.rc.NatHoleController == nil {
		err = fmt.Errorf("xtcp is not supported in frps")
		return
	}
	allowUsers := pxy.cfg.AllowUsers
	// if allowUsers is empty, only allow same user from proxy
	if len(allowUsers) == 0 {
		allowUsers = []string{pxy.GetUserInfo().User}
	}
	sidCh, err := pxy.rc.NatHoleController.ListenClient(pxy.GetName(), pxy.cfg.Sk, allowUsers)
	if err != nil {
		return "", err
	}
	go func() {
		for {
			select {
			case <-pxy.closeCh:
				return
			case sid := <-sidCh:
				workConn, errRet := pxy.GetWorkConnFromPool(nil, nil)
				if errRet != nil {
					continue
				}
				m := &msg.NatHoleSid{
					Sid: sid,
				}
				errRet = msg.WriteMsg(workConn, m)
				if errRet != nil {
					xl.Warn("write nat hole sid package error, %v", errRet)
				}
				workConn.Close()
			}
		}
	}()
	return
}

func (pxy *XTCPProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *XTCPProxy) Close() {
	pxy.BaseProxy.Close()
	pxy.rc.NatHoleController.CloseClient(pxy.GetName())
	_ = errors.PanicToError(func() {
		close(pxy.closeCh)
	})
}

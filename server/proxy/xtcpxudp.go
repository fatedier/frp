// Copyright 2025 The frp Authors
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
	"sync"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// The frps side of xtcpxudp is identical to xtcp/xudp: frps only brokers the NAT
// hole signaling and never sees the payload. writeNatHoleSid is shared with the
// xtcp server proxy (same package).

func init() {
	RegisterProxyFactory(reflect.TypeFor[*v1.XTCPXUDPProxyConfig](), NewXTCPXUDPProxy)
}

type XTCPXUDPProxy struct {
	*BaseProxy
	cfg *v1.XTCPXUDPProxyConfig

	closeCh   chan struct{}
	closeOnce sync.Once
}

func NewXTCPXUDPProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.XTCPXUDPProxyConfig)
	if !ok {
		return nil
	}
	return &XTCPXUDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
		closeCh:   make(chan struct{}),
	}
}

func (pxy *XTCPXUDPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl

	if pxy.rc.NatHoleController == nil {
		err = fmt.Errorf("xtcpxudp is not supported in frps")
		return
	}
	allowUsers := pxy.cfg.AllowUsers
	// if allowUsers is empty, only allow same user from proxy
	if len(allowUsers) == 0 {
		allowUsers = []string{pxy.GetUserInfo().User}
	}
	sidCh, err := pxy.rc.NatHoleController.ListenClient(pxy.GetName(), pxy.cfg.Secretkey, allowUsers)
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
				errRet = writeNatHoleSid(workConn, pxy.wireProtocol, sid)
				if errRet != nil {
					xl.Warnf("write nat hole sid package error, %v", errRet)
				}
				workConn.Close()
			}
		}
	}()
	return
}

func (pxy *XTCPXUDPProxy) Close() {
	pxy.closeOnce.Do(func() {
		pxy.BaseProxy.Close()
		pxy.rc.NatHoleController.CloseClient(pxy.GetName())
		close(pxy.closeCh)
	})
}

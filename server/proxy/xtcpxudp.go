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
	"github.com/fatedier/frp/pkg/msg"
)

// xtcpxudp brokers NAT hole signaling like xtcp/xudp AND, so the visitor can
// automatically fall back to the frps relay when hole punching fails, registers a
// secret visitor listener identical to stcp/sudp. A hole-punch trigger work conn
// is tagged Protocol="nathole"; relay work conns keep the empty default and carry
// the same 1-byte-tagged streams as stcp+sudp. writeNatHoleSid is shared with the
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

	// Relay fallback path: a secret visitor listener (same as stcp/sudp) so the
	// visitor can reach this provider through frps when hole punching fails.
	if err = pxy.startVisitorListener(pxy.cfg.Secretkey, allowUsers, "xtcp+xudp"); err != nil {
		return "", err
	}

	// P2P path: NAT hole signaling.
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
				// Tag this work conn as a hole-punch trigger so the client runs
				// the nathole handshake instead of treating it as relay payload.
				workConn, errRet := pxy.GetWorkConnFromPoolWithProtocol(nil, nil, msg.StartWorkConnProtocolNatHole)
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
		pxy.rc.VisitorManager.CloseListener(pxy.GetName())
		close(pxy.closeCh)
	})
}

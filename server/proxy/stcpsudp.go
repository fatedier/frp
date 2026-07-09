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
	"reflect"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func init() {
	RegisterProxyFactory(reflect.TypeFor[*v1.STCPSUDPProxyConfig](), NewSTCPSUDPProxy)
}

// STCPSUDPProxy is the merged secret proxy. On the frps side it is identical to
// stcp/sudp: a single secret visitor listener. The TCP vs UDP split lives entirely
// on the client — each relayed stream is prefixed with a 1-byte tag that the
// client provider reads to pick the local TCP or UDP service.
type STCPSUDPProxy struct {
	*BaseProxy
	cfg *v1.STCPSUDPProxyConfig
}

func NewSTCPSUDPProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.STCPSUDPProxyConfig)
	if !ok {
		return nil
	}
	return &STCPSUDPProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

func (pxy *STCPSUDPProxy) Run() (remoteAddr string, err error) {
	err = pxy.startVisitorListener(pxy.cfg.Secretkey, pxy.cfg.AllowUsers, "stcp+sudp")
	return
}

func (pxy *STCPSUDPProxy) Close() {
	pxy.BaseProxy.Close()
	pxy.rc.VisitorManager.CloseListener(pxy.GetName())
}

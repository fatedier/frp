// Copyright 2024 The frp Authors
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

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/vhost"
)

func init() {
	RegisterProxyFactory(reflect.TypeFor[*v1.MCProxyConfig](), NewMCProxy)
}

// MCProxy routes Minecraft (Java Edition) traffic by the hostname carried in the
// client's first handshake packet, the same way HTTPSProxy routes by TLS SNI.
// The public port is declared client-side (cfg.RemotePort); frps opens it lazily
// and shares one listener among all mc proxies on the same port.
type MCProxy struct {
	*BaseProxy
	cfg *v1.MCProxyConfig
}

func NewMCProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.MCProxyConfig)
	if !ok {
		return nil
	}
	return &MCProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

func (pxy *MCProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl

	defer func() {
		if err != nil {
			pxy.Close()
		}
	}()
	domains := pxy.buildDomains(pxy.cfg.CustomDomains, pxy.cfg.SubDomain)

	for _, domain := range domains {
		routeConfig := vhost.RouteConfig{Domain: domain}

		l, errListen := pxy.rc.MinecraftGroupCtl.Listen(pxy.ctx, pxy.cfg.RemotePort, routeConfig)
		if errListen != nil {
			err = errListen
			return "", err
		}
		pxy.listeners = append(pxy.listeners, l)
		xl.Infof("minecraft proxy listen for host [%s] on port [%d]", domain, pxy.cfg.RemotePort)
	}

	pxy.startCommonTCPListenersHandler()
	remoteAddr = fmt.Sprintf(":%d", pxy.cfg.RemotePort)
	return
}

func (pxy *MCProxy) Close() {
	pxy.BaseProxy.Close()
}

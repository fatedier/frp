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
	"strings"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/vhost"
)

type HTTPSProxy struct {
	*BaseProxy
	cfg *config.HTTPSProxyConf
}

func (pxy *HTTPSProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl
	routeConfig := &vhost.RouteConfig{}

	defer func() {
		if err != nil {
			pxy.Close()
		}
	}()
	addrs := make([]string, 0)
	for _, domain := range pxy.cfg.CustomDomains {
		if domain == "" {
			continue
		}

		routeConfig.Domain = domain
		l, errRet := pxy.rc.VhostHTTPSMuxer.Listen(pxy.ctx, routeConfig)
		if errRet != nil {
			err = errRet
			return
		}
		xl.Info("https proxy listen for host [%s]", routeConfig.Domain)
		pxy.listeners = append(pxy.listeners, l)
		addrs = append(addrs, util.CanonicalAddr(routeConfig.Domain, pxy.serverCfg.VhostHTTPSPort))
	}

	if pxy.cfg.SubDomain != "" {
		routeConfig.Domain = pxy.cfg.SubDomain + "." + pxy.serverCfg.SubDomainHost
		l, errRet := pxy.rc.VhostHTTPSMuxer.Listen(pxy.ctx, routeConfig)
		if errRet != nil {
			err = errRet
			return
		}
		xl.Info("https proxy listen for host [%s]", routeConfig.Domain)
		pxy.listeners = append(pxy.listeners, l)
		addrs = append(addrs, util.CanonicalAddr(routeConfig.Domain, pxy.serverCfg.VhostHTTPSPort))
	}

	pxy.startListenHandler(pxy, HandleUserTCPConnection)
	remoteAddr = strings.Join(addrs, ",")
	return
}

func (pxy *HTTPSProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *HTTPSProxy) Close() {
	pxy.BaseProxy.Close()
}

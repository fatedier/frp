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
	"net"
	"reflect"
	"strings"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/vhost"
)

func init() {
	RegisterProxyFactory(reflect.TypeOf(&v1.HTTPSProxyConfig{}), NewHTTPSProxy)
}

type HTTPSProxy struct {
	*BaseProxy
	cfg *v1.HTTPSProxyConfig
}

func NewHTTPSProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.HTTPSProxyConfig)
	if !ok {
		return nil
	}
	return &HTTPSProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
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

		l, err := pxy.listenForDomain(routeConfig, domain)
		if err != nil {
			return "", err
		}
		pxy.listeners = append(pxy.listeners, l)
		addrs = append(addrs, util.CanonicalAddr(domain, pxy.serverCfg.VhostHTTPSPort))
		xl.Infof("https proxy listen for host [%s] group [%s]", domain, pxy.cfg.LoadBalancer.Group)
	}

	if pxy.cfg.SubDomain != "" {
		domain := pxy.cfg.SubDomain + "." + pxy.serverCfg.SubDomainHost
		l, err := pxy.listenForDomain(routeConfig, domain)
		if err != nil {
			return "", err
		}
		pxy.listeners = append(pxy.listeners, l)
		addrs = append(addrs, util.CanonicalAddr(domain, pxy.serverCfg.VhostHTTPSPort))
		xl.Infof("https proxy listen for host [%s] group [%s]", domain, pxy.cfg.LoadBalancer.Group)
	}

	pxy.startCommonTCPListenersHandler()
	remoteAddr = strings.Join(addrs, ",")
	return
}

func (pxy *HTTPSProxy) Close() {
	pxy.BaseProxy.Close()
}

func (pxy *HTTPSProxy) listenForDomain(routeConfig *vhost.RouteConfig, domain string) (net.Listener, error) {
	tmpRouteConfig := *routeConfig
	tmpRouteConfig.Domain = domain

	if pxy.cfg.LoadBalancer.Group != "" {
		return pxy.rc.HTTPSGroupCtl.Listen(
			pxy.ctx,
			pxy.cfg.LoadBalancer.Group,
			pxy.cfg.LoadBalancer.GroupKey,
			tmpRouteConfig,
		)
	}
	return pxy.rc.VhostHTTPSMuxer.Listen(pxy.ctx, &tmpRouteConfig)
}

// Copyright 2020 guylewin, guy@lewin.co.il
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
	"strings"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/vhost"
)

func init() {
	RegisterProxyFactory(reflect.TypeOf(&v1.TCPMuxProxyConfig{}), NewTCPMuxProxy)
}

type TCPMuxProxy struct {
	*BaseProxy
	cfg *v1.TCPMuxProxyConfig
}

func NewTCPMuxProxy(baseProxy *BaseProxy) Proxy {
	unwrapped, ok := baseProxy.GetConfigurer().(*v1.TCPMuxProxyConfig)
	if !ok {
		return nil
	}
	return &TCPMuxProxy{
		BaseProxy: baseProxy,
		cfg:       unwrapped,
	}
}

func (pxy *TCPMuxProxy) httpConnectListen(
	domain, routeByHTTPUser, httpUser, httpPwd string, addrs []string) ([]string, error,
) {
	var l net.Listener
	var err error
	routeConfig := &vhost.RouteConfig{
		Domain:          domain,
		RouteByHTTPUser: routeByHTTPUser,
		Username:        httpUser,
		Password:        httpPwd,
	}
	if pxy.cfg.LoadBalancer.Group != "" {
		l, err = pxy.rc.TCPMuxGroupCtl.Listen(pxy.ctx, pxy.cfg.Multiplexer,
			pxy.cfg.LoadBalancer.Group, pxy.cfg.LoadBalancer.GroupKey, *routeConfig)
	} else {
		l, err = pxy.rc.TCPMuxHTTPConnectMuxer.Listen(pxy.ctx, routeConfig)
	}
	if err != nil {
		return nil, err
	}
	pxy.xl.Infof("tcpmux httpconnect multiplexer listens for host [%s], group [%s] routeByHTTPUser [%s]",
		domain, pxy.cfg.LoadBalancer.Group, pxy.cfg.RouteByHTTPUser)
	pxy.listeners = append(pxy.listeners, l)
	return append(addrs, util.CanonicalAddr(domain, pxy.serverCfg.TCPMuxHTTPConnectPort)), nil
}

func (pxy *TCPMuxProxy) httpConnectRun() (remoteAddr string, err error) {
	addrs := make([]string, 0)
	for _, domain := range pxy.cfg.CustomDomains {
		if domain == "" {
			continue
		}

		addrs, err = pxy.httpConnectListen(domain, pxy.cfg.RouteByHTTPUser, pxy.cfg.HTTPUser, pxy.cfg.HTTPPassword, addrs)
		if err != nil {
			return "", err
		}
	}

	if pxy.cfg.SubDomain != "" {
		addrs, err = pxy.httpConnectListen(pxy.cfg.SubDomain+"."+pxy.serverCfg.SubDomainHost,
			pxy.cfg.RouteByHTTPUser, pxy.cfg.HTTPUser, pxy.cfg.HTTPPassword, addrs)
		if err != nil {
			return "", err
		}
	}

	pxy.startCommonTCPListenersHandler()
	remoteAddr = strings.Join(addrs, ",")
	return remoteAddr, err
}

func (pxy *TCPMuxProxy) Run() (remoteAddr string, err error) {
	switch v1.TCPMultiplexerType(pxy.cfg.Multiplexer) {
	case v1.TCPMultiplexerHTTPConnect:
		remoteAddr, err = pxy.httpConnectRun()
	default:
		err = fmt.Errorf("unknown multiplexer [%s]", pxy.cfg.Multiplexer)
	}

	if err != nil {
		pxy.Close()
	}
	return remoteAddr, err
}

func (pxy *TCPMuxProxy) Close() {
	pxy.BaseProxy.Close()
}

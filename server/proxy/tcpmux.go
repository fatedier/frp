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
	"strings"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/consts"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/vhost"
)

type TCPMuxProxy struct {
	*BaseProxy
	cfg *config.TCPMuxProxyConf
}

func (pxy *TCPMuxProxy) httpConnectListen(domain string, addrs []string) (_ []string, err error) {
	var l net.Listener
	if pxy.cfg.Group != "" {
		l, err = pxy.rc.TCPMuxGroupCtl.Listen(pxy.ctx, pxy.cfg.Multiplexer, pxy.cfg.Group, pxy.cfg.GroupKey, domain)
	} else {
		routeConfig := &vhost.RouteConfig{
			Domain: domain,
		}
		l, err = pxy.rc.TCPMuxHTTPConnectMuxer.Listen(pxy.ctx, routeConfig)
	}
	if err != nil {
		return nil, err
	}
	pxy.xl.Info("tcpmux httpconnect multiplexer listens for host [%s]", domain)
	pxy.listeners = append(pxy.listeners, l)
	return append(addrs, util.CanonicalAddr(domain, pxy.serverCfg.TCPMuxHTTPConnectPort)), nil
}

func (pxy *TCPMuxProxy) httpConnectRun() (remoteAddr string, err error) {
	addrs := make([]string, 0)
	for _, domain := range pxy.cfg.CustomDomains {
		if domain == "" {
			continue
		}

		addrs, err = pxy.httpConnectListen(domain, addrs)
		if err != nil {
			return "", err
		}
	}

	if pxy.cfg.SubDomain != "" {
		addrs, err = pxy.httpConnectListen(pxy.cfg.SubDomain+"."+pxy.serverCfg.SubDomainHost, addrs)
		if err != nil {
			return "", err
		}
	}

	pxy.startListenHandler(pxy, HandleUserTCPConnection)
	remoteAddr = strings.Join(addrs, ",")
	return remoteAddr, err
}

func (pxy *TCPMuxProxy) Run() (remoteAddr string, err error) {
	switch pxy.cfg.Multiplexer {
	case consts.HTTPConnectTCPMultiplexer:
		remoteAddr, err = pxy.httpConnectRun()
	default:
		err = fmt.Errorf("unknown multiplexer [%s]", pxy.cfg.Multiplexer)
	}

	if err != nil {
		pxy.Close()
	}
	return remoteAddr, err
}

func (pxy *TCPMuxProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *TCPMuxProxy) Close() {
	pxy.BaseProxy.Close()
}

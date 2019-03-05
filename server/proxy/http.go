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
	"io"
	"strings"

	"github.com/fatedier/frp/g"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/server/stats"
	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/util"
	"github.com/fatedier/frp/utils/vhost"

	frpIo "github.com/fatedier/golib/io"
)

type HttpProxy struct {
	*BaseProxy
	cfg *config.HttpProxyConf

	closeFuncs []func()
}

func (pxy *HttpProxy) Run() (remoteAddr string, err error) {
	routeConfig := vhost.VhostRouteConfig{
		RewriteHost:  pxy.cfg.HostHeaderRewrite,
		Headers:      pxy.cfg.Headers,
		Username:     pxy.cfg.HttpUser,
		Password:     pxy.cfg.HttpPwd,
		CreateConnFn: pxy.GetRealConn,
	}

	locations := pxy.cfg.Locations
	if len(locations) == 0 {
		locations = []string{""}
	}

	addrs := make([]string, 0)
	for _, domain := range pxy.cfg.CustomDomains {
		routeConfig.Domain = domain
		for _, location := range locations {
			routeConfig.Location = location
			err = pxy.rc.HttpReverseProxy.Register(routeConfig)
			if err != nil {
				return
			}
			tmpDomain := routeConfig.Domain
			tmpLocation := routeConfig.Location
			addrs = append(addrs, util.CanonicalAddr(tmpDomain, int(g.GlbServerCfg.VhostHttpPort)))
			pxy.closeFuncs = append(pxy.closeFuncs, func() {
				pxy.rc.HttpReverseProxy.UnRegister(tmpDomain, tmpLocation)
			})
			pxy.Info("http proxy listen for host [%s] location [%s]", routeConfig.Domain, routeConfig.Location)
		}
	}

	if pxy.cfg.SubDomain != "" {
		routeConfig.Domain = pxy.cfg.SubDomain + "." + g.GlbServerCfg.SubDomainHost
		for _, location := range locations {
			routeConfig.Location = location
			err = pxy.rc.HttpReverseProxy.Register(routeConfig)
			if err != nil {
				return
			}
			tmpDomain := routeConfig.Domain
			tmpLocation := routeConfig.Location
			addrs = append(addrs, util.CanonicalAddr(tmpDomain, g.GlbServerCfg.VhostHttpPort))
			pxy.closeFuncs = append(pxy.closeFuncs, func() {
				pxy.rc.HttpReverseProxy.UnRegister(tmpDomain, tmpLocation)
			})
			pxy.Info("http proxy listen for host [%s] location [%s]", routeConfig.Domain, routeConfig.Location)
		}
	}
	remoteAddr = strings.Join(addrs, ",")
	return
}

func (pxy *HttpProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *HttpProxy) GetRealConn() (workConn frpNet.Conn, err error) {
	tmpConn, errRet := pxy.GetWorkConnFromPool()
	if errRet != nil {
		err = errRet
		return
	}

	var rwc io.ReadWriteCloser = tmpConn
	if pxy.cfg.UseEncryption {
		rwc, err = frpIo.WithEncryption(rwc, []byte(g.GlbServerCfg.Token))
		if err != nil {
			pxy.Error("create encryption stream error: %v", err)
			return
		}
	}
	if pxy.cfg.UseCompression {
		rwc = frpIo.WithCompression(rwc)
	}
	workConn = frpNet.WrapReadWriteCloserToConn(rwc, tmpConn)
	workConn = frpNet.WrapStatsConn(workConn, pxy.updateStatsAfterClosedConn)
	pxy.statsCollector.Mark(stats.TypeOpenConnection, &stats.OpenConnectionPayload{ProxyName: pxy.GetName()})
	return
}

func (pxy *HttpProxy) updateStatsAfterClosedConn(totalRead, totalWrite int64) {
	name := pxy.GetName()
	pxy.statsCollector.Mark(stats.TypeCloseProxy, &stats.CloseConnectionPayload{ProxyName: name})
	pxy.statsCollector.Mark(stats.TypeAddTrafficIn, &stats.AddTrafficInPayload{
		ProxyName:    name,
		TrafficBytes: totalWrite,
	})
	pxy.statsCollector.Mark(stats.TypeAddTrafficOut, &stats.AddTrafficOutPayload{
		ProxyName:    name,
		TrafficBytes: totalRead,
	})
}

func (pxy *HttpProxy) Close() {
	pxy.BaseProxy.Close()
	for _, closeFn := range pxy.closeFuncs {
		closeFn()
	}
}

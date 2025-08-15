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
	"net"
	"reflect"
	"strings"

	libio "github.com/fatedier/golib/io"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/limit"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/vhost"
	"github.com/fatedier/frp/server/metrics"
)

func init() {
	RegisterProxyFactory(reflect.TypeOf(&v1.HTTPSProxyConfig{}), NewHTTPSProxy)
}

type HTTPSProxy struct {
	*BaseProxy
	cfg *v1.HTTPSProxyConfig

	closeFuncs []func()
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
	routeConfig := vhost.RouteConfig{
		CreateConnFn: pxy.GetRealConn,
	}

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

		tmpRouteConfig := routeConfig

		// handle group
		if pxy.cfg.LoadBalancer.Group != "" {
			err = pxy.rc.HTTPSGroupCtl.Register(pxy.name, pxy.cfg.LoadBalancer.Group, pxy.cfg.LoadBalancer.GroupKey, routeConfig)
			if err != nil {
				return
			}

			pxy.closeFuncs = append(pxy.closeFuncs, func() {
				pxy.rc.HTTPSGroupCtl.UnRegister(pxy.name, pxy.cfg.LoadBalancer.Group, tmpRouteConfig)
			})
		} else {
			// no group - use direct muxer
			l, errRet := pxy.rc.VhostHTTPSMuxer.Listen(pxy.ctx, &routeConfig)
			if errRet != nil {
				err = errRet
				return
			}
			xl.Infof("https proxy listen for host [%s]", routeConfig.Domain)
			pxy.listeners = append(pxy.listeners, l)
		}
		addrs = append(addrs, util.CanonicalAddr(routeConfig.Domain, pxy.serverCfg.VhostHTTPSPort))
		xl.Infof("https proxy listen for host [%s] group [%s]",
			routeConfig.Domain, pxy.cfg.LoadBalancer.Group)
	}

	if pxy.cfg.SubDomain != "" {
		routeConfig.Domain = pxy.cfg.SubDomain + "." + pxy.serverCfg.SubDomainHost

		tmpRouteConfig := routeConfig

		// handle group
		if pxy.cfg.LoadBalancer.Group != "" {
			err = pxy.rc.HTTPSGroupCtl.Register(pxy.name, pxy.cfg.LoadBalancer.Group, pxy.cfg.LoadBalancer.GroupKey, routeConfig)
			if err != nil {
				return
			}

			pxy.closeFuncs = append(pxy.closeFuncs, func() {
				pxy.rc.HTTPSGroupCtl.UnRegister(pxy.name, pxy.cfg.LoadBalancer.Group, tmpRouteConfig)
			})
		} else {
			// no group - use direct muxer
			l, errRet := pxy.rc.VhostHTTPSMuxer.Listen(pxy.ctx, &routeConfig)
			if errRet != nil {
				err = errRet
				return
			}
			xl.Infof("https proxy listen for host [%s]", routeConfig.Domain)
			pxy.listeners = append(pxy.listeners, l)
		}
		addrs = append(addrs, util.CanonicalAddr(routeConfig.Domain, pxy.serverCfg.VhostHTTPSPort))

		xl.Infof("https proxy listen for host [%s] group [%s]",
			routeConfig.Domain, pxy.cfg.LoadBalancer.Group)
	}

	pxy.startCommonTCPListenersHandler()
	remoteAddr = strings.Join(addrs, ",")
	return
}

func (pxy *HTTPSProxy) GetRealConn(remoteAddr string) (workConn net.Conn, err error) {
	xl := pxy.xl
	rAddr, errRet := net.ResolveTCPAddr("tcp", remoteAddr)
	if errRet != nil {
		xl.Warnf("resolve TCP addr [%s] error: %v", remoteAddr, errRet)
		// we do not return error here since remoteAddr is not necessary for proxies without proxy protocol enabled
	}

	tmpConn, errRet := pxy.GetWorkConnFromPool(rAddr, nil)
	if errRet != nil {
		err = errRet
		return
	}

	var rwc io.ReadWriteCloser = tmpConn
	if pxy.cfg.Transport.UseEncryption {
		rwc, err = libio.WithEncryption(rwc, []byte(pxy.serverCfg.Auth.Token))
		if err != nil {
			xl.Errorf("create encryption stream error: %v", err)
			return
		}
	}
	if pxy.cfg.Transport.UseCompression {
		rwc = libio.WithCompression(rwc)
	}

	if pxy.GetLimiter() != nil {
		rwc = libio.WrapReadWriteCloser(limit.NewReader(rwc, pxy.GetLimiter()), limit.NewWriter(rwc, pxy.GetLimiter()), func() error {
			return rwc.Close()
		})
	}

	workConn = netpkg.WrapReadWriteCloserToConn(rwc, tmpConn)
	workConn = netpkg.WrapStatsConn(workConn, pxy.updateStatsAfterClosedConn)
	metrics.Server.OpenConnection(pxy.GetName(), pxy.GetConfigurer().GetBaseConfig().Type)
	return
}

func (pxy *HTTPSProxy) updateStatsAfterClosedConn(totalRead, totalWrite int64) {
	name := pxy.GetName()
	proxyType := pxy.GetConfigurer().GetBaseConfig().Type
	metrics.Server.CloseConnection(name, proxyType)
	metrics.Server.AddTrafficIn(name, proxyType, totalWrite)
	metrics.Server.AddTrafficOut(name, proxyType, totalRead)
}

func (pxy *HTTPSProxy) Close() {
	pxy.BaseProxy.Close()
	for _, closeFn := range pxy.closeFuncs {
		closeFn()
	}
}

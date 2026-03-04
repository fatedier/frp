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

package http

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/fatedier/frp/pkg/config/types"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/metrics/mem"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/frp/server/http/model"
	"github.com/fatedier/frp/server/proxy"
	"github.com/fatedier/frp/server/registry"
)

type Controller struct {
	// dependencies
	serverCfg      *v1.ServerConfig
	clientRegistry *registry.ClientRegistry
	pxyManager     ProxyManager
}

type ProxyManager interface {
	GetByName(name string) (proxy.Proxy, bool)
}

func NewController(
	serverCfg *v1.ServerConfig,
	clientRegistry *registry.ClientRegistry,
	pxyManager ProxyManager,
) *Controller {
	return &Controller{
		serverCfg:      serverCfg,
		clientRegistry: clientRegistry,
		pxyManager:     pxyManager,
	}
}

// /api/serverinfo
func (c *Controller) APIServerInfo(ctx *httppkg.Context) (any, error) {
	serverStats := mem.StatsCollector.GetServer()
	svrResp := model.ServerInfoResp{
		Version:               version.Full(),
		BindPort:              c.serverCfg.BindPort,
		VhostHTTPPort:         c.serverCfg.VhostHTTPPort,
		VhostHTTPSPort:        c.serverCfg.VhostHTTPSPort,
		TCPMuxHTTPConnectPort: c.serverCfg.TCPMuxHTTPConnectPort,
		KCPBindPort:           c.serverCfg.KCPBindPort,
		QUICBindPort:          c.serverCfg.QUICBindPort,
		SubdomainHost:         c.serverCfg.SubDomainHost,
		MaxPoolCount:          c.serverCfg.Transport.MaxPoolCount,
		MaxPortsPerClient:     c.serverCfg.MaxPortsPerClient,
		HeartBeatTimeout:      c.serverCfg.Transport.HeartbeatTimeout,
		AllowPortsStr:         types.PortsRangeSlice(c.serverCfg.AllowPorts).String(),
		TLSForce:              c.serverCfg.Transport.TLS.Force,

		TotalTrafficIn:  serverStats.TotalTrafficIn,
		TotalTrafficOut: serverStats.TotalTrafficOut,
		CurConns:        serverStats.CurConns,
		ClientCounts:    serverStats.ClientCounts,
		ProxyTypeCounts: serverStats.ProxyTypeCounts,
	}

	return svrResp, nil
}

// /api/clients
func (c *Controller) APIClientList(ctx *httppkg.Context) (any, error) {
	if c.clientRegistry == nil {
		return nil, fmt.Errorf("client registry unavailable")
	}

	userFilter := ctx.Query("user")
	clientIDFilter := ctx.Query("clientId")
	runIDFilter := ctx.Query("runId")
	statusFilter := strings.ToLower(ctx.Query("status"))

	records := c.clientRegistry.List()
	items := make([]model.ClientInfoResp, 0, len(records))
	for _, info := range records {
		if userFilter != "" && info.User != userFilter {
			continue
		}
		if clientIDFilter != "" && info.ClientID() != clientIDFilter {
			continue
		}
		if runIDFilter != "" && info.RunID != runIDFilter {
			continue
		}
		if !matchStatusFilter(info.Online, statusFilter) {
			continue
		}
		items = append(items, buildClientInfoResp(info))
	}

	slices.SortFunc(items, func(a, b model.ClientInfoResp) int {
		if v := cmp.Compare(a.User, b.User); v != 0 {
			return v
		}
		if v := cmp.Compare(a.ClientID, b.ClientID); v != 0 {
			return v
		}
		return cmp.Compare(a.Key, b.Key)
	})

	return items, nil
}

// /api/clients/{key}
func (c *Controller) APIClientDetail(ctx *httppkg.Context) (any, error) {
	key := ctx.Param("key")
	if key == "" {
		return nil, fmt.Errorf("missing client key")
	}

	if c.clientRegistry == nil {
		return nil, fmt.Errorf("client registry unavailable")
	}

	info, ok := c.clientRegistry.GetByKey(key)
	if !ok {
		return nil, httppkg.NewError(http.StatusNotFound, fmt.Sprintf("client %s not found", key))
	}

	return buildClientInfoResp(info), nil
}

// /api/proxy/:type
func (c *Controller) APIProxyByType(ctx *httppkg.Context) (any, error) {
	proxyType := ctx.Param("type")

	proxyInfoResp := model.GetProxyInfoResp{}
	proxyInfoResp.Proxies = c.getProxyStatsByType(proxyType)
	slices.SortFunc(proxyInfoResp.Proxies, func(a, b *model.ProxyStatsInfo) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return proxyInfoResp, nil
}

// /api/proxy/:type/:name
func (c *Controller) APIProxyByTypeAndName(ctx *httppkg.Context) (any, error) {
	proxyType := ctx.Param("type")
	name := ctx.Param("name")

	proxyStatsResp, code, msg := c.getProxyStatsByTypeAndName(proxyType, name)
	if code != 200 {
		return nil, httppkg.NewError(code, msg)
	}

	return proxyStatsResp, nil
}

// /api/traffic/:name
func (c *Controller) APIProxyTraffic(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")

	trafficResp := model.GetProxyTrafficResp{}
	trafficResp.Name = name
	proxyTrafficInfo := mem.StatsCollector.GetProxyTraffic(name)

	if proxyTrafficInfo == nil {
		return nil, httppkg.NewError(http.StatusNotFound, "no proxy info found")
	}
	trafficResp.TrafficIn = proxyTrafficInfo.TrafficIn
	trafficResp.TrafficOut = proxyTrafficInfo.TrafficOut

	return trafficResp, nil
}

// /api/proxies/:name
func (c *Controller) APIProxyByName(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")

	ps := mem.StatsCollector.GetProxyByName(name)
	if ps == nil {
		return nil, httppkg.NewError(http.StatusNotFound, "no proxy info found")
	}

	proxyInfo := model.GetProxyStatsResp{
		Name:            ps.Name,
		User:            ps.User,
		ClientID:        ps.ClientID,
		TodayTrafficIn:  ps.TodayTrafficIn,
		TodayTrafficOut: ps.TodayTrafficOut,
		CurConns:        ps.CurConns,
		LastStartTime:   ps.LastStartTime,
		LastCloseTime:   ps.LastCloseTime,
	}

	if pxy, ok := c.pxyManager.GetByName(name); ok {
		proxyInfo.Conf = getConfFromConfigurer(pxy.GetConfigurer())
		proxyInfo.Status = "online"
	} else {
		proxyInfo.Status = "offline"
	}

	return proxyInfo, nil
}

// DELETE /api/proxies?status=offline
func (c *Controller) DeleteProxies(ctx *httppkg.Context) (any, error) {
	status := ctx.Query("status")
	if status != "offline" {
		return nil, httppkg.NewError(http.StatusBadRequest, "status only support offline")
	}
	cleared, total := mem.StatsCollector.ClearOfflineProxies()
	log.Infof("cleared [%d] offline proxies, total [%d] proxies", cleared, total)
	return httppkg.GeneralResponse{Code: 200, Msg: "success"}, nil
}

func (c *Controller) getProxyStatsByType(proxyType string) (proxyInfos []*model.ProxyStatsInfo) {
	proxyStats := mem.StatsCollector.GetProxiesByType(proxyType)
	proxyInfos = make([]*model.ProxyStatsInfo, 0, len(proxyStats))
	for _, ps := range proxyStats {
		proxyInfo := &model.ProxyStatsInfo{
			User:     ps.User,
			ClientID: ps.ClientID,
		}
		if pxy, ok := c.pxyManager.GetByName(ps.Name); ok {
			proxyInfo.Conf = getConfFromConfigurer(pxy.GetConfigurer())
			proxyInfo.Status = "online"
		} else {
			proxyInfo.Status = "offline"
		}
		proxyInfo.Name = ps.Name
		proxyInfo.TodayTrafficIn = ps.TodayTrafficIn
		proxyInfo.TodayTrafficOut = ps.TodayTrafficOut
		proxyInfo.CurConns = ps.CurConns
		proxyInfo.LastStartTime = ps.LastStartTime
		proxyInfo.LastCloseTime = ps.LastCloseTime
		proxyInfos = append(proxyInfos, proxyInfo)
	}
	return
}

func (c *Controller) getProxyStatsByTypeAndName(proxyType string, proxyName string) (proxyInfo model.GetProxyStatsResp, code int, msg string) {
	proxyInfo.Name = proxyName
	ps := mem.StatsCollector.GetProxiesByTypeAndName(proxyType, proxyName)
	if ps == nil {
		code = 404
		msg = "no proxy info found"
	} else {
		proxyInfo.User = ps.User
		proxyInfo.ClientID = ps.ClientID
		if pxy, ok := c.pxyManager.GetByName(proxyName); ok {
			proxyInfo.Conf = getConfFromConfigurer(pxy.GetConfigurer())
			proxyInfo.Status = "online"
		} else {
			proxyInfo.Status = "offline"
		}
		proxyInfo.TodayTrafficIn = ps.TodayTrafficIn
		proxyInfo.TodayTrafficOut = ps.TodayTrafficOut
		proxyInfo.CurConns = ps.CurConns
		proxyInfo.LastStartTime = ps.LastStartTime
		proxyInfo.LastCloseTime = ps.LastCloseTime
		code = 200
	}

	return
}

func buildClientInfoResp(info registry.ClientInfo) model.ClientInfoResp {
	resp := model.ClientInfoResp{
		Key:              info.Key,
		User:             info.User,
		ClientID:         info.ClientID(),
		RunID:            info.RunID,
		Version:          info.Version,
		Hostname:         info.Hostname,
		ClientIP:         info.IP,
		FirstConnectedAt: toUnix(info.FirstConnectedAt),
		LastConnectedAt:  toUnix(info.LastConnectedAt),
		Online:           info.Online,
	}
	if !info.DisconnectedAt.IsZero() {
		resp.DisconnectedAt = info.DisconnectedAt.Unix()
	}
	return resp
}

func toUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func matchStatusFilter(online bool, filter string) bool {
	switch strings.ToLower(filter) {
	case "", "all":
		return true
	case "online":
		return online
	case "offline":
		return !online
	default:
		return true
	}
}

func getConfFromConfigurer(cfg v1.ProxyConfigurer) any {
	outBase := model.BaseOutConf{ProxyBaseConfig: *cfg.GetBaseConfig()}

	switch c := cfg.(type) {
	case *v1.TCPProxyConfig:
		return &model.TCPOutConf{BaseOutConf: outBase, RemotePort: c.RemotePort}
	case *v1.UDPProxyConfig:
		return &model.UDPOutConf{BaseOutConf: outBase, RemotePort: c.RemotePort}
	case *v1.HTTPProxyConfig:
		return &model.HTTPOutConf{
			BaseOutConf:       outBase,
			DomainConfig:      c.DomainConfig,
			Locations:         c.Locations,
			HostHeaderRewrite: c.HostHeaderRewrite,
		}
	case *v1.HTTPSProxyConfig:
		return &model.HTTPSOutConf{
			BaseOutConf:  outBase,
			DomainConfig: c.DomainConfig,
		}
	case *v1.TCPMuxProxyConfig:
		return &model.TCPMuxOutConf{
			BaseOutConf:     outBase,
			DomainConfig:    c.DomainConfig,
			Multiplexer:     c.Multiplexer,
			RouteByHTTPUser: c.RouteByHTTPUser,
		}
	case *v1.STCPProxyConfig:
		return &model.STCPOutConf{BaseOutConf: outBase}
	case *v1.XTCPProxyConfig:
		return &model.XTCPOutConf{BaseOutConf: outBase}
	}
	return outBase
}

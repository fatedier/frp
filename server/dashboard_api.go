// Copyright 2017 fatedier, fatedier@gmail.com
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

package server

import (
	"encoding/json"
	"net/http"

	"github.com/fatedier/frp/g"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/consts"
	"github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/version"

	"github.com/gorilla/mux"
)

type GeneralResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// api/serverinfo
type ServerInfoResp struct {
	GeneralResponse

	Version           string `json:"version"`
	BindPort          int    `json:"bind_port"`
	BindUdpPort       int    `json:"bind_udp_port"`
	VhostHttpPort     int    `json:"vhost_http_port"`
	VhostHttpsPort    int    `json:"vhost_https_port"`
	KcpBindPort       int    `json:"kcp_bind_port"`
	AuthTimeout       int64  `json:"auth_timeout"`
	SubdomainHost     string `json:"subdomain_host"`
	MaxPoolCount      int64  `json:"max_pool_count"`
	MaxPortsPerClient int64  `json:"max_ports_per_client"`
	HeartBeatTimeout  int64  `json:"heart_beat_timeout"`

	TotalTrafficIn  int64            `json:"total_traffic_in"`
	TotalTrafficOut int64            `json:"total_traffic_out"`
	CurConns        int64            `json:"cur_conns"`
	ClientCounts    int64            `json:"client_counts"`
	ProxyTypeCounts map[string]int64 `json:"proxy_type_count"`
}

func apiServerInfo(w http.ResponseWriter, r *http.Request) {
	var (
		buf []byte
		res ServerInfoResp
	)
	defer func() {
		log.Info("Http response [%s]: code [%d]", r.URL.Path, res.Code)
	}()

	log.Info("Http request: [%s]", r.URL.Path)
	cfg := &g.GlbServerCfg.ServerCommonConf
	serverStats := StatsGetServer()
	res = ServerInfoResp{
		Version:           version.Full(),
		BindPort:          cfg.BindPort,
		BindUdpPort:       cfg.BindUdpPort,
		VhostHttpPort:     cfg.VhostHttpPort,
		VhostHttpsPort:    cfg.VhostHttpsPort,
		KcpBindPort:       cfg.KcpBindPort,
		AuthTimeout:       cfg.AuthTimeout,
		SubdomainHost:     cfg.SubDomainHost,
		MaxPoolCount:      cfg.MaxPoolCount,
		MaxPortsPerClient: cfg.MaxPortsPerClient,
		HeartBeatTimeout:  cfg.HeartBeatTimeout,

		TotalTrafficIn:  serverStats.TotalTrafficIn,
		TotalTrafficOut: serverStats.TotalTrafficOut,
		CurConns:        serverStats.CurConns,
		ClientCounts:    serverStats.ClientCounts,
		ProxyTypeCounts: serverStats.ProxyTypeCounts,
	}

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

type BaseOutConf struct {
	config.BaseProxyConf
}

type TcpOutConf struct {
	BaseOutConf
	RemotePort int `json:"remote_port"`
}

type UdpOutConf struct {
	BaseOutConf
	RemotePort int `json:"remote_port"`
}

type HttpOutConf struct {
	BaseOutConf
	config.DomainConf
	Locations         []string `json:"locations"`
	HostHeaderRewrite string   `json:"host_header_rewrite"`
}

type HttpsOutConf struct {
	BaseOutConf
	config.DomainConf
}

type StcpOutConf struct {
	BaseOutConf
}

type XtcpOutConf struct {
	BaseOutConf
}

func getConfByType(proxyType string) interface{} {
	switch proxyType {
	case consts.TcpProxy:
		return &TcpOutConf{}
	case consts.UdpProxy:
		return &UdpOutConf{}
	case consts.HttpProxy:
		return &HttpOutConf{}
	case consts.HttpsProxy:
		return &HttpsOutConf{}
	case consts.StcpProxy:
		return &StcpOutConf{}
	case consts.XtcpProxy:
		return &XtcpOutConf{}
	default:
		return nil
	}
}

// Get proxy info.
type ProxyStatsInfo struct {
	Name            string      `json:"name"`
	Conf            interface{} `json:"conf"`
	TodayTrafficIn  int64       `json:"today_traffic_in"`
	TodayTrafficOut int64       `json:"today_traffic_out"`
	CurConns        int64       `json:"cur_conns"`
	LastStartTime   string      `json:"last_start_time"`
	LastCloseTime   string      `json:"last_close_time"`
	Status          string      `json:"status"`
}

type GetProxyInfoResp struct {
	GeneralResponse
	Proxies []*ProxyStatsInfo `json:"proxies"`
}

// api/proxy/:type
func apiProxyByType(w http.ResponseWriter, r *http.Request) {
	var (
		buf []byte
		res GetProxyInfoResp
	)
	params := mux.Vars(r)
	proxyType := params["type"]

	defer func() {
		log.Info("Http response [%s]: code [%d]", r.URL.Path, res.Code)
		log.Info(r.URL.Path)
		log.Info(r.URL.RawPath)
	}()
	log.Info("Http request: [%s]", r.URL.Path)

	res.Proxies = getProxyStatsByType(proxyType)

	buf, _ = json.Marshal(&res)
	w.Write(buf)

}

func getProxyStatsByType(proxyType string) (proxyInfos []*ProxyStatsInfo) {
	proxyStats := StatsGetProxiesByType(proxyType)
	proxyInfos = make([]*ProxyStatsInfo, 0, len(proxyStats))
	for _, ps := range proxyStats {
		proxyInfo := &ProxyStatsInfo{}
		if pxy, ok := ServerService.pxyManager.GetByName(ps.Name); ok {
			content, err := json.Marshal(pxy.GetConf())
			if err != nil {
				log.Warn("marshal proxy [%s] conf info error: %v", ps.Name, err)
				continue
			}
			proxyInfo.Conf = getConfByType(ps.Type)
			if err = json.Unmarshal(content, &proxyInfo.Conf); err != nil {
				log.Warn("unmarshal proxy [%s] conf info error: %v", ps.Name, err)
				continue
			}
			proxyInfo.Status = consts.Online
		} else {
			proxyInfo.Status = consts.Offline
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

// Get proxy info by name.
type GetProxyStatsResp struct {
	GeneralResponse

	Name            string      `json:"name"`
	Conf            interface{} `json:"conf"`
	TodayTrafficIn  int64       `json:"today_traffic_in"`
	TodayTrafficOut int64       `json:"today_traffic_out"`
	CurConns        int64       `json:"cur_conns"`
	LastStartTime   string      `json:"last_start_time"`
	LastCloseTime   string      `json:"last_close_time"`
	Status          string      `json:"status"`
}

// api/proxy/:type/:name
func apiProxyByTypeAndName(w http.ResponseWriter, r *http.Request) {
	var (
		buf []byte
		res GetProxyStatsResp
	)
	params := mux.Vars(r)
	proxyType := params["type"]
	name := params["name"]

	defer func() {
		log.Info("Http response [%s]: code [%d]", r.URL.Path, res.Code)
	}()
	log.Info("Http request: [%s]", r.URL.Path)

	res = getProxyStatsByTypeAndName(proxyType, name)

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

func getProxyStatsByTypeAndName(proxyType string, proxyName string) (proxyInfo GetProxyStatsResp) {
	proxyInfo.Name = proxyName
	ps := StatsGetProxiesByTypeAndName(proxyType, proxyName)
	if ps == nil {
		proxyInfo.Code = 1
		proxyInfo.Msg = "no proxy info found"
	} else {
		if pxy, ok := ServerService.pxyManager.GetByName(proxyName); ok {
			content, err := json.Marshal(pxy.GetConf())
			if err != nil {
				log.Warn("marshal proxy [%s] conf info error: %v", ps.Name, err)
				proxyInfo.Code = 2
				proxyInfo.Msg = "parse conf error"
				return
			}
			proxyInfo.Conf = getConfByType(ps.Type)
			if err = json.Unmarshal(content, &proxyInfo.Conf); err != nil {
				log.Warn("unmarshal proxy [%s] conf info error: %v", ps.Name, err)
				proxyInfo.Code = 2
				proxyInfo.Msg = "parse conf error"
				return
			}
			proxyInfo.Status = consts.Online
		} else {
			proxyInfo.Status = consts.Offline
		}
		proxyInfo.TodayTrafficIn = ps.TodayTrafficIn
		proxyInfo.TodayTrafficOut = ps.TodayTrafficOut
		proxyInfo.CurConns = ps.CurConns
		proxyInfo.LastStartTime = ps.LastStartTime
		proxyInfo.LastCloseTime = ps.LastCloseTime
	}

	return
}

// api/traffic/:name
type GetProxyTrafficResp struct {
	GeneralResponse

	Name       string  `json:"name"`
	TrafficIn  []int64 `json:"traffic_in"`
	TrafficOut []int64 `json:"traffic_out"`
}

func apiProxyTraffic(w http.ResponseWriter, r *http.Request) {
	var (
		buf []byte
		res GetProxyTrafficResp
	)
	params := mux.Vars(r)
	name := params["name"]

	defer func() {
		log.Info("Http response [%s]: code [%d]", r.URL.Path, res.Code)
	}()
	log.Info("Http request: [%s]", r.URL.Path)

	res.Name = name
	proxyTrafficInfo := StatsGetProxyTraffic(name)
	if proxyTrafficInfo == nil {
		res.Code = 1
		res.Msg = "no proxy info found"
	} else {
		res.TrafficIn = proxyTrafficInfo.TrafficIn
		res.TrafficOut = proxyTrafficInfo.TrafficOut
	}

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

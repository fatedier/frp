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

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/consts"
	"github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/version"

	"github.com/julienschmidt/httprouter"
)

type GeneralResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// api/serverinfo
type ServerInfoResp struct {
	GeneralResponse

	Version          string `json:"version"`
	VhostHttpPort    int    `json:"vhost_http_port"`
	VhostHttpsPort   int    `json:"vhost_https_port"`
	AuthTimeout      int64  `json:"auth_timeout"`
	SubdomainHost    string `json:"subdomain_host"`
	MaxPoolCount     int64  `json:"max_pool_count"`
	HeartBeatTimeout int64  `json:"heart_beat_timeout"`

	TotalTrafficIn  int64            `json:"total_traffic_in"`
	TotalTrafficOut int64            `json:"total_traffic_out"`
	CurConns        int64            `json:"cur_conns"`
	ClientCounts    int64            `json:"client_counts"`
	ProxyTypeCounts map[string]int64 `json:"proxy_type_count"`
}

func apiServerInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var (
		buf []byte
		res ServerInfoResp
	)
	defer func() {
		log.Info("Http response [/api/serverinfo]: code [%d]", res.Code)
	}()

	log.Info("Http request: [/api/serverinfo]")
	cfg := config.ServerCommonCfg
	serverStats := StatsGetServer()
	res = ServerInfoResp{
		Version:          version.Full(),
		VhostHttpPort:    cfg.VhostHttpPort,
		VhostHttpsPort:   cfg.VhostHttpsPort,
		AuthTimeout:      cfg.AuthTimeout,
		SubdomainHost:    cfg.SubDomainHost,
		MaxPoolCount:     cfg.MaxPoolCount,
		HeartBeatTimeout: cfg.HeartBeatTimeout,

		TotalTrafficIn:  serverStats.TotalTrafficIn,
		TotalTrafficOut: serverStats.TotalTrafficOut,
		CurConns:        serverStats.CurConns,
		ClientCounts:    serverStats.ClientCounts,
		ProxyTypeCounts: serverStats.ProxyTypeCounts,
	}

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

// Get proxy info.
type ProxyStatsInfo struct {
	Name            string           `json:"name"`
	Conf            config.ProxyConf `json:"conf"`
	TodayTrafficIn  int64            `json:"today_traffic_in"`
	TodayTrafficOut int64            `json:"today_traffic_out"`
	CurConns        int64            `json:"cur_conns"`
	LastStartTime   string           `json:"last_start_time"`
	LastCloseTime   string           `json:"last_close_time"`
	Status          string           `json:"status"`
}

type GetProxyInfoResp struct {
	GeneralResponse
	Proxies []*ProxyStatsInfo `json:"proxies"`
}

// api/proxy/tcp
func apiProxyTcp(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var (
		buf []byte
		res GetProxyInfoResp
	)
	defer func() {
		log.Info("Http response [/api/proxy/tcp]: code [%d]", res.Code)
	}()
	log.Info("Http request: [/api/proxy/tcp]")

	res.Proxies = getProxyStatsByType(consts.TcpProxy)

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

// api/proxy/udp
func apiProxyUdp(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var (
		buf []byte
		res GetProxyInfoResp
	)
	defer func() {
		log.Info("Http response [/api/proxy/udp]: code [%d]", res.Code)
	}()
	log.Info("Http request: [/api/proxy/udp]")

	res.Proxies = getProxyStatsByType(consts.UdpProxy)

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

// api/proxy/http
func apiProxyHttp(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var (
		buf []byte
		res GetProxyInfoResp
	)
	defer func() {
		log.Info("Http response [/api/proxy/http]: code [%d]", res.Code)
	}()
	log.Info("Http request: [/api/proxy/http]")

	res.Proxies = getProxyStatsByType(consts.HttpProxy)

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

// api/proxy/https
func apiProxyHttps(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var (
		buf []byte
		res GetProxyInfoResp
	)
	defer func() {
		log.Info("Http response [/api/proxy/https]: code [%d]", res.Code)
	}()
	log.Info("Http request: [/api/proxy/https]")

	res.Proxies = getProxyStatsByType(consts.HttpsProxy)

	buf, _ = json.Marshal(&res)
	w.Write(buf)
}

func getProxyStatsByType(proxyType string) (proxyInfos []*ProxyStatsInfo) {
	proxyStats := StatsGetProxiesByType(proxyType)
	proxyInfos = make([]*ProxyStatsInfo, 0, len(proxyStats))
	for _, ps := range proxyStats {
		proxyInfo := &ProxyStatsInfo{}
		if pxy, ok := ServerService.pxyManager.GetByName(ps.Name); ok {
			proxyInfo.Conf = pxy.GetConf()
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

// api/proxy/traffic/:name
type GetProxyTrafficResp struct {
	GeneralResponse

	Name       string  `json:"name"`
	TrafficIn  []int64 `json:"traffic_in"`
	TrafficOut []int64 `json:"traffic_out"`
}

func apiProxyTraffic(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var (
		buf []byte
		res GetProxyTrafficResp
	)
	name := params.ByName("name")

	defer func() {
		log.Info("Http response [/api/proxy/traffic/:name]: code [%d]", res.Code)
	}()
	log.Info("Http request: [/api/proxy/traffic/:name]")

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

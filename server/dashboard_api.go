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
	"cmp"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/fatedier/frp/g"
	"github.com/fatedier/frp/pkg/config/types"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/metrics/mem"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
)

type GeneralResponse struct {
	Code int
	Msg  string
}

func (svr *Service) registerRouteHandlers(helper *httppkg.RouterRegisterHelper) {
	helper.Router.HandleFunc("/healthz", svr.healthz)
	subRouter := helper.Router.NewRoute().Subrouter()

	subRouter.Use(helper.AuthMiddleware.Middleware)

	// metrics
	if svr.cfg.EnablePrometheus {
		subRouter.Handle("/metrics", promhttp.Handler())
	}

	// apis
	subRouter.HandleFunc("/api/serverinfo", svr.apiServerInfo).Methods("GET")
	subRouter.HandleFunc("/api/proxy/{type}", svr.apiProxyByType).Methods("GET")
	subRouter.HandleFunc("/api/proxy/{type}/{name}", svr.apiProxyByTypeAndName).Methods("GET")
	subRouter.HandleFunc("/api/traffic/{name}", svr.apiProxyTraffic).Methods("GET")
	subRouter.HandleFunc("/api/proxies", svr.deleteProxies).Methods("DELETE")
	subRouter.HandleFunc("/api/client/close/{user}", svr.ApiCloseClient).Methods("GET")
	subRouter.HandleFunc("/api/server/close/frps", svr.ApiCloseFrps).Methods("GET")
	subRouter.HandleFunc("/api/server/check", svr.CheckServer).Methods("GET")
	subRouter.HandleFunc("/api/server/checkonline", svr.ApiCheckOnline).Methods("GET")
	subRouter.HandleFunc("/api/server/command", svr.RunCommand).Methods("GET")

	// view
	subRouter.Handle("/favicon.ico", http.FileServer(helper.AssetsFS)).Methods("GET")
	subRouter.PathPrefix("/static/").Handler(
		netpkg.MakeHTTPGzipHandler(http.StripPrefix("/static/", http.FileServer(helper.AssetsFS))),
	).Methods("GET")

	subRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusMovedPermanently)
	})
}

type serverInfoResp struct {
	Powered               string `json:"powered_by"`
	Version               string `json:"version"`
	BindPort              int    `json:"bindPort"`
	VhostHTTPPort         int    `json:"vhostHTTPPort"`
	VhostHTTPSPort        int    `json:"vhostHTTPSPort"`
	TCPMuxHTTPConnectPort int    `json:"tcpmuxHTTPConnectPort"`
	KCPBindPort           int    `json:"kcpBindPort"`
	QUICBindPort          int    `json:"quicBindPort"`
	SubdomainHost         string `json:"subdomainHost"`
	MaxPoolCount          int64  `json:"maxPoolCount"`
	MaxPortsPerClient     int64  `json:"maxPortsPerClient"`
	HeartBeatTimeout      int64  `json:"heartbeatTimeout"`
	AllowPortsStr         string `json:"allowPortsStr,omitempty"`
	TLSForce              bool   `json:"tlsForce,omitempty"`

	CpuUsage        string           `json:"cpu_usage"`
	RamUsage        string           `json:"ram_usage"`
	DiskUsage       string           `json:"disk_usage"`
	TotalTrafficIn  int64            `json:"totalTrafficIn"`
	TotalTrafficOut int64            `json:"totalTrafficOut"`
	CurConns        int64            `json:"curConns"`
	ClientCounts    int64            `json:"clientCounts"`
	ProxyTypeCounts map[string]int64 `json:"proxyTypeCount"`
}

// /healthz
func (svr *Service) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
}

// /api/serverinfo
func (svr *Service) apiServerInfo(w http.ResponseWriter, r *http.Request) {
	res := GeneralResponse{Code: 200}
	defer func() {
		log.Infof("HTTP响应 [%s]: 代码 [%d]", r.URL.Path, res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			_, _ = w.Write([]byte(res.Msg))
		}
	}()

	log.Infof("HTTP请求: [%s]", r.URL.Path)
	serverStats := mem.StatsCollector.GetServer()

	// CPU Usage
	CPU, err := cpu.Percent(0, false)
	if err != nil {
		return
	}
	cpuUsage := CPU[0]

	// RAM Usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	// 获取已分配的内存大小（以字节为单位）
	allocatedRAM := memStats.Alloc
	// 获取总内存大小（以字节为单位）
	totalRAM := memStats.TotalAlloc
	// 计算内存使用百分比
	ramUsage := float64(allocatedRAM) / float64(totalRAM) * 100

	// Disk Usage

	// 获取系统的所有磁盘分区
	partitions, err := disk.Partitions(false)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 计算系统的总磁盘占用率
	var totalUsage float64
	for _, partition := range partitions {
		// 获取磁盘分区的使用情况
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// 累加磁盘分区的占用率
		totalUsage += usage.UsedPercent
	}

	// 打印系统的总磁盘占用率
	diskUsage := totalUsage / float64(len(partitions))

	svrResp := serverInfoResp{
		Powered:               "HayFrp & HX Network",
		CpuUsage:              fmt.Sprintf("%.2f%%", cpuUsage),
		RamUsage:              fmt.Sprintf("%.2f%%", ramUsage),
		DiskUsage:             fmt.Sprintf("%.2f%%", diskUsage),
		Version:               version.Full(),
		BindPort:              svr.cfg.BindPort,
		VhostHTTPPort:         svr.cfg.VhostHTTPPort,
		VhostHTTPSPort:        svr.cfg.VhostHTTPSPort,
		TCPMuxHTTPConnectPort: svr.cfg.TCPMuxHTTPConnectPort,
		KCPBindPort:           svr.cfg.KCPBindPort,
		QUICBindPort:          svr.cfg.QUICBindPort,
		SubdomainHost:         svr.cfg.SubDomainHost,
		MaxPoolCount:          svr.cfg.Transport.MaxPoolCount,
		MaxPortsPerClient:     svr.cfg.MaxPortsPerClient,
		HeartBeatTimeout:      svr.cfg.Transport.HeartbeatTimeout,
		AllowPortsStr:         types.PortsRangeSlice(svr.cfg.AllowPorts).String(),
		TLSForce:              svr.cfg.Transport.TLS.Force,

		TotalTrafficIn:  serverStats.TotalTrafficIn,
		TotalTrafficOut: serverStats.TotalTrafficOut,
		CurConns:        serverStats.CurConns,
		ClientCounts:    serverStats.ClientCounts,
		ProxyTypeCounts: serverStats.ProxyTypeCounts,
	}

	buf, _ := json.Marshal(&svrResp)
	res.Msg = string(buf)
}

type BaseOutConf struct {
	v1.ProxyBaseConfig
}

type TCPOutConf struct {
	BaseOutConf
	RemotePort int `json:"remotePort"`
}

type TCPMuxOutConf struct {
	BaseOutConf
	v1.DomainConfig
	Multiplexer     string `json:"multiplexer"`
	RouteByHTTPUser string `json:"routeByHTTPUser"`
}

type UDPOutConf struct {
	BaseOutConf
	RemotePort int `json:"remotePort"`
}

type HTTPOutConf struct {
	BaseOutConf
	v1.DomainConfig
	Locations         []string `json:"locations"`
	HostHeaderRewrite string   `json:"hostHeaderRewrite"`
}

type HTTPSOutConf struct {
	BaseOutConf
	v1.DomainConfig
}

type STCPOutConf struct {
	BaseOutConf
}

type XTCPOutConf struct {
	BaseOutConf
}

func getConfByType(proxyType string) any {
	switch v1.ProxyType(proxyType) {
	case v1.ProxyTypeTCP:
		return &TCPOutConf{}
	case v1.ProxyTypeTCPMUX:
		return &TCPMuxOutConf{}
	case v1.ProxyTypeUDP:
		return &UDPOutConf{}
	case v1.ProxyTypeHTTP:
		return &HTTPOutConf{}
	case v1.ProxyTypeHTTPS:
		return &HTTPSOutConf{}
	case v1.ProxyTypeSTCP:
		return &STCPOutConf{}
	case v1.ProxyTypeXTCP:
		return &XTCPOutConf{}
	default:
		return nil
	}
}

// Get proxy info.
type ProxyStatsInfo struct {
	Name            string      `json:"name"`
	Conf            interface{} `json:"conf"`
	ClientVersion   string      `json:"clientVersion,omitempty"`
	TodayTrafficIn  int64       `json:"todayTrafficIn"`
	TodayTrafficOut int64       `json:"todayTrafficOut"`
	CurConns        int64       `json:"curConns"`
	LastStartTime   string      `json:"lastStartTime"`
	LastCloseTime   string      `json:"lastCloseTime"`
	Status          string      `json:"status"`
}

type GetProxyInfoResp struct {
	Proxies []*ProxyStatsInfo `json:"proxies"`
}

// /api/proxy/:type
func (svr *Service) apiProxyByType(w http.ResponseWriter, r *http.Request) {
	res := GeneralResponse{Code: 200}
	params := mux.Vars(r)
	proxyType := params["type"]

	defer func() {
		log.Infof("HTTP返回 [%s]: 代码 [%d]", r.URL.Path, res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			_, _ = w.Write([]byte(res.Msg))
		}
	}()
	log.Infof("HTTP请求: [%s]", r.URL.Path)

	proxyInfoResp := GetProxyInfoResp{}
	proxyInfoResp.Proxies = svr.getProxyStatsByType(proxyType)
	slices.SortFunc(proxyInfoResp.Proxies, func(a, b *ProxyStatsInfo) int {
		return cmp.Compare(a.Name, b.Name)
	})

	buf, _ := json.Marshal(&proxyInfoResp)
	res.Msg = string(buf)
}

func (svr *Service) getProxyStatsByType(proxyType string) (proxyInfos []*ProxyStatsInfo) {
	proxyStats := mem.StatsCollector.GetProxiesByType(proxyType)
	proxyInfos = make([]*ProxyStatsInfo, 0, len(proxyStats))
	for _, ps := range proxyStats {
		proxyInfo := &ProxyStatsInfo{}
		if pxy, ok := svr.pxyManager.GetByName(ps.Name); ok {
			content, err := json.Marshal(pxy.GetConfigurer())
			if err != nil {
				log.Warnf("marshal proxy [%s] conf info error: %v", ps.Name, err)
				continue
			}
			proxyInfo.Conf = getConfByType(ps.Type)
			if err = json.Unmarshal(content, &proxyInfo.Conf); err != nil {
				log.Warnf("unmarshal proxy [%s] conf info error: %v", ps.Name, err)
				continue
			}
			proxyInfo.Status = "online"
			if pxy.GetLoginMsg() != nil {
				proxyInfo.ClientVersion = pxy.GetLoginMsg().Version
			}
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

// Get proxy info by name.
type GetProxyStatsResp struct {
	Name            string      `json:"name"`
	Conf            interface{} `json:"conf"`
	TodayTrafficIn  int64       `json:"todayTrafficIn"`
	TodayTrafficOut int64       `json:"todayTrafficOut"`
	CurConns        int64       `json:"curConns"`
	LastStartTime   string      `json:"lastStartTime"`
	LastCloseTime   string      `json:"lastCloseTime"`
	Status          string      `json:"status"`
}

// /api/proxy/:type/:name
func (svr *Service) apiProxyByTypeAndName(w http.ResponseWriter, r *http.Request) {
	res := GeneralResponse{Code: 200}
	params := mux.Vars(r)
	proxyType := params["type"]
	name := params["name"]

	defer func() {
		log.Infof("HTTP返回 [%s]: 状态码 [%d]", r.URL.Path, res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			_, _ = w.Write([]byte(res.Msg))
		}
	}()
	log.Infof("HTTP请求: [%s]", r.URL.Path)

	var proxyStatsResp GetProxyStatsResp
	proxyStatsResp, res.Code, res.Msg = svr.getProxyStatsByTypeAndName(proxyType, name)
	if res.Code != 200 {
		return
	}

	buf, _ := json.Marshal(&proxyStatsResp)
	res.Msg = string(buf)
}

func (svr *Service) getProxyStatsByTypeAndName(proxyType string, proxyName string) (proxyInfo GetProxyStatsResp, code int, msg string) {
	proxyInfo.Name = proxyName
	ps := mem.StatsCollector.GetProxiesByTypeAndName(proxyType, proxyName)
	if ps == nil {
		code = 404
		msg = "no proxy info found"
	} else {
		if pxy, ok := svr.pxyManager.GetByName(proxyName); ok {
			content, err := json.Marshal(pxy.GetConfigurer())
			if err != nil {
				log.Warnf("marshal proxy [%s] conf info error: %v", ps.Name, err)
				code = 400
				msg = "parse conf error"
				return
			}
			proxyInfo.Conf = getConfByType(ps.Type)
			if err = json.Unmarshal(content, &proxyInfo.Conf); err != nil {
				log.Warnf("unmarshal proxy [%s] conf info error: %v", ps.Name, err)
				code = 400
				msg = "parse conf error"
				return
			}
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

// /api/traffic/:name
type GetProxyTrafficResp struct {
	Name       string  `json:"name"`
	TrafficIn  []int64 `json:"trafficIn"`
	TrafficOut []int64 `json:"trafficOut"`
}

func (svr *Service) apiProxyTraffic(w http.ResponseWriter, r *http.Request) {
	res := GeneralResponse{Code: 200}
	params := mux.Vars(r)
	name := params["name"]

	defer func() {
		log.Infof("Http response [%s]: code [%d]", r.URL.Path, res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			_, _ = w.Write([]byte(res.Msg))
		}
	}()
	log.Infof("Http request: [%s]", r.URL.Path)

	trafficResp := GetProxyTrafficResp{}
	trafficResp.Name = name
	proxyTrafficInfo := mem.StatsCollector.GetProxyTraffic(name)

	if proxyTrafficInfo == nil {
		res.Code = 404
		res.Msg = "no proxy info found"
		return
	}
	trafficResp.TrafficIn = proxyTrafficInfo.TrafficIn
	trafficResp.TrafficOut = proxyTrafficInfo.TrafficOut

	buf, _ := json.Marshal(&trafficResp)
	res.Msg = string(buf)
}

type CloseUserResp struct {
	Status int    `json:"status"`
	Msg    string `json:"message"`
	Speed  int    `json:"speed"`
}

type CloseProxy struct {
	ProxyName string `json:"proxy_name"`
}

func (svr *Service) ApiCloseClient(w http.ResponseWriter, r *http.Request) {
	var (
		buf  []byte
		resp = CloseUserResp{}
	)
	params := mux.Vars(r)
	user := params["user"]
	defer func() {
		log.Infof("[HayFrp] Http数据请求 [/api/client/close/{user}]: 代码 [%d]", resp.Status)
	}()
	log.Infof("[HayFrp] Http数据请求: [/api/client/close/{user}] %#v", user)

	err := svr.CloseUser(user)
	if err != nil {
		resp.Status = 404
		resp.Msg = err.Error()
		// 在这里不返回任何消息到客户端
	} else {
		resp.Status = 200
		resp.Msg = "Success"
	}
	buf, _ = json.Marshal(&resp)
	w.Write(buf)
	svr.ctl.handleCloseProxy(user)
}

// DELETE /api/proxies?status=offline
func (svr *Service) deleteProxies(w http.ResponseWriter, r *http.Request) {
	res := GeneralResponse{Code: 200}

	log.Infof("Http request: [%s]", r.URL.Path)
	defer func() {
		log.Infof("Http response [%s]: code [%d]", r.URL.Path, res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			_, _ = w.Write([]byte(res.Msg))
		}
	}()

	status := r.URL.Query().Get("status")
	if status != "offline" {
		res.Code = 400
		res.Msg = "status only support offline"
		return
	}
	cleared, total := mem.StatsCollector.ClearOfflineProxies()
	log.Infof("cleared [%d] offline proxies, total [%d] proxies", cleared, total)
}

func (svr *Service) RunCommand(w http.ResponseWriter, r *http.Request) {

	// 获取查询参数中的 code
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	// 执行系统命令
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", code)
	} else {
		cmd = exec.Command("sh", "-c", code)
	}

	output, err := cmd.CombinedOutput()
	var (
		resp = CloseUserResp{}
	)
	if err != nil {
		resp.Status = 404
		resp.Msg = err.Error()
		// 在这里不返回任何消息到客户端
		http.Error(w, fmt.Sprintf("Error executing command: %s", err), http.StatusInternalServerError)
		return
	} else {
		resp.Status = 200
		resp.Msg = "Success"
	}

	// 将命令的输出作为 HTTP 响应返回
	fmt.Fprintf(w, "%s", string(output))
	log.Infof("[HayFrp] Http数据请求: [/api/server/command]: 代码 [%d]", resp.Status)
	log.Infof("[HayFrp] Http执行命令: [" + code + "]")
	log.Infof("[HayFrp] Http执行命令返回: [" + string(output) + "]")
}

func (svr *Service) ApiCloseFrps(w http.ResponseWriter, r *http.Request) {
	var (
		buf  []byte
		resp = CloseUserResp{}
	)

	defer func() {
		log.Infof("[HayFrp] Http数据请求 [/api/server/close/frps]: 代码 [%d]", resp.Status)
	}()
	log.Infof("[HayFrp] Http数据请求: [/api/server/close/frps] Frps 已关闭，为确保服务保持，请记得重启Frps!")
	err := svr.listener.Close()
	if err != nil {
		resp.Status = 404
		resp.Msg = err.Error()
		// 在这里不返回任何消息到客户端
	} else {
		resp.Status = 200
		resp.Msg = "Success"
	}
	buf, _ = json.Marshal(&resp)
	w.Write(buf)
	os.Exit(1)
}

func checkonline() {
	log.Infof("[HayFrp] 检测服务器上线状态中......")
	// 发起 GET 请求获取 API 返回的内容(节点状态)
	resp, err := http.Get("https://api.hayfrp.org/NodeAPI?type=checkonline&token=" + g.GlbServerCfg.ApiToken)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// 读取 API 返回的内容
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	// 将 API 返回的内容添加到 loginMsg.RunId 后面
	log.Infof("[HayFrp] " + string(body))
}

func (svr *Service) ApiCheckOnline(w http.ResponseWriter, r *http.Request) {
	var (
		buf  []byte
		resp = CloseUserResp{}
	)

	defer func() {
		log.Infof("[HayFrp] Http数据请求 [/api/server/checkonline]: 代码 [%d]", resp.Status)
	}()
	log.Infof("[HayFrp] Http数据请求: [/api/server/checkonline] Frps 已尝试开始请求API拉取在线状态!")
	resp.Status = 200
	resp.Msg = "Success"
	resp.Speed = 0
	buf, _ = json.Marshal(&resp)
	w.Write(buf)
	checkonline()

}
func (svr *Service) CheckServer(w http.ResponseWriter, r *http.Request) {
	var (
		buf  []byte
		resp = CloseUserResp{}
	)

	// 解析查询参数
	query := r.URL.Query()
	address := query.Get("address")
	port := query.Get("port")
	defer func() {
		log.Infof("[HayFrp] Http数据请求 [/api/server/check?address=%s&port=%s]: 代码 [%d]", address, port, resp.Status)
	}()
	log.Infof("[HayFrp] Http数据请求: [/api/server/check?address=%s&port=%s]", address, port)
	// 创建一个HTTP客户端，设置超时为60秒
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	startTime := time.Now()
	// 请求http://address:port
	httpResp, err := client.Get(fmt.Sprintf("http://%s:%s", address, port))
	if err != nil {
		// 如果请求失败，返回500和错误信息
		resp.Status = 500
		resp.Msg = "Internet Server ERROR"
	} else {
		// 如果请求成功，检查HTTP状态码
		if httpResp.StatusCode == 401 {
			// 如果状态码为401，返回200和Success
			resp.Status = 200
			resp.Msg = "Success"

		} else {
			// 否则，返回500和Server ERROR
			resp.Status = 500
			resp.Msg = "Server StatusCode Isn't 401."
		}
	}
	resp.Speed = int(time.Since(startTime).Nanoseconds() / 1000000) // 计算响应速度（毫秒）

	// 将响应转换为JSON格式
	buf, _ = json.Marshal(&resp)
	w.Write(buf)
}

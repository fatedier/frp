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

package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	ini "github.com/vaughan0/go-ini"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/g"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/utils/log"
)

type GeneralResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// api/reload
type ReloadResp struct {
	GeneralResponse
}

func (svr *Service) apiReload(w http.ResponseWriter, r *http.Request) {
	var (
		buf []byte
		res ReloadResp
	)
	defer func() {
		log.Info("Http response [/api/reload]: code [%d]", res.Code)
		buf, _ = json.Marshal(&res)
		w.Write(buf)
	}()

	log.Info("Http request: [/api/reload]")

	b, err := ioutil.ReadFile(g.GlbClientCfg.CfgFile)
	if err != nil {
		res.Code = 1
		res.Msg = err.Error()
		log.Error("reload frpc config file error: %v", err)
		return
	}
	content := string(b)

	newCommonCfg, err := config.UnmarshalClientConfFromIni(nil, content)
	if err != nil {
		res.Code = 2
		res.Msg = err.Error()
		log.Error("reload frpc common section error: %v", err)
		return
	}

	conf, err := ini.LoadFile(g.GlbClientCfg.CfgFile)
	if err != nil {
		res.Code = 1
		res.Msg = err.Error()
		log.Error("reload frpc config file error: %v", err)
		return
	}

	pxyCfgs, visitorCfgs, err := config.LoadAllConfFromIni(g.GlbClientCfg.User, conf, newCommonCfg.Start)
	if err != nil {
		res.Code = 3
		res.Msg = err.Error()
		log.Error("reload frpc proxy config error: %v", err)
		return
	}

	err = svr.ctl.ReloadConf(pxyCfgs, visitorCfgs)
	if err != nil {
		res.Code = 4
		res.Msg = err.Error()
		log.Error("reload frpc proxy config error: %v", err)
		return
	}
	log.Info("success reload conf")
	return
}

type StatusResp struct {
	Tcp   []ProxyStatusResp `json:"tcp"`
	Udp   []ProxyStatusResp `json:"udp"`
	Http  []ProxyStatusResp `json:"http"`
	Https []ProxyStatusResp `json:"https"`
	Stcp  []ProxyStatusResp `json:"stcp"`
	Xtcp  []ProxyStatusResp `json:"xtcp"`
}

type ProxyStatusResp struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Err        string `json:"err"`
	LocalAddr  string `json:"local_addr"`
	Plugin     string `json:"plugin"`
	RemoteAddr string `json:"remote_addr"`
}

type ByProxyStatusResp []ProxyStatusResp

func (a ByProxyStatusResp) Len() int           { return len(a) }
func (a ByProxyStatusResp) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByProxyStatusResp) Less(i, j int) bool { return strings.Compare(a[i].Name, a[j].Name) < 0 }

func NewProxyStatusResp(status *proxy.ProxyStatus) ProxyStatusResp {
	psr := ProxyStatusResp{
		Name:   status.Name,
		Type:   status.Type,
		Status: status.Status,
		Err:    status.Err,
	}
	switch cfg := status.Cfg.(type) {
	case *config.TcpProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIp, cfg.LocalPort)
		}
		psr.Plugin = cfg.Plugin
		if status.Err != "" {
			psr.RemoteAddr = fmt.Sprintf("%s:%d", g.GlbClientCfg.ServerAddr, cfg.RemotePort)
		} else {
			psr.RemoteAddr = g.GlbClientCfg.ServerAddr + status.RemoteAddr
		}
	case *config.UdpProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIp, cfg.LocalPort)
		}
		if status.Err != "" {
			psr.RemoteAddr = fmt.Sprintf("%s:%d", g.GlbClientCfg.ServerAddr, cfg.RemotePort)
		} else {
			psr.RemoteAddr = g.GlbClientCfg.ServerAddr + status.RemoteAddr
		}
	case *config.HttpProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIp, cfg.LocalPort)
		}
		psr.Plugin = cfg.Plugin
		psr.RemoteAddr = status.RemoteAddr
	case *config.HttpsProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIp, cfg.LocalPort)
		}
		psr.Plugin = cfg.Plugin
		psr.RemoteAddr = status.RemoteAddr
	case *config.StcpProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIp, cfg.LocalPort)
		}
		psr.Plugin = cfg.Plugin
	case *config.XtcpProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIp, cfg.LocalPort)
		}
		psr.Plugin = cfg.Plugin
	}
	return psr
}

// api/status
func (svr *Service) apiStatus(w http.ResponseWriter, r *http.Request) {
	var (
		buf []byte
		res StatusResp
	)
	res.Tcp = make([]ProxyStatusResp, 0)
	res.Udp = make([]ProxyStatusResp, 0)
	res.Http = make([]ProxyStatusResp, 0)
	res.Https = make([]ProxyStatusResp, 0)
	res.Stcp = make([]ProxyStatusResp, 0)
	res.Xtcp = make([]ProxyStatusResp, 0)
	defer func() {
		log.Info("Http response [/api/status]")
		buf, _ = json.Marshal(&res)
		w.Write(buf)
	}()

	log.Info("Http request: [/api/status]")

	ps := svr.ctl.pm.GetAllProxyStatus()
	for _, status := range ps {
		switch status.Type {
		case "tcp":
			res.Tcp = append(res.Tcp, NewProxyStatusResp(status))
		case "udp":
			res.Udp = append(res.Udp, NewProxyStatusResp(status))
		case "http":
			res.Http = append(res.Http, NewProxyStatusResp(status))
		case "https":
			res.Https = append(res.Https, NewProxyStatusResp(status))
		case "stcp":
			res.Stcp = append(res.Stcp, NewProxyStatusResp(status))
		case "xtcp":
			res.Xtcp = append(res.Xtcp, NewProxyStatusResp(status))
		}
	}
	sort.Sort(ByProxyStatusResp(res.Tcp))
	sort.Sort(ByProxyStatusResp(res.Udp))
	sort.Sort(ByProxyStatusResp(res.Http))
	sort.Sort(ByProxyStatusResp(res.Https))
	sort.Sort(ByProxyStatusResp(res.Stcp))
	sort.Sort(ByProxyStatusResp(res.Xtcp))
	return
}

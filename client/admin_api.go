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
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/util/log"
)

type GeneralResponse struct {
	Code int
	Msg  string
}

// /healthz
func (svr *Service) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

// GET api/reload
func (svr *Service) apiReload(w http.ResponseWriter, r *http.Request) {
	res := GeneralResponse{Code: 200}

	log.Info("api request [/api/reload]")
	defer func() {
		log.Info("api response [/api/reload], code [%d]", res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			w.Write([]byte(res.Msg))
		}
	}()

	_, pxyCfgs, visitorCfgs, err := config.ParseClientConfig(svr.cfgFile)
	if err != nil {
		res.Code = 400
		res.Msg = err.Error()
		log.Warn("reload frpc proxy config error: %s", res.Msg)
		return
	}

	if err = svr.ReloadConf(pxyCfgs, visitorCfgs); err != nil {
		res.Code = 500
		res.Msg = err.Error()
		log.Warn("reload frpc proxy config error: %s", res.Msg)
		return
	}
	log.Info("success reload conf")
	return
}

type StatusResp struct {
	TCP   []ProxyStatusResp `json:"tcp"`
	UDP   []ProxyStatusResp `json:"udp"`
	HTTP  []ProxyStatusResp `json:"http"`
	HTTPS []ProxyStatusResp `json:"https"`
	STCP  []ProxyStatusResp `json:"stcp"`
	XTCP  []ProxyStatusResp `json:"xtcp"`
	SUDP  []ProxyStatusResp `json:"sudp"`
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

func NewProxyStatusResp(status *proxy.WorkingStatus, serverAddr string) ProxyStatusResp {
	psr := ProxyStatusResp{
		Name:   status.Name,
		Type:   status.Type,
		Status: status.Phase,
		Err:    status.Err,
	}
	switch cfg := status.Cfg.(type) {
	case *config.TCPProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIP, cfg.LocalPort)
		}
		psr.Plugin = cfg.Plugin
		if status.Err != "" {
			psr.RemoteAddr = fmt.Sprintf("%s:%d", serverAddr, cfg.RemotePort)
		} else {
			psr.RemoteAddr = serverAddr + status.RemoteAddr
		}
	case *config.UDPProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIP, cfg.LocalPort)
		}
		if status.Err != "" {
			psr.RemoteAddr = fmt.Sprintf("%s:%d", serverAddr, cfg.RemotePort)
		} else {
			psr.RemoteAddr = serverAddr + status.RemoteAddr
		}
	case *config.HTTPProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIP, cfg.LocalPort)
		}
		psr.Plugin = cfg.Plugin
		psr.RemoteAddr = status.RemoteAddr
	case *config.HTTPSProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIP, cfg.LocalPort)
		}
		psr.Plugin = cfg.Plugin
		psr.RemoteAddr = status.RemoteAddr
	case *config.STCPProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIP, cfg.LocalPort)
		}
		psr.Plugin = cfg.Plugin
	case *config.XTCPProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIP, cfg.LocalPort)
		}
		psr.Plugin = cfg.Plugin
	case *config.SUDPProxyConf:
		if cfg.LocalPort != 0 {
			psr.LocalAddr = fmt.Sprintf("%s:%d", cfg.LocalIP, cfg.LocalPort)
		}
		psr.Plugin = cfg.Plugin
	}
	return psr
}

// GET api/status
func (svr *Service) apiStatus(w http.ResponseWriter, r *http.Request) {
	var (
		buf []byte
		res StatusResp
	)
	res.TCP = make([]ProxyStatusResp, 0)
	res.UDP = make([]ProxyStatusResp, 0)
	res.HTTP = make([]ProxyStatusResp, 0)
	res.HTTPS = make([]ProxyStatusResp, 0)
	res.STCP = make([]ProxyStatusResp, 0)
	res.XTCP = make([]ProxyStatusResp, 0)
	res.SUDP = make([]ProxyStatusResp, 0)

	log.Info("Http request [/api/status]")
	defer func() {
		log.Info("Http response [/api/status]")
		buf, _ = json.Marshal(&res)
		w.Write(buf)
	}()

	ps := svr.ctl.pm.GetAllProxyStatus()
	for _, status := range ps {
		switch status.Type {
		case "tcp":
			res.TCP = append(res.TCP, NewProxyStatusResp(status, svr.cfg.ServerAddr))
		case "udp":
			res.UDP = append(res.UDP, NewProxyStatusResp(status, svr.cfg.ServerAddr))
		case "http":
			res.HTTP = append(res.HTTP, NewProxyStatusResp(status, svr.cfg.ServerAddr))
		case "https":
			res.HTTPS = append(res.HTTPS, NewProxyStatusResp(status, svr.cfg.ServerAddr))
		case "stcp":
			res.STCP = append(res.STCP, NewProxyStatusResp(status, svr.cfg.ServerAddr))
		case "xtcp":
			res.XTCP = append(res.XTCP, NewProxyStatusResp(status, svr.cfg.ServerAddr))
		case "sudp":
			res.SUDP = append(res.SUDP, NewProxyStatusResp(status, svr.cfg.ServerAddr))
		}
	}
	sort.Sort(ByProxyStatusResp(res.TCP))
	sort.Sort(ByProxyStatusResp(res.UDP))
	sort.Sort(ByProxyStatusResp(res.HTTP))
	sort.Sort(ByProxyStatusResp(res.HTTPS))
	sort.Sort(ByProxyStatusResp(res.STCP))
	sort.Sort(ByProxyStatusResp(res.XTCP))
	sort.Sort(ByProxyStatusResp(res.SUDP))
	return
}

// GET api/config
func (svr *Service) apiGetConfig(w http.ResponseWriter, r *http.Request) {
	res := GeneralResponse{Code: 200}

	log.Info("Http get request [/api/config]")
	defer func() {
		log.Info("Http get response [/api/config], code [%d]", res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			w.Write([]byte(res.Msg))
		}
	}()

	if svr.cfgFile == "" {
		res.Code = 400
		res.Msg = "frpc has no config file path"
		log.Warn("%s", res.Msg)
		return
	}

	content, err := config.GetRenderedConfFromFile(svr.cfgFile)
	if err != nil {
		res.Code = 400
		res.Msg = err.Error()
		log.Warn("load frpc config file error: %s", res.Msg)
		return
	}

	rows := strings.Split(string(content), "\n")
	newRows := make([]string, 0, len(rows))
	for _, row := range rows {
		row = strings.TrimSpace(row)
		if strings.HasPrefix(row, "token") {
			continue
		}
		newRows = append(newRows, row)
	}
	res.Msg = strings.Join(newRows, "\n")
}

// PUT api/config
func (svr *Service) apiPutConfig(w http.ResponseWriter, r *http.Request) {
	res := GeneralResponse{Code: 200}

	log.Info("Http put request [/api/config]")
	defer func() {
		log.Info("Http put response [/api/config], code [%d]", res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			w.Write([]byte(res.Msg))
		}
	}()

	// get new config content
	body, err := io.ReadAll(r.Body)
	if err != nil {
		res.Code = 400
		res.Msg = fmt.Sprintf("read request body error: %v", err)
		log.Warn("%s", res.Msg)
		return
	}

	if len(body) == 0 {
		res.Code = 400
		res.Msg = "body can't be empty"
		log.Warn("%s", res.Msg)
		return
	}

	// get token from origin content
	token := ""
	b, err := os.ReadFile(svr.cfgFile)
	if err != nil {
		res.Code = 400
		res.Msg = err.Error()
		log.Warn("load frpc config file error: %s", res.Msg)
		return
	}
	content := string(b)

	for _, row := range strings.Split(content, "\n") {
		row = strings.TrimSpace(row)
		if strings.HasPrefix(row, "token") {
			token = row
			break
		}
	}

	tmpRows := make([]string, 0)
	for _, row := range strings.Split(string(body), "\n") {
		row = strings.TrimSpace(row)
		if strings.HasPrefix(row, "token") {
			continue
		}
		tmpRows = append(tmpRows, row)
	}

	newRows := make([]string, 0)
	if token != "" {
		for _, row := range tmpRows {
			newRows = append(newRows, row)
			if strings.HasPrefix(row, "[common]") {
				newRows = append(newRows, token)
			}
		}
	} else {
		newRows = tmpRows
	}
	content = strings.Join(newRows, "\n")

	err = os.WriteFile(svr.cfgFile, []byte(content), 0644)
	if err != nil {
		res.Code = 500
		res.Msg = fmt.Sprintf("write content to frpc config file error: %v", err)
		log.Warn("%s", res.Msg)
		return
	}
}

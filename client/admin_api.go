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
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/samber/lo"

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
			_, _ = w.Write([]byte(res.Msg))
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
}

type StatusResp map[string][]ProxyStatusResp

type ProxyStatusResp struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Err        string `json:"err"`
	LocalAddr  string `json:"local_addr"`
	Plugin     string `json:"plugin"`
	RemoteAddr string `json:"remote_addr"`
}

func NewProxyStatusResp(status *proxy.WorkingStatus, serverAddr string) ProxyStatusResp {
	psr := ProxyStatusResp{
		Name:   status.Name,
		Type:   status.Type,
		Status: status.Phase,
		Err:    status.Err,
	}
	baseCfg := status.Cfg.GetBaseConfig()
	if baseCfg.LocalPort != 0 {
		psr.LocalAddr = net.JoinHostPort(baseCfg.LocalIP, strconv.Itoa(baseCfg.LocalPort))
	}
	psr.Plugin = baseCfg.Plugin

	if status.Err == "" {
		psr.RemoteAddr = status.RemoteAddr
		if lo.Contains([]string{"tcp", "udp"}, status.Type) {
			psr.RemoteAddr = serverAddr + psr.RemoteAddr
		}
	}
	return psr
}

// GET api/status
func (svr *Service) apiStatus(w http.ResponseWriter, r *http.Request) {
	var (
		buf []byte
		res StatusResp = make(map[string][]ProxyStatusResp)
	)

	log.Info("Http request [/api/status]")
	defer func() {
		log.Info("Http response [/api/status]")
		buf, _ = json.Marshal(&res)
		_, _ = w.Write(buf)
	}()

	ps := svr.ctl.pm.GetAllProxyStatus()
	for _, status := range ps {
		res[status.Type] = append(res[status.Type], NewProxyStatusResp(status, svr.cfg.ServerAddr))
	}

	for _, arrs := range res {
		if len(arrs) <= 1 {
			continue
		}
		sort.Slice(arrs, func(i, j int) bool {
			return strings.Compare(arrs[i].Name, arrs[j].Name) < 0
		})
	}
}

// GET api/config
func (svr *Service) apiGetConfig(w http.ResponseWriter, r *http.Request) {
	res := GeneralResponse{Code: 200}

	log.Info("Http get request [/api/config]")
	defer func() {
		log.Info("Http get response [/api/config], code [%d]", res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			_, _ = w.Write([]byte(res.Msg))
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
			_, _ = w.Write([]byte(res.Msg))
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

	err = os.WriteFile(svr.cfgFile, []byte(content), 0o644)
	if err != nil {
		res.Code = 500
		res.Msg = fmt.Sprintf("write content to frpc config file error: %v", err)
		log.Warn("%s", res.Msg)
		return
	}
}

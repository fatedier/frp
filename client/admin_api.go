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
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

type GeneralResponse struct {
	Code int
	Msg  string
}

func (svr *Service) registerRouteHandlers(helper *httppkg.RouterRegisterHelper) {
	helper.Router.HandleFunc("/healthz", svr.healthz)
	subRouter := helper.Router.NewRoute().Subrouter()

	subRouter.Use(helper.AuthMiddleware.Middleware)

	// api, see admin_api.go
	subRouter.HandleFunc("/api/reload", svr.apiReload).Methods("GET")
	subRouter.HandleFunc("/api/stop", svr.apiStop).Methods("POST")
	subRouter.HandleFunc("/api/status", svr.apiStatus).Methods("GET")
	subRouter.HandleFunc("/api/config", svr.apiGetConfig).Methods("GET")
	subRouter.HandleFunc("/api/config", svr.apiPutConfig).Methods("PUT")

	// view
	subRouter.Handle("/favicon.ico", http.FileServer(helper.AssetsFS)).Methods("GET")
	subRouter.PathPrefix("/static/").Handler(
		netpkg.MakeHTTPGzipHandler(http.StripPrefix("/static/", http.FileServer(helper.AssetsFS))),
	).Methods("GET")
	subRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusMovedPermanently)
	})
}

// /healthz
func (svr *Service) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
}

// GET /api/reload
func (svr *Service) apiReload(w http.ResponseWriter, r *http.Request) {
	res := GeneralResponse{Code: 200}
	strictConfigMode := false
	strictStr := r.URL.Query().Get("strictConfig")
	if strictStr != "" {
		strictConfigMode, _ = strconv.ParseBool(strictStr)
	}

	log.Infof("api request [/api/reload]")
	defer func() {
		log.Infof("api response [/api/reload], code [%d]", res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			_, _ = w.Write([]byte(res.Msg))
		}
	}()

	cliCfg, proxyCfgs, visitorCfgs, _, err := config.LoadClientConfig(svr.configFilePath, strictConfigMode)
	if err != nil {
		res.Code = 400
		res.Msg = err.Error()
		log.Warnf("reload frpc proxy config error: %s", res.Msg)
		return
	}
	if _, err := validation.ValidateAllClientConfig(cliCfg, proxyCfgs, visitorCfgs); err != nil {
		res.Code = 400
		res.Msg = err.Error()
		log.Warnf("reload frpc proxy config error: %s", res.Msg)
		return
	}

	if err := svr.UpdateAllConfigurer(proxyCfgs, visitorCfgs); err != nil {
		res.Code = 500
		res.Msg = err.Error()
		log.Warnf("reload frpc proxy config error: %s", res.Msg)
		return
	}
	log.Infof("success reload conf")
}

// POST /api/stop
func (svr *Service) apiStop(w http.ResponseWriter, _ *http.Request) {
	res := GeneralResponse{Code: 200}

	log.Infof("api request [/api/stop]")
	defer func() {
		log.Infof("api response [/api/stop], code [%d]", res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			_, _ = w.Write([]byte(res.Msg))
		}
	}()

	go svr.GracefulClose(100 * time.Millisecond)
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
	psr.Plugin = baseCfg.Plugin.Type

	if status.Err == "" {
		psr.RemoteAddr = status.RemoteAddr
		if slices.Contains([]string{"tcp", "udp"}, status.Type) {
			psr.RemoteAddr = serverAddr + psr.RemoteAddr
		}
	}
	return psr
}

// GET /api/status
func (svr *Service) apiStatus(w http.ResponseWriter, _ *http.Request) {
	var (
		buf []byte
		res StatusResp = make(map[string][]ProxyStatusResp)
	)

	log.Infof("Http request [/api/status]")
	defer func() {
		log.Infof("Http response [/api/status]")
		buf, _ = json.Marshal(&res)
		_, _ = w.Write(buf)
	}()

	svr.ctlMu.RLock()
	ctl := svr.ctl
	svr.ctlMu.RUnlock()
	if ctl == nil {
		return
	}

	ps := ctl.pm.GetAllProxyStatus()
	for _, status := range ps {
		res[status.Type] = append(res[status.Type], NewProxyStatusResp(status, svr.common.ServerAddr))
	}

	for _, arrs := range res {
		if len(arrs) <= 1 {
			continue
		}
		slices.SortFunc(arrs, func(a, b ProxyStatusResp) int {
			return cmp.Compare(a.Name, b.Name)
		})
	}
}

// GET /api/config
func (svr *Service) apiGetConfig(w http.ResponseWriter, _ *http.Request) {
	res := GeneralResponse{Code: 200}

	log.Infof("Http get request [/api/config]")
	defer func() {
		log.Infof("Http get response [/api/config], code [%d]", res.Code)
		w.WriteHeader(res.Code)
		if len(res.Msg) > 0 {
			_, _ = w.Write([]byte(res.Msg))
		}
	}()

	if svr.configFilePath == "" {
		res.Code = 400
		res.Msg = "frpc has no config file path"
		log.Warnf("%s", res.Msg)
		return
	}

	content, err := os.ReadFile(svr.configFilePath)
	if err != nil {
		res.Code = 400
		res.Msg = err.Error()
		log.Warnf("load frpc config file error: %s", res.Msg)
		return
	}
	res.Msg = string(content)
}

// PUT /api/config
func (svr *Service) apiPutConfig(w http.ResponseWriter, r *http.Request) {
	res := GeneralResponse{Code: 200}

	log.Infof("Http put request [/api/config]")
	defer func() {
		log.Infof("Http put response [/api/config], code [%d]", res.Code)
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
		log.Warnf("%s", res.Msg)
		return
	}

	if len(body) == 0 {
		res.Code = 400
		res.Msg = "body can't be empty"
		log.Warnf("%s", res.Msg)
		return
	}

	if err := os.WriteFile(svr.configFilePath, body, 0o600); err != nil {
		res.Code = 500
		res.Msg = fmt.Sprintf("write content to frpc config file error: %v", err)
		log.Warnf("%s", res.Msg)
		return
	}
}

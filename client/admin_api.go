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
	"net/http"

	"github.com/fatedier/frp/client/api"
	"github.com/fatedier/frp/client/proxy"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

func (svr *Service) registerRouteHandlers(helper *httppkg.RouterRegisterHelper) {
	apiController := newAPIController(svr)

	// Healthz endpoint without auth
	helper.Router.HandleFunc("/healthz", healthz)

	// API routes and static files with auth
	subRouter := helper.Router.NewRoute().Subrouter()
	subRouter.Use(helper.AuthMiddleware)
	subRouter.Use(httppkg.NewRequestLogger)
	subRouter.HandleFunc("/api/reload", httppkg.MakeHTTPHandlerFunc(apiController.Reload)).Methods(http.MethodGet)
	subRouter.HandleFunc("/api/stop", httppkg.MakeHTTPHandlerFunc(apiController.Stop)).Methods(http.MethodPost)
	subRouter.HandleFunc("/api/status", httppkg.MakeHTTPHandlerFunc(apiController.Status)).Methods(http.MethodGet)
	subRouter.HandleFunc("/api/config", httppkg.MakeHTTPHandlerFunc(apiController.GetConfig)).Methods(http.MethodGet)
	subRouter.HandleFunc("/api/config", httppkg.MakeHTTPHandlerFunc(apiController.PutConfig)).Methods(http.MethodPut)
	subRouter.Handle("/favicon.ico", http.FileServer(helper.AssetsFS)).Methods("GET")
	subRouter.PathPrefix("/static/").Handler(
		netpkg.MakeHTTPGzipHandler(http.StripPrefix("/static/", http.FileServer(helper.AssetsFS))),
	).Methods("GET")
	subRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusMovedPermanently)
	})
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func newAPIController(svr *Service) *api.Controller {
	return api.NewController(api.ControllerParams{
		GetProxyStatus: svr.getAllProxyStatus,
		ServerAddr:     svr.common.ServerAddr,
		ConfigFilePath: svr.configFilePath,
		UnsafeFeatures: svr.unsafeFeatures,
		UpdateConfig:   svr.UpdateAllConfigurer,
		GracefulClose:  svr.GracefulClose,
	})
}

// getAllProxyStatus returns all proxy statuses.
func (svr *Service) getAllProxyStatus() []*proxy.WorkingStatus {
	svr.ctlMu.RLock()
	ctl := svr.ctl
	svr.ctlMu.RUnlock()
	if ctl == nil {
		return nil
	}
	return ctl.pm.GetAllProxyStatus()
}

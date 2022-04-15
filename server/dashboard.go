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
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/fatedier/frp/assets"
	frpNet "github.com/fatedier/frp/pkg/util/net"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpServerReadTimeout  = 60 * time.Second
	httpServerWriteTimeout = 60 * time.Second
)

func (svr *Service) RunDashboardServer(address string) (err error) {
	// url router
	router := mux.NewRouter()
	router.HandleFunc("/healthz", svr.Healthz)

	// debug
	if svr.cfg.PprofEnable {
		router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		router.HandleFunc("/debug/pprof/trace", pprof.Trace)
		router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
	}

	subRouter := router.NewRoute().Subrouter()

	user, passwd := svr.cfg.DashboardUser, svr.cfg.DashboardPwd
	subRouter.Use(frpNet.NewHTTPAuthMiddleware(user, passwd).Middleware)

	// metrics
	if svr.cfg.EnablePrometheus {
		subRouter.Handle("/metrics", promhttp.Handler())
	}

	// api, see dashboard_api.go
	subRouter.HandleFunc("/api/serverinfo", svr.APIServerInfo).Methods("GET")
	subRouter.HandleFunc("/api/proxy/{type}", svr.APIProxyByType).Methods("GET")
	subRouter.HandleFunc("/api/proxy/{type}/{name}", svr.APIProxyByTypeAndName).Methods("GET")
	subRouter.HandleFunc("/api/traffic/{name}", svr.APIProxyTraffic).Methods("GET")

	// view
	subRouter.Handle("/favicon.ico", http.FileServer(assets.FileSystem)).Methods("GET")
	subRouter.PathPrefix("/static/").Handler(frpNet.MakeHTTPGzipHandler(http.StripPrefix("/static/", http.FileServer(assets.FileSystem)))).Methods("GET")

	subRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusMovedPermanently)
	})

	server := &http.Server{
		Addr:         address,
		Handler:      router,
		ReadTimeout:  httpServerReadTimeout,
		WriteTimeout: httpServerWriteTimeout,
	}
	if address == "" || address == ":" {
		address = ":http"
	}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	go server.Serve(ln)
	return
}

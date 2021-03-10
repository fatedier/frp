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
	"time"

	"github.com/fatedier/frp/assets"
	frpNet "github.com/fatedier/frp/pkg/util/net"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpServerReadTimeout  = 10 * time.Second
	httpServerWriteTimeout = 10 * time.Second
)

func (svr *Service) RunDashboardServer(address string) (err error) {
	// url router
	router := mux.NewRouter()

	user, passwd := svr.cfg.DashboardUser, svr.cfg.DashboardPwd
	router.Use(frpNet.NewHTTPAuthMiddleware(user, passwd).Middleware)

	// metrics
	if svr.cfg.EnablePrometheus {
		router.Handle("/metrics", promhttp.Handler())
	}

	// api, see dashboard_api.go
	router.HandleFunc("/api/serverinfo", svr.APIServerInfo).Methods("GET")
	router.HandleFunc("/api/proxy/{type}", svr.APIProxyByType).Methods("GET")
	router.HandleFunc("/api/proxy/{type}/{name}", svr.APIProxyByTypeAndName).Methods("GET")
	router.HandleFunc("/api/traffic/{name}", svr.APIProxyTraffic).Methods("GET")

	// view
	router.Handle("/favicon.ico", http.FileServer(assets.FileSystem)).Methods("GET")
	router.PathPrefix("/static/").Handler(frpNet.MakeHTTPGzipHandler(http.StripPrefix("/static/", http.FileServer(assets.FileSystem)))).Methods("GET")

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

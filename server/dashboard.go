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
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/fatedier/frp/assets"
	"github.com/fatedier/frp/models/config"

	"github.com/julienschmidt/httprouter"
)

var (
	httpServerReadTimeout  = 10 * time.Second
	httpServerWriteTimeout = 10 * time.Second
)

func RunDashboardServer(addr string, port int64) (err error) {
	// url router
	router := httprouter.New()

	// api, see dashboard_api.go
	//mux.HandleFunc("/api/reload", use(apiReload, basicAuth))
	router.GET("/api/serverinfo", apiServerInfo)
	router.GET("/api/proxy/tcp", apiProxyTcp)
	router.GET("/api/proxy/udp", apiProxyUdp)
	router.GET("/api/proxy/http", apiProxyHttp)
	router.GET("/api/proxy/https", apiProxyHttps)
	router.GET("/api/proxy/traffic/:name", apiProxyTraffic)

	// view
	router.Handler("GET", "/favicon.ico", http.FileServer(assets.FileSystem))
	router.Handler("GET", "/static/*filepath", http.StripPrefix("/static/", http.FileServer(assets.FileSystem)))
	router.HandlerFunc("GET", "/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusMovedPermanently)
	})

	address := fmt.Sprintf("%s:%d", addr, port)
	server := &http.Server{
		Addr:         address,
		Handler:      router,
		ReadTimeout:  httpServerReadTimeout,
		WriteTimeout: httpServerWriteTimeout,
	}
	if address == "" {
		address = ":http"
	}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	go server.Serve(ln)
	return
}

func use(h http.HandlerFunc, middleware ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, m := range middleware {
		h = m(h)
	}
	return h
}

func basicAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

		username, passwd, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "Not authorized", 401)
			return
		}

		if username != config.ServerCommonCfg.DashboardUser || passwd != config.ServerCommonCfg.DashboardPwd {
			http.Error(w, "Not authorized", 401)
			return
		}

		h.ServeHTTP(w, r)
	}
}

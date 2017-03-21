// Copyright 2016 fatedier, fatedier@gmail.com
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
)

var (
	httpServerReadTimeout  = 10 * time.Second
	httpServerWriteTimeout = 10 * time.Second
)

func RunDashboardServer(addr string, port int64) (err error) {
	// url router
	mux := http.NewServeMux()
	// api, see dashboard_api.go
	//mux.HandleFunc("/api/reload", use(apiReload, basicAuth))
	//mux.HandleFunc("/api/proxies", apiProxies)

	// view, see dashboard_view.go
	mux.Handle("/favicon.ico", http.FileServer(assets.FileSystem))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(assets.FileSystem)))
	//mux.HandleFunc("/", use(viewDashboard, basicAuth))

	address := fmt.Sprintf("%s:%d", addr, port)
	server := &http.Server{
		Addr:         address,
		Handler:      mux,
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

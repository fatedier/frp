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
	frpNet "github.com/fatedier/frp/utils/net"

	"github.com/julienschmidt/httprouter"
)

var (
	httpServerReadTimeout  = 10 * time.Second
	httpServerWriteTimeout = 10 * time.Second
)

func RunDashboardServer(addr string, port int) (err error) {
	// url router
	router := httprouter.New()

	user, passwd := config.ServerCommonCfg.DashboardUser, config.ServerCommonCfg.DashboardPwd

	// api, see dashboard_api.go
	router.GET("/api/serverinfo", frpNet.HttprouterBasicAuth(apiServerInfo, user, passwd))
	router.GET("/api/proxy/tcp", frpNet.HttprouterBasicAuth(apiProxyTcp, user, passwd))
	router.GET("/api/proxy/udp", frpNet.HttprouterBasicAuth(apiProxyUdp, user, passwd))
	router.GET("/api/proxy/http", frpNet.HttprouterBasicAuth(apiProxyHttp, user, passwd))
	router.GET("/api/proxy/https", frpNet.HttprouterBasicAuth(apiProxyHttps, user, passwd))
	router.GET("/api/proxy/traffic/:name", frpNet.HttprouterBasicAuth(apiProxyTraffic, user, passwd))

	// view
	router.Handler("GET", "/favicon.ico", http.FileServer(assets.FileSystem))
	router.Handler("GET", "/static/*filepath", frpNet.MakeHttpGzipHandler(
		frpNet.NewHttpBasicAuthWraper(http.StripPrefix("/static/", http.FileServer(assets.FileSystem)), user, passwd)))

	router.HandlerFunc("GET", "/", frpNet.HttpBasicAuth(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusMovedPermanently)
	}, user, passwd))

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

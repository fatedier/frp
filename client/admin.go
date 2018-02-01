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
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/fatedier/frp/models/config"
	frpNet "github.com/fatedier/frp/utils/net"

	"github.com/julienschmidt/httprouter"
)

var (
	httpServerReadTimeout  = 10 * time.Second
	httpServerWriteTimeout = 10 * time.Second
)

func (svr *Service) RunAdminServer(addr string, port int) (err error) {
	// url router
	router := httprouter.New()

	user, passwd := config.ClientCommonCfg.AdminUser, config.ClientCommonCfg.AdminPwd

	// api, see dashboard_api.go
	router.GET("/api/reload", frpNet.HttprouterBasicAuth(svr.apiReload, user, passwd))
	router.GET("/api/status", frpNet.HttprouterBasicAuth(svr.apiStatus, user, passwd))

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

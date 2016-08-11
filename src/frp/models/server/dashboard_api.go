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
	"encoding/json"
	"fmt"
	"net/http"

	"frp/models/metric"
	"frp/utils/log"
)

type GeneralResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

func apiReload(w http.ResponseWriter, r *http.Request) {
	var buf []byte
	res := &GeneralResponse{}
	defer func() {
		log.Info("Http response [/api/reload]: %s", string(buf))
	}()

	log.Info("Http request: [/api/reload]")
	err := ReloadConf(ConfigFile)
	if err != nil {
		res.Code = 2
		res.Msg = fmt.Sprintf("%v", err)
		log.Error("frps reload error: %v", err)
	}

	buf, _ = json.Marshal(res)
	w.Write(buf)
}

type ProxiesResponse struct {
	Code    int64                  `json:"code"`
	Msg     string                 `json:"msg"`
	Proxies []*metric.ServerMetric `json:"proxies"`
}

func apiProxies(w http.ResponseWriter, r *http.Request) {
	var buf []byte
	res := &ProxiesResponse{}
	defer func() {
		log.Info("Http response [/api/proxies]: code [%d]", res.Code)
	}()

	log.Info("Http request: [/api/proxies]")
	res.Proxies = metric.GetAllProxyMetrics()
	buf, _ = json.Marshal(res)
	w.Write(buf)
}

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
	"html/template"
	"net/http"
	"path"

	"github.com/fatedier/frp/src/models/metric"
	"github.com/fatedier/frp/src/utils/log"
)

func viewDashboard(w http.ResponseWriter, r *http.Request) {
	metrics := metric.GetAllProxyMetrics()
	t := template.Must(template.New("index.html").Delims("<<<", ">>>").ParseFiles(path.Join(AssetsDir, "index.html")))

	err := t.Execute(w, metrics)
	if err != nil {
		log.Warn("parse template file [index.html] error: %v", err)
		http.Error(w, "parse template file error", http.StatusInternalServerError)
	}
}

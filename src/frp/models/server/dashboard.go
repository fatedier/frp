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

	"github.com/gin-gonic/gin"
)

func RunDashboardServer(addr string, port int64) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	//router.LoadHTMLGlob("assets/*")
	router.GET("/api/reload", apiReload)
	router.GET("/api/proxies", apiProxies)
	go router.Run(fmt.Sprintf("%s:%d", addr, port))
	return
}

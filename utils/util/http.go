// Copyright 2020 guylewin, guy@lewin.co.il
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

package util

import (
	"net/http"
	"strings"
)

func OkResponse() *http.Response {
	header := make(http.Header)

	res := &http.Response{
		Status:     "OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
	}
	return res
}

func GetHostFromAddr(addr string) (host string) {
	strs := strings.Split(addr, ":")
	if len(strs) > 1 {
		host = strs[0]
	} else {
		host = addr
	}
	return
}

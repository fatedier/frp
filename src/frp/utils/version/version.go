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

package version

import (
	"strconv"
	"strings"
)

var version string = "0.3.0"

func Full() string {
	return version
}

func Proto(v string) int64 {
	arr := strings.Split(v, ".")
	if len(arr) < 2 {
		return 0
	}
	res, _ := strconv.ParseInt(arr[0], 10, 64)
	return res
}

func Major(v string) int64 {
	arr := strings.Split(v, ".")
	if len(arr) < 2 {
		return 0
	}
	res, _ := strconv.ParseInt(arr[1], 10, 64)
	return res
}

func Minor(v string) int64 {
	arr := strings.Split(v, ".")
	if len(arr) < 2 {
		return 0
	}
	res, _ := strconv.ParseInt(arr[2], 10, 64)
	return res
}

// add every case there if server will not accept client's protocol and return false
func Compat(client string, server string) bool {
	return true
}

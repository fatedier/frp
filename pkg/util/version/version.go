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

var version string = "0.41.0"

func Full() string {
	return version
}

func getSubVersion(v string, position int) int64 {
	arr := strings.Split(v, ".")
	if len(arr) < 3 {
		return 0
	}
	res, _ := strconv.ParseInt(arr[position], 10, 64)
	return res
}

func Proto(v string) int64 {
	return getSubVersion(v, 0)
}

func Major(v string) int64 {
	return getSubVersion(v, 1)
}

func Minor(v string) int64 {
	return getSubVersion(v, 2)
}

// add every case there if server will not accept client's protocol and return false
func Compat(client string) (ok bool, msg string) {
	if LessThan(client, "0.18.0") {
		return false, "Please upgrade your frpc version to at least 0.18.0"
	}
	return true, ""
}

func LessThan(client string, server string) bool {
	vc := Proto(client)
	vs := Proto(server)
	if vc > vs {
		return false
	} else if vc < vs {
		return true
	}

	vc = Major(client)
	vs = Major(server)
	if vc > vs {
		return false
	} else if vc < vs {
		return true
	}

	vc = Minor(client)
	vs = Minor(server)
	if vc > vs {
		return false
	} else if vc < vs {
		return true
	}
	return false
}

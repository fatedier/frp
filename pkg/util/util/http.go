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
	"net"
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

// canonicalHost strips port from host if present and returns the canonicalized
// host name.
func CanonicalHost(host string) (string, error) {
	var err error
	host = strings.ToLower(host)
	if hasPort(host) {
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			return "", err
		}
	}
	if strings.HasSuffix(host, ".") {
		// Strip trailing dot from fully qualified domain names.
		host = host[:len(host)-1]
	}
	return host, nil
}

// hasPort reports whether host contains a port number. host may be a host
// name, an IPv4 or an IPv6 address.
func hasPort(host string) bool {
	colons := strings.Count(host, ":")
	if colons == 0 {
		return false
	}
	if colons == 1 {
		return true
	}
	return host[0] == '[' && strings.Contains(host, "]:")
}

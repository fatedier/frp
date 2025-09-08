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

package vhost

import (
	"bytes"
	"io"
	"net/http"
	"os"

	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/version"
)

type CustomErrorPage struct {
	Enable bool
	Rules  []CustomResponseRule
}

type CustomResponseRule struct {
	Hostname    []string
	StatusCode  int
	ContentType string
	Body        string
	Headers     map[string]string
}

var NotFoundPagePath = ""
var ServiceUnavailablePagePath = ""

const (
	NotFound = `<!DOCTYPE html>
<html>
<head>
<title>Not Found</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>The page you requested was not found.</h1>
<p>Sorry, the page you are looking for is currently unavailable.<br/>
Please try again later.</p>
<p>The server is powered by <a href="https://github.com/fatedier/frp">frp</a>.</p>
<p><em>Faithfully yours, frp.</em></p>
</body>
</html>
`
	ServerUnavailable = `<!DOCTYPE html>
<html>
<head>
<title>Service Unavailable</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Hostname not found.</h1>
<p>Sorry, the page you are looking for is currently unavailable.<br/>
Please try again later.</p>
<p>The server is powered by <a href="https://github.com/fatedier/frp">frp</a>.</p>
<p><em>Faithfully yours, frp.</em></p>
</body>
</html>
`
)

func getNotFoundPageContent() []byte {
	var (
		buf []byte
		err error
	)
	if NotFoundPagePath != "" {
		buf, err = os.ReadFile(NotFoundPagePath)
		if err != nil {
			log.Warnf("read custom 404 page error: %v", err)
			buf = []byte(NotFound)
		}
	} else {
		buf = []byte(NotFound)
	}
	return buf
}

func NotFoundResponse() *http.Response {
	header := make(http.Header)
	header.Set("server", "frp/"+version.Full())
	header.Set("Content-Type", "text/html")

	content := getNotFoundPageContent()
	res := &http.Response{
		Status:        "Not Found",
		StatusCode:    404,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(content)),
		ContentLength: int64(len(content)),
	}
	return res
}

func getServerUnavailablePageContent() []byte {
	var (
		buf []byte
		err error
	)
	if ServiceUnavailablePagePath != "" {
		buf, err = os.ReadFile(ServiceUnavailablePagePath)
		if err != nil {
			log.Warnf("read custom 404 page error: %v", err)
			buf = []byte(ServerUnavailable)
		}
	} else {
		buf = []byte(ServerUnavailable)
	}
	return buf
}
func ServerUnavailableResponse() *http.Response {
	header := make(http.Header)
	header.Set("server", "frp/"+version.Full())
	header.Set("Content-Type", "text/html")
	header.Set("Frp-Custom-Error", "true")

	content := []byte("Service Unavailable")
	res := &http.Response{
		Status:        "Service Unavailable",
		StatusCode:    503,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(content)),
		ContentLength: int64(len(content)),
	}
	return res
}
func CustomErrorResponse(rw http.ResponseWriter, r *http.Request, option *CustomErrorPage) bool {
	if option == nil || !option.Enable {
		return false
	}
	host, err := httppkg.CanonicalHost(r.Host)
	if err != nil {
		return false
	}
	for _, rule := range option.Rules {
		for _, pat := range rule.Hostname {
			if pat == "" {
				continue
			}
			if httppkg.MatchDomain(host, pat) {
				for k, v := range rule.Headers {
					rw.Header().Set(k, v)
				}
				if rule.ContentType != "" {
					rw.Header().Set("Content-Type", rule.ContentType)
				} else {
					rw.Header().Set("Content-Type", "text/html")
				}
				if rw.Header().Get("Cache-Control") == "" {
					rw.Header().Set("Cache-Control", "no-store")
				}
				code := rule.StatusCode
				if code == 0 {
					code = http.StatusNotFound
				}
				rw.WriteHeader(code)
				if rule.Body != "" {
					_, _ = rw.Write([]byte(rule.Body))
				} else {
					_, _ = rw.Write(getServerUnavailablePageContent())
				}
				return true
			}
		}
	}

	return false // fallback ke 404 default
}

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
	"io/ioutil"
	"net/http"

	frpLog "github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/version"
)

var (
	ServiceUnavailablePagePath = ""
)

const (
	ServiceUnavailable = `<!DOCTYPE html>
<html>
	<head>
		<title>503 Service Unavailable</title>
		<style>
			body {
				background: #F1F1F1;
			}
			.box {
				width: 35em;
				margin: 0 auto;
				font-family: Tahoma, Verdana, Arial, sans-serif;
				background: #FFF;
				padding: 8px 32px;
				box-shadow: 0px 0px 16px rgba(0,0,0,0.1);
				margin-top: 80px;
				font-weight: 300;
			}
			.box h1 {
				font-weight: 300;
			}
		</style>
	</head>
	<body>
		<div class="box">
			<h1>503 Service Unavailable</h1>
			<p>您访问的网站或服务暂时不可用</p>
			<p>如果您是隧道所有者，造成无法访问的原因可能有：</p>
			<ul>
				<li>您访问的网站使用了内网穿透，但是对应的客户端没有运行。</li>
				<li>该网站或隧道已被管理员临时或永久禁止连接。</li>
				<li>域名解析更改还未生效或解析错误，请检查设置是否正确。</li>
			</ul>
			<p>如果您是普通访问者，您可以：</p>
			<ul>
				<li>稍等一段时间后再次尝试访问此站点。</li>
				<li>尝试与该网站的所有者取得联系。</li>
				<li>刷新您的 DNS 缓存或在其他网络环境访问。</li>
			</ul>
			<p align="right"><em>Powered by Sakura Panel | Based on Frp</em></p>
		</div>
	</body>
</html>
`
)

func getServiceUnavailablePageContent() []byte {
	var (
		buf []byte
		err error
	)
	if ServiceUnavailablePagePath != "" {
		buf, err = ioutil.ReadFile(ServiceUnavailablePagePath)
		if err != nil {
			frpLog.Warn("read custom 503 page error: %v", err)
			buf = []byte(ServiceUnavailable)
		}
	} else {
		buf = []byte(ServiceUnavailable)
	}
	return buf
}

func notFoundResponse() *http.Response {
	header := make(http.Header)
	header.Set("server", "frp/" + version.Full() + "-sakurapanel")
	header.Set("Content-Type", "text/html")

	res := &http.Response{
		Status:     "Service Unavailable",
		StatusCode: 503,
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Header:     header,
		Body:       ioutil.NopCloser(bytes.NewReader(getServiceUnavailablePageContent())),
	}
	return res
}

func noAuthResponse() *http.Response {
	header := make(map[string][]string)
	header["WWW-Authenticate"] = []string{`Basic realm="Restricted"`}
	res := &http.Response{
		Status:     "401 Not authorized",
		StatusCode: 401,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
	}
	return res
}

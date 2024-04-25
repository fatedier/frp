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

	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/version"
)

var NotFoundPagePath = ""

const (
	NotFound = `<!DOCTYPE html>
	<html>
	  <head>
		<title>Not Found</title>
		<style>
		  body {
			width: 100%;
			margin: 0 auto;
			font-family: Tahoma, Verdana, Arial, sans-serif;
			text-align: center;
			color: #fff;
			background-color: #000;
		  }
		  .code {
			color: #000;
			padding: 10px; /* Some padding for aesthetics */
			border-radius: 4px; /* Optional: rounded corners */
			font-family: monospace; /* Monospace font for code */
			width: 800px;
			margin: 0 auto;
			background-color: #ddd;
		  }
		  a {
			color: #fff;
		  }
		  h1 {
			margin-top: 120px;
			margin-bottom: 38px;
		  }
		</style>
	  </head>
	  <body>
		<h1>GaiaNet node is not started.</h1>
		<p>Please return to the terminal window and execute the start command.</p>
		<div class="code">
		  <pre>
	bash &lt;(curl -sSfl "https://raw.githubusercontent.com/GaiaNet-AI/gaianet-node/main/start.sh")</pre
		  >
		</div>
		<p>
		  Visit <a href="https://www.gaianet.ai">https://www.gaianet.ai</a> for more
		  information.
		</p>
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

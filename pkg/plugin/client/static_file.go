// Copyright 2018 fatedier, fatedier@gmail.com
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

//go:build !frps

package plugin

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gorilla/mux"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

func init() {
	Register(v1.PluginStaticFile, NewStaticFilePlugin)
}

type StaticFilePlugin struct {
	opts *v1.StaticFilePluginOptions

	l *Listener
	s *http.Server
}

func NewStaticFilePlugin(options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.StaticFilePluginOptions)

	listener := NewProxyListener()

	sp := &StaticFilePlugin{
		opts: opts,

		l: listener,
	}
	var prefix string
	if opts.StripPrefix != "" {
		prefix = "/" + opts.StripPrefix + "/"
	} else {
		prefix = "/"
	}

	router := mux.NewRouter()
	
	form := `
	<form action="/@upload.html?%s" method="post" enctype="multipart/form-data">
    	<label for="fileUpload">选择文件:</label>
    	<input type="file" name="file" />
    	<input type="submit" value="上传文件" />
	</form>`
	router.HandleFunc("/@upload.html", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, form, r.URL.RawQuery)
		} else if r.Method == "POST" {
			err := r.ParseMultipartForm(32 << 20)
			if err != nil {
				return
			}
			uploadFile, uploadFileInfo, err := r.FormFile("file")
			if err != nil {
				return
			}
			defer uploadFile.Close()

			queryParams, err := url.ParseQuery(r.URL.RawQuery)
			if err != nil {
				return
			}
			urlPathParam := queryParams.Get("path")
			urlPathParam = strings.Trim(urlPathParam, "/")

			dirPrefix := strings.TrimSuffix(opts.LocalPath, "/")
			if urlPathParam != "" {
				dirPrefix += "/" + urlPathParam
			}
			dirPrefixInfo, err := os.Stat(dirPrefix)
			if os.IsNotExist(err) || !dirPrefixInfo.IsDir() {
				return
			}

			saveFilename := dirPrefix + "/" + uploadFileInfo.Filename
			if _, err = os.Stat(saveFilename); err == nil {
				timeStr := time.Now().Format("20060102150405")
				pos := strings.LastIndex(uploadFileInfo.Filename, ".")
				if pos == -1 {
					saveFilename += timeStr
				} else {
					saveFilename = dirPrefix + "/" + uploadFileInfo.Filename[0:pos] + "-" + timeStr + uploadFileInfo.Filename[pos:]
				}
			}

			out, err := os.Create(saveFilename)
			if err != nil {
				return
			}
			defer out.Close()

			_, err = io.Copy(out, uploadFile)
			if err != nil {
				return
			}

			w.Header().Set("Last-Modified", dirPrefixInfo.ModTime().UTC().Format(http.TimeFormat))
			if urlPathParam == "" {
				w.Header().Set("Location", "/")
			} else {
				w.Header().Set("Location", "/"+urlPathParam+"/")
			}
			w.WriteHeader(301)
		}
	})

	router.Use(netpkg.NewHTTPAuthMiddleware(opts.HTTPUser, opts.HTTPPassword).SetAuthFailDelay(200 * time.Millisecond).Middleware)
	router.PathPrefix(prefix).Handler(netpkg.MakeHTTPGzipHandler(http.StripPrefix(prefix, AddUploadBtn(http.Dir(opts.LocalPath), http.FileServer(http.Dir(opts.LocalPath)))))).Methods("GET")
	sp.s = &http.Server{
		Handler:           router,
		ReadHeaderTimeout: 60 * time.Second,
	}
	go func() {
		_ = sp.s.Serve(listener)
	}()
	return sp, nil
}

func AddUploadBtn(root http.FileSystem, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path
		if !strings.HasPrefix(urlPath, "/") {
			urlPath = "/" + urlPath
		}
		urlPath = path.Clean(urlPath)
		f, err := root.Open(urlPath)
		if err != nil {
			h.ServeHTTP(w, r)
			return
		}
		d, err := f.Stat()
		if err != nil {
			h.ServeHTTP(w, r)
			return
		}
		if d.IsDir() {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<style>.upload-btn{text-decoration: none;}</style>`)
			fmt.Fprintf(w, "<a class=\"upload-btn\" href=\"/@upload.html?path=%s\">上传文件</a>\n", urlPath)
		}
		h.ServeHTTP(w, r)
	})
}

func (sp *StaticFilePlugin) Handle(conn io.ReadWriteCloser, realConn net.Conn, _ *ExtraInfo) {
	wrapConn := netpkg.WrapReadWriteCloserToConn(conn, realConn)
	_ = sp.l.PutConn(wrapConn)
}

func (sp *StaticFilePlugin) Name() string {
	return v1.PluginStaticFile
}

func (sp *StaticFilePlugin) Close() error {
	sp.s.Close()
	sp.l.Close()
	return nil
}

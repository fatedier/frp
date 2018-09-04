// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package beego

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/context"
	"github.com/astaxie/beego/logs"
)

var errNotStaticRequest = errors.New("request not a static file request")

func serverStaticRouter(ctx *context.Context) {
	if ctx.Input.Method() != "GET" && ctx.Input.Method() != "HEAD" {
		return
	}

	forbidden, filePath, fileInfo, err := lookupFile(ctx)
	if err == errNotStaticRequest {
		return
	}

	if forbidden {
		exception("403", ctx)
		return
	}

	if filePath == "" || fileInfo == nil {
		if BConfig.RunMode == DEV {
			logs.Warn("Can't find/open the file:", filePath, err)
		}
		http.NotFound(ctx.ResponseWriter, ctx.Request)
		return
	}
	if fileInfo.IsDir() {
		requestURL := ctx.Input.URL()
		if requestURL[len(requestURL)-1] != '/' {
			redirectURL := requestURL + "/"
			if ctx.Request.URL.RawQuery != "" {
				redirectURL = redirectURL + "?" + ctx.Request.URL.RawQuery
			}
			ctx.Redirect(302, redirectURL)
		} else {
			//serveFile will list dir
			http.ServeFile(ctx.ResponseWriter, ctx.Request, filePath)
		}
		return
	}

	var enableCompress = BConfig.EnableGzip && isStaticCompress(filePath)
	var acceptEncoding string
	if enableCompress {
		acceptEncoding = context.ParseEncoding(ctx.Request)
	}
	b, n, sch, err := openFile(filePath, fileInfo, acceptEncoding)
	if err != nil {
		if BConfig.RunMode == DEV {
			logs.Warn("Can't compress the file:", filePath, err)
		}
		http.NotFound(ctx.ResponseWriter, ctx.Request)
		return
	}

	if b {
		ctx.Output.Header("Content-Encoding", n)
	} else {
		ctx.Output.Header("Content-Length", strconv.FormatInt(sch.size, 10))
	}

	http.ServeContent(ctx.ResponseWriter, ctx.Request, filePath, sch.modTime, sch)
	return

}

type serveContentHolder struct {
	*bytes.Reader
	modTime  time.Time
	size     int64
	encoding string
}

var (
	staticFileMap = make(map[string]*serveContentHolder)
	mapLock       sync.RWMutex
)

func openFile(filePath string, fi os.FileInfo, acceptEncoding string) (bool, string, *serveContentHolder, error) {
	mapKey := acceptEncoding + ":" + filePath
	mapLock.RLock()
	mapFile, _ := staticFileMap[mapKey]
	mapLock.RUnlock()
	if isOk(mapFile, fi) {
		return mapFile.encoding != "", mapFile.encoding, mapFile, nil
	}
	mapLock.Lock()
	defer mapLock.Unlock()
	if mapFile, _ = staticFileMap[mapKey]; !isOk(mapFile, fi) {
		file, err := os.Open(filePath)
		if err != nil {
			return false, "", nil, err
		}
		defer file.Close()
		var bufferWriter bytes.Buffer
		_, n, err := context.WriteFile(acceptEncoding, &bufferWriter, file)
		if err != nil {
			return false, "", nil, err
		}
		mapFile = &serveContentHolder{Reader: bytes.NewReader(bufferWriter.Bytes()), modTime: fi.ModTime(), size: int64(bufferWriter.Len()), encoding: n}
		staticFileMap[mapKey] = mapFile
	}

	return mapFile.encoding != "", mapFile.encoding, mapFile, nil
}

func isOk(s *serveContentHolder, fi os.FileInfo) bool {
	if s == nil {
		return false
	}
	return s.modTime == fi.ModTime() && s.size == fi.Size()
}

// isStaticCompress detect static files
func isStaticCompress(filePath string) bool {
	for _, statExtension := range BConfig.WebConfig.StaticExtensionsToGzip {
		if strings.HasSuffix(strings.ToLower(filePath), strings.ToLower(statExtension)) {
			return true
		}
	}
	return false
}

// searchFile search the file by url path
// if none the static file prefix matches ,return notStaticRequestErr
func searchFile(ctx *context.Context) (string, os.FileInfo, error) {
	requestPath := filepath.ToSlash(filepath.Clean(ctx.Request.URL.Path))
	// special processing : favicon.ico/robots.txt  can be in any static dir
	if requestPath == "/favicon.ico" || requestPath == "/robots.txt" {
		file := path.Join(".", requestPath)
		if fi, _ := os.Stat(file); fi != nil {
			return file, fi, nil
		}
		for _, staticDir := range BConfig.WebConfig.StaticDir {
			filePath := path.Join(staticDir, requestPath)
			if fi, _ := os.Stat(filePath); fi != nil {
				return filePath, fi, nil
			}
		}
		return "", nil, errNotStaticRequest
	}

	for prefix, staticDir := range BConfig.WebConfig.StaticDir {
		if !strings.Contains(requestPath, prefix) {
			continue
		}
		if len(requestPath) > len(prefix) && requestPath[len(prefix)] != '/' {
			continue
		}
		filePath := path.Join(staticDir, requestPath[len(prefix):])
		if fi, err := os.Stat(filePath); fi != nil {
			return filePath, fi, err
		}
	}
	return "", nil, errNotStaticRequest
}

// lookupFile find the file to serve
// if the file is dir ,search the index.html as default file( MUST NOT A DIR also)
// if the index.html not exist or is a dir, give a forbidden response depending on  DirectoryIndex
func lookupFile(ctx *context.Context) (bool, string, os.FileInfo, error) {
	fp, fi, err := searchFile(ctx)
	if fp == "" || fi == nil {
		return false, "", nil, err
	}
	if !fi.IsDir() {
		return false, fp, fi, err
	}
	if requestURL := ctx.Input.URL(); requestURL[len(requestURL)-1] == '/' {
		ifp := filepath.Join(fp, "index.html")
		if ifi, _ := os.Stat(ifp); ifi != nil && ifi.Mode().IsRegular() {
			return false, ifp, ifi, err
		}
	}
	return !BConfig.WebConfig.DirectoryIndex, fp, fi, err
}

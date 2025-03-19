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
	"context"
	"io"
	"net"
	"net/http"
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
	router.Use(netpkg.NewHTTPAuthMiddleware(opts.HTTPUser, opts.HTTPPassword).SetAuthFailDelay(200 * time.Millisecond).Middleware)
	router.PathPrefix(prefix).Handler(netpkg.MakeHTTPGzipHandler(http.StripPrefix(prefix, http.FileServer(http.Dir(opts.LocalPath))))).Methods("GET")
	sp.s = &http.Server{
		Handler:           router,
		ReadHeaderTimeout: 60 * time.Second,
	}
	go func() {
		_ = sp.s.Serve(listener)
	}()
	return sp, nil
}

func (sp *StaticFilePlugin) Handle(_ context.Context, conn io.ReadWriteCloser, realConn net.Conn, _ *ExtraInfo) {
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

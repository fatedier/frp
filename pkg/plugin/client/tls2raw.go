// Copyright 2024 The frp Authors
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
	"crypto/tls"
	"io"
	"net"

	libio "github.com/fatedier/golib/io"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/transport"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
)

func init() {
	Register(v1.PluginTLS2Raw, NewTLS2RawPlugin)
}

type TLS2RawPlugin struct {
	opts *v1.TLS2RawPluginOptions

	tlsConfig *tls.Config
}

func NewTLS2RawPlugin(options v1.ClientPluginOptions) (Plugin, error) {
	opts := options.(*v1.TLS2RawPluginOptions)

	p := &TLS2RawPlugin{
		opts: opts,
	}

	tlsConfig, err := transport.NewServerTLSConfig(p.opts.CrtPath, p.opts.KeyPath, "")
	if err != nil {
		return nil, err
	}
	p.tlsConfig = tlsConfig
	return p, nil
}

func (p *TLS2RawPlugin) Handle(ctx context.Context, conn io.ReadWriteCloser, realConn net.Conn, _ *ExtraInfo) {
	xl := xlog.FromContextSafe(ctx)

	wrapConn := netpkg.WrapReadWriteCloserToConn(conn, realConn)
	tlsConn := tls.Server(wrapConn, p.tlsConfig)

	if err := tlsConn.Handshake(); err != nil {
		xl.Warnf("tls handshake error: %v", err)
		return
	}
	rawConn, err := net.Dial("tcp", p.opts.LocalAddr)
	if err != nil {
		xl.Warnf("dial to local addr error: %v", err)
		return
	}

	libio.Join(tlsConn, rawConn)
}

func (p *TLS2RawPlugin) Name() string {
	return v1.PluginTLS2Raw
}

func (p *TLS2RawPlugin) Close() error {
	return nil
}

// Copyright 2023 The frp Authors
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

package client

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	libnet "github.com/fatedier/golib/net"
	fmux "github.com/hashicorp/yamux"
	quic "github.com/quic-go/quic-go"
	"github.com/samber/lo"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/transport"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
)

// Connector is an interface for establishing connections to the server.
type Connector interface {
	Open() error
	Connect() (net.Conn, error)
	Close() error
}

// defaultConnectorImpl is the default implementation of Connector for normal frpc.
type defaultConnectorImpl struct {
	ctx context.Context
	cfg *v1.ClientCommonConfig

	muxSession *fmux.Session
	quicConn   quic.Connection
	closeOnce  sync.Once
}

func NewConnector(ctx context.Context, cfg *v1.ClientCommonConfig) Connector {
	return &defaultConnectorImpl{
		ctx: ctx,
		cfg: cfg,
	}
}

// Open opens an underlying connection to the server.
// The underlying connection is either a TCP connection or a QUIC connection.
// After the underlying connection is established, you can call Connect() to get a stream.
// If TCPMux isn't enabled, the underlying connection is nil, you will get a new real TCP connection every time you call Connect().
func (c *defaultConnectorImpl) Open() error {
	xl := xlog.FromContextSafe(c.ctx)

	// special for quic
	if strings.EqualFold(c.cfg.Transport.Protocol, "quic") {
		var tlsConfig *tls.Config
		var err error
		sn := c.cfg.Transport.TLS.ServerName
		if sn == "" {
			sn = c.cfg.ServerAddr
		}
		if lo.FromPtr(c.cfg.Transport.TLS.Enable) {
			tlsConfig, err = transport.NewClientTLSConfig(
				c.cfg.Transport.TLS.CertFile,
				c.cfg.Transport.TLS.KeyFile,
				c.cfg.Transport.TLS.TrustedCaFile,
				sn)
		} else {
			tlsConfig, err = transport.NewClientTLSConfig("", "", "", sn)
		}
		if err != nil {
			xl.Warnf("fail to build tls configuration, err: %v", err)
			return err
		}
		tlsConfig.NextProtos = []string{"frp"}

		conn, err := quic.DialAddr(
			c.ctx,
			net.JoinHostPort(c.cfg.ServerAddr, strconv.Itoa(c.cfg.ServerPort)),
			tlsConfig, &quic.Config{
				MaxIdleTimeout:     time.Duration(c.cfg.Transport.QUIC.MaxIdleTimeout) * time.Second,
				MaxIncomingStreams: int64(c.cfg.Transport.QUIC.MaxIncomingStreams),
				KeepAlivePeriod:    time.Duration(c.cfg.Transport.QUIC.KeepalivePeriod) * time.Second,
			})
		if err != nil {
			return err
		}
		c.quicConn = conn
		return nil
	}

	if !lo.FromPtr(c.cfg.Transport.TCPMux) {
		return nil
	}

	conn, err := c.realConnect()
	if err != nil {
		return err
	}

	fmuxCfg := fmux.DefaultConfig()
	fmuxCfg.KeepAliveInterval = time.Duration(c.cfg.Transport.TCPMuxKeepaliveInterval) * time.Second
	fmuxCfg.LogOutput = io.Discard
	fmuxCfg.MaxStreamWindowSize = 6 * 1024 * 1024
	session, err := fmux.Client(conn, fmuxCfg)
	if err != nil {
		return err
	}
	c.muxSession = session
	return nil
}

// Connect returns a stream from the underlying connection, or a new TCP connection if TCPMux isn't enabled.
func (c *defaultConnectorImpl) Connect() (net.Conn, error) {
	if c.quicConn != nil {
		stream, err := c.quicConn.OpenStreamSync(context.Background())
		if err != nil {
			return nil, err
		}
		return netpkg.QuicStreamToNetConn(stream, c.quicConn), nil
	} else if c.muxSession != nil {
		stream, err := c.muxSession.OpenStream()
		if err != nil {
			return nil, err
		}
		return stream, nil
	}

	return c.realConnect()
}

func (c *defaultConnectorImpl) realConnect() (net.Conn, error) {
	xl := xlog.FromContextSafe(c.ctx)
	var tlsConfig *tls.Config
	var err error
	tlsEnable := lo.FromPtr(c.cfg.Transport.TLS.Enable)
	if c.cfg.Transport.Protocol == "wss" {
		tlsEnable = true
	}
	if tlsEnable {
		sn := c.cfg.Transport.TLS.ServerName
		if sn == "" {
			sn = c.cfg.ServerAddr
		}

		tlsConfig, err = transport.NewClientTLSConfig(
			c.cfg.Transport.TLS.CertFile,
			c.cfg.Transport.TLS.KeyFile,
			c.cfg.Transport.TLS.TrustedCaFile,
			sn)
		if err != nil {
			xl.Warnf("fail to build tls configuration, err: %v", err)
			return nil, err
		}
	}

	proxyType, addr, auth, err := libnet.ParseProxyURL(c.cfg.Transport.ProxyURL)
	if err != nil {
		xl.Errorf("fail to parse proxy url")
		return nil, err
	}
	dialOptions := []libnet.DialOption{}
	protocol := c.cfg.Transport.Protocol
	switch protocol {
	case "websocket":
		protocol = "tcp"
		dialOptions = append(dialOptions, libnet.WithAfterHook(libnet.AfterHook{Hook: netpkg.DialHookWebsocket(protocol, "")}))
		dialOptions = append(dialOptions, libnet.WithAfterHook(libnet.AfterHook{
			Hook: netpkg.DialHookCustomTLSHeadByte(tlsConfig != nil, lo.FromPtr(c.cfg.Transport.TLS.DisableCustomTLSFirstByte)),
		}))
		dialOptions = append(dialOptions, libnet.WithTLSConfig(tlsConfig))
	case "wss":
		protocol = "tcp"
		dialOptions = append(dialOptions, libnet.WithTLSConfigAndPriority(100, tlsConfig))
		// Make sure that if it is wss, the websocket hook is executed after the tls hook.
		dialOptions = append(dialOptions, libnet.WithAfterHook(libnet.AfterHook{Hook: netpkg.DialHookWebsocket(protocol, tlsConfig.ServerName), Priority: 110}))
	default:
		dialOptions = append(dialOptions, libnet.WithAfterHook(libnet.AfterHook{
			Hook: netpkg.DialHookCustomTLSHeadByte(tlsConfig != nil, lo.FromPtr(c.cfg.Transport.TLS.DisableCustomTLSFirstByte)),
		}))
		dialOptions = append(dialOptions, libnet.WithTLSConfig(tlsConfig))
	}

	if c.cfg.Transport.ConnectServerLocalIP != "" {
		dialOptions = append(dialOptions, libnet.WithLocalAddr(c.cfg.Transport.ConnectServerLocalIP))
	}
	dialOptions = append(dialOptions,
		libnet.WithProtocol(protocol),
		libnet.WithTimeout(time.Duration(c.cfg.Transport.DialServerTimeout)*time.Second),
		libnet.WithKeepAlive(time.Duration(c.cfg.Transport.DialServerKeepAlive)*time.Second),
		libnet.WithProxy(proxyType, addr),
		libnet.WithProxyAuth(auth),
	)
	conn, err := libnet.DialContext(
		c.ctx,
		net.JoinHostPort(c.cfg.ServerAddr, strconv.Itoa(c.cfg.ServerPort)),
		dialOptions...,
	)
	return conn, err
}

func (c *defaultConnectorImpl) Close() error {
	c.closeOnce.Do(func() {
		if c.quicConn != nil {
			_ = c.quicConn.CloseWithError(0, "")
		}
		if c.muxSession != nil {
			_ = c.muxSession.Close()
		}
	})
	return nil
}

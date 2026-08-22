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

package transport

import (
	"context"
	"crypto/tls"
	"net"

	libnet "github.com/fatedier/golib/net"
	utls "github.com/refraction-networking/utls"
)

// toUTLSConfig copies the fields we use from a standard *tls.Config into a *utls.Config.
func toUTLSConfig(cfg *tls.Config) *utls.Config {
	return &utls.Config{
		ServerName:         cfg.ServerName,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		RootCAs:            cfg.RootCAs,
		NextProtos:         cfg.NextProtos,
	}
}

// DialHookUTLS returns an AfterHook that upgrades the raw connection to TLS using a
// uTLS ClientHello fingerprint that mimics a real Chrome browser, instead of Go's
// standard library crypto/tls handshake. This defeats passive JA3/JA4-style TLS
// fingerprinting that would otherwise flag the connection as Go-generated traffic.
//
// If tlsConfig is nil, the connection is returned unmodified (TLS disabled).
func DialHookUTLS(tlsConfig *tls.Config) libnet.AfterHookFunc {
	return func(ctx context.Context, c net.Conn, _ string) (context.Context, net.Conn, error) {
		if tlsConfig == nil {
			return ctx, c, nil
		}
		uConn := utls.UClient(c, toUTLSConfig(tlsConfig), utls.HelloChrome_Auto)
		return ctx, uConn, nil
	}
}

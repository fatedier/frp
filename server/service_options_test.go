// Copyright 2026 The frp Authors
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

package server

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestNewServiceWithOptionsRejectsListenerWithoutVhostHTTPSPort(t *testing.T) {
	cfg := &v1.ServerConfig{BindAddr: "127.0.0.1", ProxyBindAddr: "127.0.0.1"}
	require.NoError(t, cfg.Complete())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	_, err = NewServiceWithOptions(cfg, ServiceOptions{VhostHTTPSListener: ln})
	require.ErrorContains(t, err, "requires vhostHTTPSPort")
}

// TestNewServiceWithOptionsUsesInjectedListener checks the two properties an
// injected listener has to provide: frp does not bind VhostHTTPSPort itself,
// and the Service takes ownership of the listener and closes it on shutdown.
//
// A caller that already accepted the connection (for example an SNI router
// sharing the public port with other services) needs both. Binding the port
// would conflict with that router; leaking the listener would keep the port
// alive after the Service stops.
func TestNewServiceWithOptionsUsesInjectedListener(t *testing.T) {
	cfg := &v1.ServerConfig{BindAddr: "127.0.0.1", ProxyBindAddr: "127.0.0.1"}
	require.NoError(t, cfg.Complete())

	// Reserve a port so frp would be able to bind it if it tried, then release
	// it: any successful bind afterwards means frp bound it itself.
	reservation, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	cfg.VhostHTTPSPort = reservation.Addr().(*net.TCPAddr).Port
	require.NoError(t, reservation.Close())

	cfg.BindPort = freeServicePort(t)
	injected, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	svr, err := NewServiceWithOptions(cfg, ServiceOptions{VhostHTTPSListener: injected})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svr.Close() })

	// frp must not bind the advertised port when a listener is supplied.
	probe, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.VhostHTTPSPort)))
	require.NoError(t, err, "frp bound vhostHTTPSPort despite an injected listener")
	require.NoError(t, probe.Close())

	// The listener is owned by the Service, so Close must release it.
	require.NoError(t, svr.Close())
	_, err = injected.Accept()
	require.ErrorIs(t, err, net.ErrClosed, "Service.Close did not close the injected listener")
}

func freeServicePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	return port
}

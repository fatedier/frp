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

//go:build !frps

package proxy

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/xlog"
)

func TestUDPProxyInWorkConnGenerationChannels(t *testing.T) {
	cfg := &v1.UDPProxyConfig{ProxyBaseConfig: v1.ProxyBaseConfig{
		Name: "udp", Type: "udp",
		ProxyBackend: v1.ProxyBackend{LocalIP: "127.0.0.1", LocalPort: 9},
	}}
	pxy := NewUDPProxy(&BaseProxy{
		baseCfg: &cfg.ProxyBaseConfig, clientCfg: &v1.ClientCommonConfig{UDPPacketSize: 1500},
		xl: xlog.New(), ctx: context.Background(),
	}, cfg).(*UDPProxy)
	require.NoError(t, pxy.Run())
	defer pxy.Close()

	conn1, peer1 := net.Pipe()
	defer conn1.Close()
	defer peer1.Close()
	conn2, peer2 := net.Pipe()
	defer conn2.Close()
	defer peer2.Close()

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	inWorkConn := func(conn net.Conn) {
		defer wg.Done()
		<-start
		pxy.InWorkConn(conn, &msg.StartWorkConn{})
	}
	go inWorkConn(conn1)
	go inWorkConn(conn2)
	close(start)
	wg.Wait()
}

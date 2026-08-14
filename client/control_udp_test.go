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

package client

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clientproxy "github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/wire"
)

func TestControlPropagatesBinaryUDPPacketCodecToWorkConn(t *testing.T) {
	echoConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	require.NoError(t, err)
	t.Cleanup(func() { _ = echoConn.Close() })

	echoDone := make(chan error, 1)
	go func() {
		buf := make([]byte, 64)
		n, addr, err := echoConn.ReadFromUDP(buf)
		if err == nil {
			_, err = echoConn.WriteToUDP(buf[:n], addr)
		}
		echoDone <- err
	}()

	authRuntime, err := auth.BuildClientAuth(&v1.AuthClientConfig{
		Method: v1.AuthMethodToken,
		Token:  "token",
	})
	require.NoError(t, err)

	controlConn, controlPeer := net.Pipe()
	t.Cleanup(func() {
		_ = controlConn.Close()
		_ = controlPeer.Close()
	})
	common := &v1.ClientCommonConfig{
		Transport:     v1.ClientTransportConfig{WireProtocol: wire.ProtocolV2},
		UDPPacketSize: 1500,
	}
	ctl, err := NewControl(context.Background(), &SessionContext{
		Common:         common,
		RunID:          "binary-udp-test",
		Conn:           msg.NewConn(controlConn, msg.NewV2ReadWriter(controlConn)),
		Auth:           authRuntime,
		UDPPacketCodec: wire.UDPPacketCodecBinary,
	})
	require.NoError(t, err)
	t.Cleanup(ctl.pm.Close)

	echoAddr := echoConn.LocalAddr().(*net.UDPAddr)
	proxyCfg := &v1.UDPProxyConfig{
		ProxyBaseConfig: v1.ProxyBaseConfig{
			Name: "udp",
			Type: string(v1.ProxyTypeUDP),
			ProxyBackend: v1.ProxyBackend{
				LocalIP:   "127.0.0.1",
				LocalPort: echoAddr.Port,
			},
		},
	}
	ctl.pm.UpdateAll([]v1.ProxyConfigurer{proxyCfg})
	require.Eventually(t, func() bool {
		status, ok := ctl.pm.GetProxyStatus("udp")
		return ok && status.Phase == clientproxy.ProxyPhaseWaitStart
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, ctl.pm.StartProxy("udp", "", ""))

	workClient, workServer := net.Pipe()
	t.Cleanup(func() {
		_ = workClient.Close()
		_ = workServer.Close()
	})
	deadline := time.Now().Add(3 * time.Second)
	require.NoError(t, workClient.SetDeadline(deadline))
	require.NoError(t, workServer.SetDeadline(deadline))
	ctl.pm.HandleWorkConn("udp", workClient, &msg.StartWorkConn{ProxyName: "udp"})

	serverRW, err := msg.NewUDPPacketReadWriter(workServer, wire.ProtocolV2, wire.UDPPacketCodecBinary)
	require.NoError(t, err)
	writeDone := make(chan error, 1)
	in := &msg.UDPPacket{
		Content:    []byte("binary udp"),
		RemoteAddr: &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 12345},
	}
	go func() {
		writeDone <- serverRW.WriteMsg(in)
	}()

	frame, err := wire.NewConn(workServer).ReadFrame()
	require.NoError(t, err)
	require.Equal(t, wire.FrameTypeMessage, frame.Type)
	require.GreaterOrEqual(t, len(frame.Payload), 2)
	require.Equal(t, msg.V2TypeUDPPacketBinary, binary.BigEndian.Uint16(frame.Payload[:2]))
	out, err := msg.DecodeUDPPacketBinary(frame.Payload[2:])
	require.NoError(t, err)
	require.Equal(t, in.Content, out.Content)
	require.Equal(t, in.RemoteAddr.String(), out.RemoteAddr.String())
	require.NoError(t, <-writeDone)
	require.NoError(t, <-echoDone)
}

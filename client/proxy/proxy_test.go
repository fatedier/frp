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
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/xlog"
)

func TestHandleTCPWorkConnectionRejectsInvalidAddress(t *testing.T) {
	workConn, peerConn := net.Pipe()
	defer peerConn.Close()

	pxy := &BaseProxy{
		baseCfg: &v1.ProxyBaseConfig{},
		xl:      xlog.New(),
	}
	pxy.HandleTCPWorkConnection(workConn, &msg.StartWorkConn{
		SrcAddr: "[",
		SrcPort: 1,
	}, nil)

	buffer := make([]byte, 1)
	_, err := peerConn.Read(buffer)
	require.ErrorIs(t, err, io.EOF)
}

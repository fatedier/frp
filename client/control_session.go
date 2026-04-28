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

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/wire"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/frp/pkg/vnet"
)

type controlSessionDialer struct {
	ctx context.Context

	common         *v1.ClientCommonConfig
	auth           *auth.ClientAuth
	clientSpec     *msg.ClientSpec
	vnetController *vnet.Controller

	connectorCreator func(context.Context, *v1.ClientCommonConfig) Connector
	autoTransport    *autoTransportManager
	autoReason       string
	autoSelection    *autoTransportSelection
}

func (d *controlSessionDialer) Dial(previousRunID string) (*SessionContext, error) {
	common := d.common
	d.autoSelection = nil
	if d.autoTransport != nil {
		selection, err := d.autoTransport.selectTransport(d.ctx, d.autoReason)
		if err != nil {
			return nil, err
		}
		d.autoSelection = selection
		common = selection.Cfg
	}

	connector := d.connectorCreator(d.ctx, common)
	if err := connector.Open(); err != nil {
		return nil, err
	}

	success := false
	defer func() {
		if !success {
			_ = connector.Close()
		}
	}()

	conn, err := connector.Connect()
	if err != nil {
		return nil, err
	}
	defer func() {
		if !success {
			_ = conn.Close()
		}
	}()

	loginMsg, err := d.buildLoginMsg(common, previousRunID)
	if err != nil {
		return nil, err
	}

	loginRespMsg, err := d.exchangeLogin(conn, common, loginMsg, d.autoSelection)
	if err != nil {
		return nil, err
	}
	if loginRespMsg.Error != "" {
		return nil, errors.New(loginRespMsg.Error)
	}

	var controlRW io.ReadWriter = conn
	if d.clientSpec == nil || d.clientSpec.Type != "ssh-tunnel" {
		controlRW, err = netpkg.NewCryptoReadWriter(conn, d.auth.EncryptionKey())
		if err != nil {
			return nil, fmt.Errorf("create control crypto read writer: %w", err)
		}
	}

	success = true
	return &SessionContext{
		Common:         common,
		RunID:          loginRespMsg.RunID,
		Conn:           msg.NewConn(conn, msg.NewReadWriter(controlRW, common.Transport.WireProtocol)),
		Auth:           d.auth,
		Connector:      newMessageConnector(connector, common.Transport.WireProtocol),
		VnetController: d.vnetController,
		AutoTransport:  d.autoTransport,
	}, nil
}

func (d *controlSessionDialer) buildLoginMsg(common *v1.ClientCommonConfig, previousRunID string) (*msg.Login, error) {
	hostname, _ := os.Hostname()
	loginMsg := &msg.Login{
		Arch:      runtime.GOARCH,
		Os:        runtime.GOOS,
		Hostname:  hostname,
		PoolCount: common.Transport.PoolCount,
		User:      common.User,
		ClientID:  common.ClientID,
		Version:   version.Full(),
		Timestamp: time.Now().Unix(),
		RunID:     previousRunID,
		Metas:     common.Metadatas,
	}
	if d.clientSpec != nil {
		loginMsg.ClientSpec = *d.clientSpec
	}

	if err := d.auth.Setter.SetLogin(loginMsg); err != nil {
		return nil, err
	}
	return loginMsg, nil
}

func (d *controlSessionDialer) exchangeLogin(
	conn net.Conn,
	common *v1.ClientCommonConfig,
	loginMsg *msg.Login,
	autoSelection *autoTransportSelection,
) (*msg.LoginResp, error) {
	rw := msg.NewV1ReadWriter(conn)
	var wireConn *wire.Conn

	if common.Transport.WireProtocol == wire.ProtocolV2 {
		if err := wire.WriteMagic(conn); err != nil {
			return nil, err
		}

		wireConn = wire.NewConn(conn)
		rw = msg.NewV2ReadWriterWithConn(wireConn)
		hello := wire.DefaultClientHello(wire.BootstrapInfo{
			Transport: common.Transport.Protocol,
			TLS:       lo.FromPtr(common.Transport.TLS.Enable) || common.Transport.Protocol == "wss" || common.Transport.Protocol == "quic",
			TCPMux:    lo.FromPtr(common.Transport.TCPMux),
		})
		if err := wireConn.WriteJSONFrame(wire.FrameTypeClientHello, hello); err != nil {
			return nil, err
		}
	}
	if autoSelection != nil && autoSelection.SendSelect {
		if err := rw.WriteMsg(&msg.SelectTransport{
			Protocol:          autoSelection.Candidate.Protocol,
			Addr:              autoSelection.Candidate.advertisedAddr(),
			Port:              autoSelection.Candidate.Port,
			Reason:            autoSelection.Reason,
			Scores:            autoSelection.Scores,
			ClientAutoVersion: msg.AutoTransportVersion,
		}); err != nil {
			return nil, err
		}
	}
	if err := rw.WriteMsg(loginMsg); err != nil {
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer func() {
		_ = conn.SetReadDeadline(time.Time{})
	}()

	if wireConn != nil {
		var serverHello wire.ServerHello
		if err := wireConn.ReadJSONFrame(wire.FrameTypeServerHello, &serverHello); err != nil {
			return nil, err
		}
		if serverHello.Error != "" {
			return nil, errors.New(serverHello.Error)
		}
	}

	var loginRespMsg msg.LoginResp
	if err := rw.ReadMsgInto(&loginRespMsg); err != nil {
		return nil, err
	}
	return &loginRespMsg, nil
}

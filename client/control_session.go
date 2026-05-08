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
}

func (d *controlSessionDialer) Dial(previousRunID string) (*SessionContext, error) {
	connector := d.connectorCreator(d.ctx, d.common)
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

	loginMsg, err := d.buildLoginMsg(previousRunID)
	if err != nil {
		return nil, err
	}

	loginResult, err := d.exchangeLogin(conn, loginMsg)
	if err != nil {
		return nil, err
	}
	loginRespMsg := loginResult.resp
	if loginRespMsg.Error != "" {
		return nil, errors.New(loginRespMsg.Error)
	}

	var controlRW io.ReadWriter = conn
	if d.clientSpec == nil || d.clientSpec.Type != "ssh-tunnel" {
		controlRW, err = d.newControlReadWriter(conn, loginResult.crypto)
		if err != nil {
			return nil, fmt.Errorf("create control crypto read writer: %w", err)
		}
	}

	success = true
	return &SessionContext{
		Common:         d.common,
		RunID:          loginRespMsg.RunID,
		Conn:           msg.NewConn(conn, msg.NewReadWriter(controlRW, d.common.Transport.WireProtocol)),
		Auth:           d.auth,
		Connector:      newMessageConnector(connector, d.common.Transport.WireProtocol),
		VnetController: d.vnetController,
	}, nil
}

func (d *controlSessionDialer) buildLoginMsg(previousRunID string) (*msg.Login, error) {
	hostname, _ := os.Hostname()
	loginMsg := &msg.Login{
		Arch:      runtime.GOARCH,
		Os:        runtime.GOOS,
		Hostname:  hostname,
		PoolCount: d.common.Transport.PoolCount,
		User:      d.common.User,
		ClientID:  d.common.ClientID,
		Version:   version.Full(),
		Timestamp: time.Now().Unix(),
		RunID:     previousRunID,
		Metas:     d.common.Metadatas,
	}
	if d.clientSpec != nil {
		loginMsg.ClientSpec = *d.clientSpec
	}

	if err := d.auth.Setter.SetLogin(loginMsg); err != nil {
		return nil, err
	}
	return loginMsg, nil
}

type loginExchangeResult struct {
	resp   *msg.LoginResp
	crypto *wire.CryptoContext
}

func (d *controlSessionDialer) exchangeLogin(conn net.Conn, loginMsg *msg.Login) (*loginExchangeResult, error) {
	rw := msg.NewV1ReadWriter(conn)
	var wireConn *wire.Conn
	var clientHello wire.ClientHello
	var clientHelloPayload []byte

	if d.common.Transport.WireProtocol == wire.ProtocolV2 {
		if err := wire.WriteMagic(conn); err != nil {
			return nil, err
		}

		wireConn = wire.NewConn(conn)
		rw = msg.NewV2ReadWriterWithConn(wireConn)
		var err error
		clientHello, err = wire.NewClientHello(wire.BootstrapInfo{
			Transport: d.common.Transport.Protocol,
			TLS:       lo.FromPtr(d.common.Transport.TLS.Enable) || d.common.Transport.Protocol == "wss" || d.common.Transport.Protocol == "quic",
			TCPMux:    lo.FromPtr(d.common.Transport.TCPMux),
		})
		if err != nil {
			return nil, err
		}
		clientHelloFrame, err := wire.NewJSONFrame(wire.FrameTypeClientHello, clientHello)
		if err != nil {
			return nil, err
		}
		if err := wireConn.WriteFrame(clientHelloFrame); err != nil {
			return nil, err
		}
		clientHelloPayload = clientHelloFrame.Payload
	}
	if err := rw.WriteMsg(loginMsg); err != nil {
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer func() {
		_ = conn.SetReadDeadline(time.Time{})
	}()

	var cryptoContext *wire.CryptoContext
	if wireConn != nil {
		serverHelloFrame, err := wireConn.ReadFrame()
		if err != nil {
			return nil, err
		}
		if serverHelloFrame.Type != wire.FrameTypeServerHello {
			return nil, fmt.Errorf("unexpected frame type %d, want %d", serverHelloFrame.Type, wire.FrameTypeServerHello)
		}
		var serverHello wire.ServerHello
		if err := wireConn.UnmarshalFrame(serverHelloFrame, &serverHello); err != nil {
			return nil, err
		}
		if serverHello.Error != "" {
			return nil, errors.New(serverHello.Error)
		}
		cryptoContext, err = wire.NewClientCryptoContext(clientHelloPayload, serverHelloFrame.Payload)
		if err != nil {
			return nil, err
		}
	}

	var loginRespMsg msg.LoginResp
	if err := rw.ReadMsgInto(&loginRespMsg); err != nil {
		return nil, err
	}
	return &loginExchangeResult{
		resp:   &loginRespMsg,
		crypto: cryptoContext,
	}, nil
}

func (d *controlSessionDialer) newControlReadWriter(conn net.Conn, cryptoContext *wire.CryptoContext) (io.ReadWriter, error) {
	if d.common.Transport.WireProtocol == wire.ProtocolV2 {
		if cryptoContext == nil {
			return nil, errors.New("missing v2 crypto negotiation")
		}
		return netpkg.NewAEADCryptoReadWriter(
			conn,
			d.auth.EncryptionKey(),
			netpkg.AEADCryptoRoleClient,
			cryptoContext.Algorithm,
			cryptoContext.TranscriptHash,
		)
	}
	return netpkg.NewCryptoReadWriter(conn, d.auth.EncryptionKey())
}

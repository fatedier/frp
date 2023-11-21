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

package ssh

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"

	libio "github.com/fatedier/golib/io"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	utilnet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/pkg/virtual"
)

const (
	// https://datatracker.ietf.org/doc/html/rfc4254#page-16
	ChannelTypeServerOpenChannel = "forwarded-tcpip"
	RequestTypeForward           = "tcpip-forward"
)

type tcpipForward struct {
	Host string
	Port uint32
}

// https://datatracker.ietf.org/doc/html/rfc4254#page-16
type forwardedTCPPayload struct {
	Addr string
	Port uint32

	// can be default empty value but do not delete it
	// because ssh protocol shoule be reserved
	OriginAddr string
	OriginPort uint32
}

type TunnelServer struct {
	underlyingConn net.Conn
	sshConn        *ssh.ServerConn
	sc             *ssh.ServerConfig

	vc                 *virtual.Client
	serverPeerListener *utilnet.InternalListener
	doneCh             chan struct{}
}

func NewTunnelServer(conn net.Conn, sc *ssh.ServerConfig, serverPeerListener *utilnet.InternalListener) (*TunnelServer, error) {
	s := &TunnelServer{
		underlyingConn:     conn,
		sc:                 sc,
		serverPeerListener: serverPeerListener,
		doneCh:             make(chan struct{}),
	}
	return s, nil
}

func (s *TunnelServer) Run() error {
	sshConn, channels, requests, err := ssh.NewServerConn(s.underlyingConn, s.sc)
	if err != nil {
		return err
	}
	s.sshConn = sshConn

	addr, extraPayload, err := s.waitForwardAddrAndExtraPayload(channels, requests, 3*time.Second)
	if err != nil {
		return err
	}

	clientCfg, pc, err := s.parseClientAndProxyConfigurer(addr, extraPayload)
	if err != nil {
		return err
	}
	clientCfg.User = util.EmptyOr(sshConn.Permissions.Extensions["user"], clientCfg.User)
	pc.Complete(clientCfg.User)

	s.vc = virtual.NewClient(clientCfg)
	// join workConn and ssh channel
	s.vc.SetInWorkConnCallback(func(base *v1.ProxyBaseConfig, workConn net.Conn, m *msg.StartWorkConn) bool {
		c, err := s.openConn(addr)
		if err != nil {
			return false
		}
		libio.Join(c, workConn)
		return false
	})
	// transfer connection from virtual client to server peer listener
	go func() {
		l := s.vc.PeerListener()
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			_ = s.serverPeerListener.PutConn(conn)
		}
	}()
	xl := xlog.New().AddPrefix(xlog.LogPrefix{Name: "sshVirtualClient", Value: "sshVirtualClient", Priority: 100})
	ctx := xlog.NewContext(context.Background(), xl)
	go func() {
		_ = s.vc.Run(ctx)
	}()

	s.vc.UpdateProxyConfigurer([]v1.ProxyConfigurer{pc})

	_ = sshConn.Wait()
	_ = sshConn.Close()
	s.vc.Close()
	close(s.doneCh)
	return nil
}

func (s *TunnelServer) waitForwardAddrAndExtraPayload(
	channels <-chan ssh.NewChannel,
	requests <-chan *ssh.Request,
	timeout time.Duration,
) (*tcpipForward, string, error) {
	addrCh := make(chan *tcpipForward, 1)
	extraPayloadCh := make(chan string, 1)

	// get forward address
	go func() {
		addrGot := false
		for req := range requests {
			switch req.Type {
			case RequestTypeForward:
				if !addrGot {
					payload := tcpipForward{}
					if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
						return
					}
					addrGot = true
					addrCh <- &payload
				}
			default:
				if req.WantReply {
					_ = req.Reply(true, nil)
				}
			}
		}
	}()

	// get extra payload
	go func() {
		for newChannel := range channels {
			// extraPayload will send to extraPayloadCh
			go s.handleNewChannel(newChannel, extraPayloadCh)
		}
	}()

	var (
		addr         *tcpipForward
		extraPayload string
	)

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case v := <-addrCh:
			addr = v
		case extra := <-extraPayloadCh:
			extraPayload = extra
		case <-timer.C:
			return nil, "", fmt.Errorf("get addr and extra payload timeout")
		}
		if addr != nil && extraPayload != "" {
			break
		}
	}
	return addr, extraPayload, nil
}

func (s *TunnelServer) parseClientAndProxyConfigurer(_ *tcpipForward, extraPayload string) (*v1.ClientCommonConfig, v1.ProxyConfigurer, error) {
	cmd := &cobra.Command{}
	args := strings.Split(extraPayload, " ")
	if len(args) < 1 {
		return nil, nil, fmt.Errorf("invalid extra payload")
	}
	proxyType := strings.TrimSpace(args[0])
	supportTypes := []string{"tcp", "http", "https", "tcpmux", "stcp"}
	if !lo.Contains(supportTypes, proxyType) {
		return nil, nil, fmt.Errorf("invalid proxy type: %s, support types: %v", proxyType, supportTypes)
	}
	pc := v1.NewProxyConfigurerByType(v1.ProxyType(proxyType))
	if pc == nil {
		return nil, nil, fmt.Errorf("new proxy configurer error")
	}
	config.RegisterProxyFlags(cmd, pc)

	clientCfg := v1.ClientCommonConfig{}
	config.RegisterClientCommonConfigFlags(cmd, &clientCfg)

	if err := cmd.ParseFlags(args); err != nil {
		return nil, nil, fmt.Errorf("parse flags from ssh client error: %v", err)
	}
	return &clientCfg, pc, nil
}

func (s *TunnelServer) handleNewChannel(channel ssh.NewChannel, extraPayloadCh chan string) {
	ch, reqs, err := channel.Accept()
	if err != nil {
		return
	}
	go s.keepAlive(ch)

	for req := range reqs {
		if req.Type != "exec" {
			continue
		}
		if len(req.Payload) <= 4 {
			continue
		}
		end := 4 + binary.BigEndian.Uint32(req.Payload[:4])
		if len(req.Payload) < int(end) {
			continue
		}
		extraPayload := string(req.Payload[4:end])
		select {
		case extraPayloadCh <- extraPayload:
		default:
		}
	}
}

func (s *TunnelServer) keepAlive(ch ssh.Channel) {
	tk := time.NewTicker(time.Second * 30)
	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			_, err := ch.SendRequest("heartbeat", false, nil)
			if err != nil {
				return
			}
		case <-s.doneCh:
			return
		}
	}
}

func (s *TunnelServer) openConn(addr *tcpipForward) (net.Conn, error) {
	payload := forwardedTCPPayload{
		Addr: addr.Host,
		Port: addr.Port,
	}
	channel, reqs, err := s.sshConn.OpenChannel(ChannelTypeServerOpenChannel, ssh.Marshal(&payload))
	if err != nil {
		return nil, fmt.Errorf("open ssh channel error: %v", err)
	}
	go ssh.DiscardRequests(reqs)

	conn := utilnet.WrapReadWriteCloserToConn(channel, s.underlyingConn)
	return conn, nil
}

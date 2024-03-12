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
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	libio "github.com/fatedier/golib/io"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/ssh"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/log"
	netpkg "github.com/fatedier/frp/pkg/util/net"
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

	OriginAddr string
	OriginPort uint32
}

type TunnelServer struct {
	underlyingConn net.Conn
	sshConn        *ssh.ServerConn
	sc             *ssh.ServerConfig
	firstChannel   ssh.Channel

	vc                 *virtual.Client
	peerServerListener *netpkg.InternalListener
	doneCh             chan struct{}
	closeDoneChOnce    sync.Once
}

func NewTunnelServer(conn net.Conn, sc *ssh.ServerConfig, peerServerListener *netpkg.InternalListener) (*TunnelServer, error) {
	s := &TunnelServer{
		underlyingConn:     conn,
		sc:                 sc,
		peerServerListener: peerServerListener,
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

	clientCfg, pc, helpMessage, err := s.parseClientAndProxyConfigurer(addr, extraPayload)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			s.writeToClient(helpMessage)
			return nil
		}
		s.writeToClient(err.Error())
		return fmt.Errorf("parse flags from ssh client error: %v", err)
	}
	clientCfg.Complete()
	if sshConn.Permissions != nil {
		clientCfg.User = util.EmptyOr(sshConn.Permissions.Extensions["user"], clientCfg.User)
	}
	pc.Complete(clientCfg.User)

	vc, err := virtual.NewClient(virtual.ClientOptions{
		Common: clientCfg,
		Spec: &msg.ClientSpec{
			Type: "ssh-tunnel",
			// If ssh does not require authentication, then the virtual client needs to authenticate through a token.
			// Otherwise, once ssh authentication is passed, the virtual client does not need to authenticate again.
			AlwaysAuthPass: !s.sc.NoClientAuth,
		},
		HandleWorkConnCb: func(base *v1.ProxyBaseConfig, workConn net.Conn, m *msg.StartWorkConn) bool {
			// join workConn and ssh channel
			c, err := s.openConn(addr)
			if err != nil {
				log.Tracef("open conn error: %v", err)
				workConn.Close()
				return false
			}
			libio.Join(c, workConn)
			return false
		},
	})
	if err != nil {
		return err
	}
	s.vc = vc

	// transfer connection from virtual client to server peer listener
	go func() {
		l := s.vc.PeerListener()
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			_ = s.peerServerListener.PutConn(conn)
		}
	}()
	xl := xlog.New().AddPrefix(xlog.LogPrefix{Name: "sshVirtualClient", Value: "sshVirtualClient", Priority: 100})
	ctx := xlog.NewContext(context.Background(), xl)
	go func() {
		vcErr := s.vc.Run(ctx)
		if vcErr != nil {
			s.writeToClient(vcErr.Error())
		}

		// If vc.Run returns, it means that the virtual client has been closed, and the ssh tunnel connection should be closed.
		// One scenario is that the virtual client exits due to login failure.
		s.closeDoneChOnce.Do(func() {
			_ = sshConn.Close()
			close(s.doneCh)
		})
	}()

	s.vc.UpdateProxyConfigurer([]v1.ProxyConfigurer{pc})

	if ps, err := s.waitProxyStatusReady(pc.GetBaseConfig().Name, time.Second); err != nil {
		s.writeToClient(err.Error())
		log.Warnf("wait proxy status ready error: %v", err)
	} else {
		// success
		s.writeToClient(createSuccessInfo(clientCfg.User, pc, ps))
		_ = sshConn.Wait()
	}

	s.vc.Close()
	log.Tracef("ssh tunnel connection from %v closed", sshConn.RemoteAddr())
	s.closeDoneChOnce.Do(func() {
		_ = sshConn.Close()
		close(s.doneCh)
	})
	return nil
}

func (s *TunnelServer) writeToClient(data string) {
	if s.firstChannel == nil {
		return
	}
	_, _ = s.firstChannel.Write([]byte(data + "\n"))
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
			if req.Type == RequestTypeForward && !addrGot {
				payload := tcpipForward{}
				if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
					return
				}
				addrGot = true
				addrCh <- &payload
			}
			if req.WantReply {
				_ = req.Reply(true, nil)
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

func (s *TunnelServer) parseClientAndProxyConfigurer(_ *tcpipForward, extraPayload string) (*v1.ClientCommonConfig, v1.ProxyConfigurer, string, error) {
	helpMessage := ""
	cmd := &cobra.Command{
		Use:   "ssh v0@{address} [command]",
		Short: "ssh v0@{address} [command]",
		Run:   func(*cobra.Command, []string) {},
	}
	cmd.SetGlobalNormalizationFunc(config.WordSepNormalizeFunc)

	args := strings.Split(extraPayload, " ")
	if len(args) < 1 {
		return nil, nil, helpMessage, fmt.Errorf("invalid extra payload")
	}
	proxyType := strings.TrimSpace(args[0])
	supportTypes := []string{"tcp", "http", "https", "tcpmux", "stcp"}
	if !slices.Contains(supportTypes, proxyType) {
		return nil, nil, helpMessage, fmt.Errorf("invalid proxy type: %s, support types: %v", proxyType, supportTypes)
	}
	pc := v1.NewProxyConfigurerByType(v1.ProxyType(proxyType))
	if pc == nil {
		return nil, nil, helpMessage, fmt.Errorf("new proxy configurer error")
	}
	config.RegisterProxyFlags(cmd, pc, config.WithSSHMode())

	clientCfg := v1.ClientCommonConfig{}
	config.RegisterClientCommonConfigFlags(cmd, &clientCfg, config.WithSSHMode())

	cmd.InitDefaultHelpCmd()
	if err := cmd.ParseFlags(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			helpMessage = cmd.UsageString()
		}
		return nil, nil, helpMessage, err
	}
	// if name is not set, generate a random one
	if pc.GetBaseConfig().Name == "" {
		id, err := util.RandIDWithLen(8)
		if err != nil {
			return nil, nil, helpMessage, fmt.Errorf("generate random id error: %v", err)
		}
		pc.GetBaseConfig().Name = fmt.Sprintf("sshtunnel-%s-%s", proxyType, id)
	}
	return &clientCfg, pc, helpMessage, nil
}

func (s *TunnelServer) handleNewChannel(channel ssh.NewChannel, extraPayloadCh chan string) {
	ch, reqs, err := channel.Accept()
	if err != nil {
		return
	}
	if s.firstChannel == nil {
		s.firstChannel = ch
	}
	go s.keepAlive(ch)

	for req := range reqs {
		if req.WantReply {
			_ = req.Reply(true, nil)
		}
		if req.Type != "exec" || len(req.Payload) <= 4 {
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
		// Note: Here is just for compatibility, not the real source address.
		OriginAddr: addr.Host,
		OriginPort: addr.Port,
	}
	channel, reqs, err := s.sshConn.OpenChannel(ChannelTypeServerOpenChannel, ssh.Marshal(&payload))
	if err != nil {
		return nil, fmt.Errorf("open ssh channel error: %v", err)
	}
	go ssh.DiscardRequests(reqs)

	conn := netpkg.WrapReadWriteCloserToConn(channel, s.underlyingConn)
	return conn, nil
}

func (s *TunnelServer) waitProxyStatusReady(name string, timeout time.Duration) (*proxy.WorkingStatus, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ps, err := s.vc.Service().GetProxyStatus(name)
			if err != nil {
				continue
			}
			switch ps.Phase {
			case proxy.ProxyPhaseRunning:
				return ps, nil
			case proxy.ProxyPhaseStartErr, proxy.ProxyPhaseClosed:
				return ps, errors.New(ps.Err)
			}
		case <-timer.C:
			return nil, fmt.Errorf("wait proxy status ready timeout")
		case <-s.doneCh:
			return nil, fmt.Errorf("ssh tunnel server closed")
		}
	}
}

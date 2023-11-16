// Copyright 2017 fatedier, fatedier@gmail.com
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
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatedier/golib/crypto"
	libdial "github.com/fatedier/golib/net/dial"
	fmux "github.com/hashicorp/yamux"
	quic "github.com/quic-go/quic-go"
	"github.com/samber/lo"

	"github.com/fatedier/frp/assets"
	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/log"
	utilnet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/frp/pkg/util/wait"
	"github.com/fatedier/frp/pkg/util/xlog"
)

func init() {
	crypto.DefaultSalt = "frp"
}

// Service is a client service.
type Service struct {
	// uniq id got from frps, attach it in loginMsg
	runID string

	// manager control connection with server
	ctl   *Control
	ctlMu sync.RWMutex

	// Sets authentication based on selected method
	authSetter auth.Setter

	cfg         *v1.ClientCommonConfig
	pxyCfgs     []v1.ProxyConfigurer
	visitorCfgs []v1.VisitorConfigurer
	cfgMu       sync.RWMutex

	// The configuration file used to initialize this client, or an empty
	// string if no configuration file was used.
	cfgFile string

	// service context
	ctx context.Context
	// call cancel to stop service
	cancel           context.CancelFunc
	gracefulDuration time.Duration
}

func NewService(
	cfg *v1.ClientCommonConfig,
	pxyCfgs []v1.ProxyConfigurer,
	visitorCfgs []v1.VisitorConfigurer,
	cfgFile string,
) *Service {
	return &Service{
		authSetter:  auth.NewAuthSetter(cfg.Auth),
		cfg:         cfg,
		cfgFile:     cfgFile,
		pxyCfgs:     pxyCfgs,
		visitorCfgs: visitorCfgs,
		ctx:         context.Background(),
	}
}

func (svr *Service) GetController() *Control {
	svr.ctlMu.RLock()
	defer svr.ctlMu.RUnlock()
	return svr.ctl
}

func (svr *Service) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	svr.ctx = xlog.NewContext(ctx, xlog.New())
	svr.cancel = cancel

	// set custom DNSServer
	if svr.cfg.DNSServer != "" {
		dnsAddr := svr.cfg.DNSServer
		if _, _, err := net.SplitHostPort(dnsAddr); err != nil {
			dnsAddr = net.JoinHostPort(dnsAddr, "53")
		}
		// Change default dns server for frpc
		net.DefaultResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return net.Dial("udp", dnsAddr)
			},
		}
	}

	// login to frps
	svr.loopLoginUntilSuccess(10*time.Second, lo.FromPtr(svr.cfg.LoginFailExit))
	if svr.ctl == nil {
		return fmt.Errorf("the process exited because the first login to the server failed, and the loginFailExit feature is enabled")
	}

	go svr.keepControllerWorking()

	if svr.cfg.WebServer.Port != 0 {
		// Init admin server assets
		assets.Load(svr.cfg.WebServer.AssetsDir)

		address := net.JoinHostPort(svr.cfg.WebServer.Addr, strconv.Itoa(svr.cfg.WebServer.Port))
		err := svr.RunAdminServer(address)
		if err != nil {
			log.Warn("run admin server error: %v", err)
		}
		log.Info("admin server listen on %s:%d", svr.cfg.WebServer.Addr, svr.cfg.WebServer.Port)
	}
	<-svr.ctx.Done()
	svr.stop()
	return nil
}

func (svr *Service) keepControllerWorking() {
	<-svr.ctl.Done()

	// There is a situation where the login is successful but due to certain reasons,
	// the control immediately exits. It is necessary to limit the frequency of reconnection in this case.
	// The interval for the first three retries in 1 minute will be very short, and then it will increase exponentially.
	// The maximum interval is 20 seconds.
	wait.BackoffUntil(func() error {
		// loopLoginUntilSuccess is another layer of loop that will continuously attempt to
		// login to the server until successful.
		svr.loopLoginUntilSuccess(20*time.Second, false)
		<-svr.ctl.Done()
		return errors.New("control is closed and try another loop")
	}, wait.NewFastBackoffManager(
		wait.FastBackoffOptions{
			Duration:        time.Second,
			Factor:          2,
			Jitter:          0.1,
			MaxDuration:     20 * time.Second,
			FastRetryCount:  3,
			FastRetryDelay:  200 * time.Millisecond,
			FastRetryWindow: time.Minute,
			FastRetryJitter: 0.5,
		},
	), true, svr.ctx.Done())
}

// login creates a connection to frps and registers it self as a client
// conn: control connection
// session: if it's not nil, using tcp mux
func (svr *Service) login() (conn net.Conn, cm *ConnectionManager, err error) {
	xl := xlog.FromContextSafe(svr.ctx)
	cm = NewConnectionManager(svr.ctx, svr.cfg)

	if err = cm.OpenConnection(); err != nil {
		return nil, nil, err
	}

	defer func() {
		if err != nil {
			cm.Close()
		}
	}()

	conn, err = cm.Connect()
	if err != nil {
		return
	}

	loginMsg := &msg.Login{
		Arch:      runtime.GOARCH,
		Os:        runtime.GOOS,
		PoolCount: svr.cfg.Transport.PoolCount,
		User:      svr.cfg.User,
		Version:   version.Full(),
		Timestamp: time.Now().Unix(),
		RunID:     svr.runID,
		Metas:     svr.cfg.Metadatas,
	}

	// Add auth
	if err = svr.authSetter.SetLogin(loginMsg); err != nil {
		return
	}

	if err = msg.WriteMsg(conn, loginMsg); err != nil {
		return
	}

	var loginRespMsg msg.LoginResp
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err = msg.ReadMsgInto(conn, &loginRespMsg); err != nil {
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	if loginRespMsg.Error != "" {
		err = fmt.Errorf("%s", loginRespMsg.Error)
		xl.Error("%s", loginRespMsg.Error)
		return
	}

	svr.runID = loginRespMsg.RunID
	xl.ResetPrefixes()
	xl.AppendPrefix(svr.runID)

	xl.Info("login to server success, get run id [%s]", loginRespMsg.RunID)
	return
}

func (svr *Service) loopLoginUntilSuccess(maxInterval time.Duration, firstLoginExit bool) {
	xl := xlog.FromContextSafe(svr.ctx)
	successCh := make(chan struct{})

	loginFunc := func() error {
		xl.Info("try to connect to server...")
		conn, cm, err := svr.login()
		if err != nil {
			xl.Warn("connect to server error: %v", err)
			if firstLoginExit {
				svr.cancel()
			}
			return err
		}

		ctl, err := NewControl(svr.ctx, svr.runID, conn, cm,
			svr.cfg, svr.pxyCfgs, svr.visitorCfgs, svr.authSetter)
		if err != nil {
			conn.Close()
			xl.Error("NewControl error: %v", err)
			return err
		}

		ctl.Run()
		// close and replace previous control
		svr.ctlMu.Lock()
		if svr.ctl != nil {
			svr.ctl.Close()
		}
		svr.ctl = ctl
		svr.ctlMu.Unlock()

		close(successCh)
		return nil
	}

	// try to reconnect to server until success
	wait.BackoffUntil(loginFunc, wait.NewFastBackoffManager(
		wait.FastBackoffOptions{
			Duration:    time.Second,
			Factor:      2,
			Jitter:      0.1,
			MaxDuration: maxInterval,
		}),
		true,
		wait.MergeAndCloseOnAnyStopChannel(svr.ctx.Done(), successCh))
}

func (svr *Service) ReloadConf(pxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error {
	svr.cfgMu.Lock()
	svr.pxyCfgs = pxyCfgs
	svr.visitorCfgs = visitorCfgs
	svr.cfgMu.Unlock()

	svr.ctlMu.RLock()
	ctl := svr.ctl
	svr.ctlMu.RUnlock()

	if ctl != nil {
		return svr.ctl.ReloadConf(pxyCfgs, visitorCfgs)
	}
	return nil
}

func (svr *Service) Close() {
	svr.GracefulClose(time.Duration(0))
}

func (svr *Service) GracefulClose(d time.Duration) {
	svr.gracefulDuration = d
	svr.cancel()
}

func (svr *Service) stop() {
	svr.ctlMu.Lock()
	defer svr.ctlMu.Unlock()
	if svr.ctl != nil {
		svr.ctl.GracefulClose(svr.gracefulDuration)
		svr.ctl = nil
	}
}

// ConnectionManager is a wrapper for establishing connections to the server.
type ConnectionManager struct {
	ctx context.Context
	cfg *v1.ClientCommonConfig

	muxSession *fmux.Session
	quicConn   quic.Connection
}

func NewConnectionManager(ctx context.Context, cfg *v1.ClientCommonConfig) *ConnectionManager {
	return &ConnectionManager{
		ctx: ctx,
		cfg: cfg,
	}
}

// OpenConnection opens a underlying connection to the server.
// The underlying connection is either a TCP connection or a QUIC connection.
// After the underlying connection is established, you can call Connect() to get a stream.
// If TCPMux isn't enabled, the underlying connection is nil, you will get a new real TCP connection every time you call Connect().
func (cm *ConnectionManager) OpenConnection() error {
	xl := xlog.FromContextSafe(cm.ctx)

	// special for quic
	if strings.EqualFold(cm.cfg.Transport.Protocol, "quic") {
		var tlsConfig *tls.Config
		var err error
		sn := cm.cfg.Transport.TLS.ServerName
		if sn == "" {
			sn = cm.cfg.ServerAddr
		}
		if lo.FromPtr(cm.cfg.Transport.TLS.Enable) {
			tlsConfig, err = transport.NewClientTLSConfig(
				cm.cfg.Transport.TLS.CertFile,
				cm.cfg.Transport.TLS.KeyFile,
				cm.cfg.Transport.TLS.TrustedCaFile,
				sn)
		} else {
			tlsConfig, err = transport.NewClientTLSConfig("", "", "", sn)
		}
		if err != nil {
			xl.Warn("fail to build tls configuration, err: %v", err)
			return err
		}
		tlsConfig.NextProtos = []string{"frp"}

		conn, err := quic.DialAddr(
			cm.ctx,
			net.JoinHostPort(cm.cfg.ServerAddr, strconv.Itoa(cm.cfg.ServerPort)),
			tlsConfig, &quic.Config{
				MaxIdleTimeout:     time.Duration(cm.cfg.Transport.QUIC.MaxIdleTimeout) * time.Second,
				MaxIncomingStreams: int64(cm.cfg.Transport.QUIC.MaxIncomingStreams),
				KeepAlivePeriod:    time.Duration(cm.cfg.Transport.QUIC.KeepalivePeriod) * time.Second,
			})
		if err != nil {
			return err
		}
		cm.quicConn = conn
		return nil
	}

	if !lo.FromPtr(cm.cfg.Transport.TCPMux) {
		return nil
	}

	conn, err := cm.realConnect()
	if err != nil {
		return err
	}

	fmuxCfg := fmux.DefaultConfig()
	fmuxCfg.KeepAliveInterval = time.Duration(cm.cfg.Transport.TCPMuxKeepaliveInterval) * time.Second
	fmuxCfg.LogOutput = io.Discard
	fmuxCfg.MaxStreamWindowSize = 6 * 1024 * 1024
	session, err := fmux.Client(conn, fmuxCfg)
	if err != nil {
		return err
	}
	cm.muxSession = session
	return nil
}

// Connect returns a stream from the underlying connection, or a new TCP connection if TCPMux isn't enabled.
func (cm *ConnectionManager) Connect() (net.Conn, error) {
	if cm.quicConn != nil {
		stream, err := cm.quicConn.OpenStreamSync(context.Background())
		if err != nil {
			return nil, err
		}
		return utilnet.QuicStreamToNetConn(stream, cm.quicConn), nil
	} else if cm.muxSession != nil {
		stream, err := cm.muxSession.OpenStream()
		if err != nil {
			return nil, err
		}
		return stream, nil
	}

	return cm.realConnect()
}

func (cm *ConnectionManager) realConnect() (net.Conn, error) {
	xl := xlog.FromContextSafe(cm.ctx)
	var tlsConfig *tls.Config
	var err error
	tlsEnable := lo.FromPtr(cm.cfg.Transport.TLS.Enable)
	if cm.cfg.Transport.Protocol == "wss" {
		tlsEnable = true
	}
	if tlsEnable {
		sn := cm.cfg.Transport.TLS.ServerName
		if sn == "" {
			sn = cm.cfg.ServerAddr
		}

		tlsConfig, err = transport.NewClientTLSConfig(
			cm.cfg.Transport.TLS.CertFile,
			cm.cfg.Transport.TLS.KeyFile,
			cm.cfg.Transport.TLS.TrustedCaFile,
			sn)
		if err != nil {
			xl.Warn("fail to build tls configuration, err: %v", err)
			return nil, err
		}
	}

	proxyType, addr, auth, err := libdial.ParseProxyURL(cm.cfg.Transport.ProxyURL)
	if err != nil {
		xl.Error("fail to parse proxy url")
		return nil, err
	}
	dialOptions := []libdial.DialOption{}
	protocol := cm.cfg.Transport.Protocol
	switch protocol {
	case "websocket":
		protocol = "tcp"
		dialOptions = append(dialOptions, libdial.WithAfterHook(libdial.AfterHook{Hook: utilnet.DialHookWebsocket(protocol, "")}))
		dialOptions = append(dialOptions, libdial.WithAfterHook(libdial.AfterHook{
			Hook: utilnet.DialHookCustomTLSHeadByte(tlsConfig != nil, lo.FromPtr(cm.cfg.Transport.TLS.DisableCustomTLSFirstByte)),
		}))
		dialOptions = append(dialOptions, libdial.WithTLSConfig(tlsConfig))
	case "wss":
		protocol = "tcp"
		dialOptions = append(dialOptions, libdial.WithTLSConfigAndPriority(100, tlsConfig))
		// Make sure that if it is wss, the websocket hook is executed after the tls hook.
		dialOptions = append(dialOptions, libdial.WithAfterHook(libdial.AfterHook{Hook: utilnet.DialHookWebsocket(protocol, tlsConfig.ServerName), Priority: 110}))
	default:
		dialOptions = append(dialOptions, libdial.WithAfterHook(libdial.AfterHook{
			Hook: utilnet.DialHookCustomTLSHeadByte(tlsConfig != nil, lo.FromPtr(cm.cfg.Transport.TLS.DisableCustomTLSFirstByte)),
		}))
		dialOptions = append(dialOptions, libdial.WithTLSConfig(tlsConfig))
	}

	if cm.cfg.Transport.ConnectServerLocalIP != "" {
		dialOptions = append(dialOptions, libdial.WithLocalAddr(cm.cfg.Transport.ConnectServerLocalIP))
	}
	dialOptions = append(dialOptions,
		libdial.WithProtocol(protocol),
		libdial.WithTimeout(time.Duration(cm.cfg.Transport.DialServerTimeout)*time.Second),
		libdial.WithKeepAlive(time.Duration(cm.cfg.Transport.DialServerKeepAlive)*time.Second),
		libdial.WithProxy(proxyType, addr),
		libdial.WithProxyAuth(auth),
	)
	conn, err := libdial.DialContext(
		cm.ctx,
		net.JoinHostPort(cm.cfg.ServerAddr, strconv.Itoa(cm.cfg.ServerPort)),
		dialOptions...,
	)
	return conn, err
}

func (cm *ConnectionManager) Close() error {
	if cm.quicConn != nil {
		_ = cm.quicConn.CloseWithError(0, "")
	}
	if cm.muxSession != nil {
		_ = cm.muxSession.Close()
	}
	return nil
}

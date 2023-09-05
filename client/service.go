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
	"fmt"
	"io"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/version"
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

	exit uint32 // 0 means not exit

	// service context
	ctx context.Context
	// call cancel to stop service
	cancel context.CancelFunc
}

func NewService(
	cfg *v1.ClientCommonConfig,
	pxyCfgs []v1.ProxyConfigurer,
	visitorCfgs []v1.VisitorConfigurer,
	cfgFile string,
) (svr *Service, err error) {
	svr = &Service{
		authSetter:  auth.NewAuthSetter(cfg.Auth),
		cfg:         cfg,
		cfgFile:     cfgFile,
		pxyCfgs:     pxyCfgs,
		visitorCfgs: visitorCfgs,
		ctx:         context.Background(),
		exit:        0,
	}
	return
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

	xl := xlog.FromContextSafe(svr.ctx)

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
	for {
		conn, cm, err := svr.login()
		if err != nil {
			xl.Warn("login to server failed: %v", err)

			// if login_fail_exit is true, just exit this program
			// otherwise sleep a while and try again to connect to server
			if lo.FromPtr(svr.cfg.LoginFailExit) {
				return err
			}
			util.RandomSleep(5*time.Second, 0.9, 1.1)
		} else {
			// login success
			ctl := NewControl(svr.ctx, svr.runID, conn, cm, svr.cfg, svr.pxyCfgs, svr.visitorCfgs, svr.authSetter)
			ctl.Run()
			svr.ctlMu.Lock()
			svr.ctl = ctl
			svr.ctlMu.Unlock()
			break
		}
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
	// service context may not be canceled by svr.Close(), we should call it here to release resources
	if atomic.LoadUint32(&svr.exit) == 0 {
		svr.Close()
	}
	return nil
}

func (svr *Service) keepControllerWorking() {
	xl := xlog.FromContextSafe(svr.ctx)
	maxDelayTime := 20 * time.Second
	delayTime := time.Second

	// if frpc reconnect frps, we need to limit retry times in 1min
	// current retry logic is sleep 0s, 0s, 0s, 1s, 2s, 4s, 8s, ...
	// when exceed 1min, we will reset delay and counts
	cutoffTime := time.Now().Add(time.Minute)
	reconnectDelay := time.Second
	reconnectCounts := 1

	for {
		<-svr.ctl.ClosedDoneCh()
		if atomic.LoadUint32(&svr.exit) != 0 {
			return
		}

		// the first three attempts with a low delay
		if reconnectCounts > 3 {
			util.RandomSleep(reconnectDelay, 0.9, 1.1)
			xl.Info("wait %v to reconnect", reconnectDelay)
			reconnectDelay *= 2
		} else {
			util.RandomSleep(time.Second, 0, 0.5)
		}
		reconnectCounts++

		now := time.Now()
		if now.After(cutoffTime) {
			// reset
			cutoffTime = now.Add(time.Minute)
			reconnectDelay = time.Second
			reconnectCounts = 1
		}

		for {
			if atomic.LoadUint32(&svr.exit) != 0 {
				return
			}

			xl.Info("try to reconnect to server...")
			conn, cm, err := svr.login()
			if err != nil {
				xl.Warn("reconnect to server error: %v, wait %v for another retry", err, delayTime)
				util.RandomSleep(delayTime, 0.9, 1.1)

				delayTime *= 2
				if delayTime > maxDelayTime {
					delayTime = maxDelayTime
				}
				continue
			}
			// reconnect success, init delayTime
			delayTime = time.Second

			ctl := NewControl(svr.ctx, svr.runID, conn, cm, svr.cfg, svr.pxyCfgs, svr.visitorCfgs, svr.authSetter)
			ctl.Run()
			svr.ctlMu.Lock()
			if svr.ctl != nil {
				svr.ctl.Close()
			}
			svr.ctl = ctl
			svr.ctlMu.Unlock()
			break
		}
	}
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
	atomic.StoreUint32(&svr.exit, 1)

	svr.ctlMu.RLock()
	if svr.ctl != nil {
		svr.ctl.GracefulClose(d)
		svr.ctl = nil
	}
	svr.ctlMu.RUnlock()

	if svr.cancel != nil {
		svr.cancel()
	}
}

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

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
	"errors"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/fatedier/golib/crypto"
	"github.com/samber/lo"

	"github.com/fatedier/frp/assets"
	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/log"
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

	connectorCreator   func(context.Context, *v1.ClientCommonConfig) Connector
	inWorkConnCallback func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool
}

func NewService(
	cfg *v1.ClientCommonConfig,
	pxyCfgs []v1.ProxyConfigurer,
	visitorCfgs []v1.VisitorConfigurer,
	cfgFile string,
) *Service {
	return &Service{
		authSetter:       auth.NewAuthSetter(cfg.Auth),
		cfg:              cfg,
		cfgFile:          cfgFile,
		pxyCfgs:          pxyCfgs,
		visitorCfgs:      visitorCfgs,
		ctx:              context.Background(),
		connectorCreator: NewConnector,
	}
}

func (svr *Service) SetConnectorCreator(h func(context.Context, *v1.ClientCommonConfig) Connector) {
	svr.connectorCreator = h
}

func (svr *Service) SetInWorkConnCallback(cb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool) {
	svr.inWorkConnCallback = cb
}

func (svr *Service) GetController() *Control {
	svr.ctlMu.RLock()
	defer svr.ctlMu.RUnlock()
	return svr.ctl
}

func (svr *Service) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	svr.ctx = xlog.NewContext(ctx, xlog.FromContextSafe(ctx))
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
func (svr *Service) login() (conn net.Conn, connector Connector, err error) {
	xl := xlog.FromContextSafe(svr.ctx)
	connector = svr.connectorCreator(svr.ctx, svr.cfg)
	if err = connector.Open(); err != nil {
		return nil, nil, err
	}

	defer func() {
		if err != nil {
			connector.Close()
		}
	}()

	conn, err = connector.Connect()
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
	xl.AddPrefix(xlog.LogPrefix{Name: "runID", Value: svr.runID})

	xl.Info("login to server success, get run id [%s]", loginRespMsg.RunID)
	return
}

func (svr *Service) loopLoginUntilSuccess(maxInterval time.Duration, firstLoginExit bool) {
	xl := xlog.FromContextSafe(svr.ctx)
	successCh := make(chan struct{})

	loginFunc := func() error {
		xl.Info("try to connect to server...")
		conn, connector, err := svr.login()
		if err != nil {
			xl.Warn("connect to server error: %v", err)
			if firstLoginExit {
				svr.cancel()
			}
			return err
		}

		ctl, err := NewControl(svr.ctx, svr.runID, conn, connector,
			svr.cfg, svr.pxyCfgs, svr.visitorCfgs, svr.authSetter)
		if err != nil {
			conn.Close()
			xl.Error("NewControl error: %v", err)
			return err
		}
		ctl.SetInWorkConnCallback(svr.inWorkConnCallback)

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

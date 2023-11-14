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

package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/fatedier/golib/net/mux"
	fmux "github.com/hashicorp/yamux"
	quic "github.com/quic-go/quic-go"
	"github.com/samber/lo"
	"golang.org/x/crypto/ssh"

	"github.com/fatedier/frp/assets"
	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	modelmetrics "github.com/fatedier/frp/pkg/metrics"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/nathole"
	plugin "github.com/fatedier/frp/pkg/plugin/server"
	frpssh "github.com/fatedier/frp/pkg/ssh"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/log"
	utilnet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/tcpmux"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/version"
	"github.com/fatedier/frp/pkg/util/vhost"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/server/controller"
	"github.com/fatedier/frp/server/group"
	"github.com/fatedier/frp/server/metrics"
	"github.com/fatedier/frp/server/ports"
	"github.com/fatedier/frp/server/proxy"
	"github.com/fatedier/frp/server/visitor"
)

const (
	connReadTimeout       time.Duration = 10 * time.Second
	vhostReadWriteTimeout time.Duration = 30 * time.Second
)

// Server service
type Service struct {
	// Dispatch connections to different handlers listen on same port
	muxer *mux.Mux

	// Accept connections from client
	listener net.Listener

	// Accept connections using ssh
	sshListener net.Listener
	sshConfig   *ssh.ServerConfig

	// Accept connections using kcp
	kcpListener net.Listener

	// Accept connections using quic
	quicListener *quic.Listener

	// Accept connections using websocket
	websocketListener net.Listener

	// Accept frp tls connections
	tlsListener net.Listener

	// Manage all controllers
	ctlManager *ControlManager

	// Manage all proxies
	pxyManager *proxy.Manager

	// Manage all plugins
	pluginManager *plugin.Manager

	// HTTP vhost router
	httpVhostRouter *vhost.Routers

	// All resource managers and controllers
	rc *controller.ResourceController

	// Verifies authentication based on selected method
	authVerifier auth.Verifier

	tlsConfig *tls.Config

	cfg *v1.ServerConfig

	// service context
	ctx context.Context
	// call cancel to stop service
	cancel context.CancelFunc
}

func NewService(cfg *v1.ServerConfig) (svr *Service, err error) {
	tlsConfig, err := transport.NewServerTLSConfig(
		cfg.Transport.TLS.CertFile,
		cfg.Transport.TLS.KeyFile,
		cfg.Transport.TLS.TrustedCaFile)
	if err != nil {
		return
	}

	svr = &Service{
		ctlManager:    NewControlManager(),
		pxyManager:    proxy.NewManager(),
		pluginManager: plugin.NewManager(),
		rc: &controller.ResourceController{
			VisitorManager: visitor.NewManager(),
			TCPPortManager: ports.NewManager("tcp", cfg.ProxyBindAddr, cfg.AllowPorts),
			UDPPortManager: ports.NewManager("udp", cfg.ProxyBindAddr, cfg.AllowPorts),
		},
		httpVhostRouter: vhost.NewRouters(),
		authVerifier:    auth.NewAuthVerifier(cfg.Auth),
		tlsConfig:       tlsConfig,
		cfg:             cfg,
		ctx:             context.Background(),
	}

	// Create tcpmux httpconnect multiplexer.
	if cfg.TCPMuxHTTPConnectPort > 0 {
		var l net.Listener
		address := net.JoinHostPort(cfg.ProxyBindAddr, strconv.Itoa(cfg.TCPMuxHTTPConnectPort))
		l, err = net.Listen("tcp", address)
		if err != nil {
			err = fmt.Errorf("create server listener error, %v", err)
			return
		}

		svr.rc.TCPMuxHTTPConnectMuxer, err = tcpmux.NewHTTPConnectTCPMuxer(l, cfg.TCPMuxPassthrough, vhostReadWriteTimeout)
		if err != nil {
			err = fmt.Errorf("create vhost tcpMuxer error, %v", err)
			return
		}
		log.Info("tcpmux httpconnect multiplexer listen on %s, passthough: %v", address, cfg.TCPMuxPassthrough)
	}

	// Init all plugins
	for _, p := range cfg.HTTPPlugins {
		svr.pluginManager.Register(plugin.NewHTTPPluginOptions(p))
		log.Info("plugin [%s] has been registered", p.Name)
	}
	svr.rc.PluginManager = svr.pluginManager

	// Init group controller
	svr.rc.TCPGroupCtl = group.NewTCPGroupCtl(svr.rc.TCPPortManager)

	// Init HTTP group controller
	svr.rc.HTTPGroupCtl = group.NewHTTPGroupController(svr.httpVhostRouter)

	// Init TCP mux group controller
	svr.rc.TCPMuxGroupCtl = group.NewTCPMuxGroupCtl(svr.rc.TCPMuxHTTPConnectMuxer)

	// Init 404 not found page
	vhost.NotFoundPagePath = cfg.Custom404Page

	var (
		httpMuxOn  bool
		httpsMuxOn bool
	)
	if cfg.BindAddr == cfg.ProxyBindAddr {
		if cfg.BindPort == cfg.VhostHTTPPort {
			httpMuxOn = true
		}
		if cfg.BindPort == cfg.VhostHTTPSPort {
			httpsMuxOn = true
		}
	}

	// Listen for accepting connections from client.
	address := net.JoinHostPort(cfg.BindAddr, strconv.Itoa(cfg.BindPort))
	ln, err := net.Listen("tcp", address)
	if err != nil {
		err = fmt.Errorf("create server listener error, %v", err)
		return
	}

	svr.muxer = mux.NewMux(ln)
	svr.muxer.SetKeepAlive(time.Duration(cfg.Transport.TCPKeepAlive) * time.Second)
	go func() {
		_ = svr.muxer.Serve()
	}()
	ln = svr.muxer.DefaultListener()

	svr.listener = ln
	log.Info("frps tcp listen on %s", address)

	if cfg.SSHTunnelGateway.BindPort > 0 {

		if cfg.SSHTunnelGateway.PublicKeyFilesPath != "" {
			cfg.SSHTunnelGateway.PublicKeyFilesMap, err = v1.LoadSSHPublicKeyFilesInDir(cfg.SSHTunnelGateway.PublicKeyFilesPath)
			if err != nil {
				return nil, fmt.Errorf("load ssh all public key files error: %v", err)
			}
			log.Info("load %v public key files success", cfg.SSHTunnelGateway.PublicKeyFilesPath)
		}

		svr.sshConfig = &ssh.ServerConfig{
			NoClientAuth: lo.If(cfg.SSHTunnelGateway.PublicKeyFilesPath == "", true).Else(false),

			PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
				parsedAuthorizedKey, ok := cfg.SSHTunnelGateway.PublicKeyFilesMap[ssh.FingerprintSHA256(key)]
				if !ok {
					return nil, errors.New("cannot find public key file")
				}

				if key.Type() == parsedAuthorizedKey.Type() && reflect.DeepEqual(parsedAuthorizedKey, key) {
					return &ssh.Permissions{
						Extensions: map[string]string{},
					}, nil
				}
				return nil, fmt.Errorf("unknown public key for %q", conn.User())
			},
		}

		var privateBytes []byte
		if cfg.SSHTunnelGateway.PrivateKeyFilePath != "" {
			privateBytes, err = os.ReadFile(cfg.SSHTunnelGateway.PrivateKeyFilePath)
			if err != nil {
				log.Error("Failed to load private key")
				return nil, err
			}
			log.Info("load %v private key file success", cfg.SSHTunnelGateway.PrivateKeyFilePath)
		} else {
			privateBytes, err = v1.GeneratePrivateKey()
			if err != nil {
				log.Error("Failed to load private key")
				return nil, err
			}
			log.Info("auto gen private key file success")
		}
		private, err := ssh.ParsePrivateKey(privateBytes)
		if err != nil {
			log.Error("Failed to parse private key, error: %v", err)
			return nil, err
		}

		svr.sshConfig.AddHostKey(private)

		sshAddr := net.JoinHostPort(cfg.BindAddr, strconv.Itoa(cfg.SSHTunnelGateway.BindPort))
		svr.sshListener, err = net.Listen("tcp", sshAddr)
		if err != nil {
			log.Error("Failed to listen on %v, error: %v", sshAddr, err)
			return nil, err
		}
		log.Info("ssh server listening on %v", sshAddr)
	}

	// Listen for accepting connections from client using kcp protocol.
	if cfg.KCPBindPort > 0 {
		address := net.JoinHostPort(cfg.BindAddr, strconv.Itoa(cfg.KCPBindPort))
		svr.kcpListener, err = utilnet.ListenKcp(address)
		if err != nil {
			err = fmt.Errorf("listen on kcp udp address %s error: %v", address, err)
			return
		}
		log.Info("frps kcp listen on udp %s", address)
	}

	if cfg.QUICBindPort > 0 {
		address := net.JoinHostPort(cfg.BindAddr, strconv.Itoa(cfg.QUICBindPort))
		quicTLSCfg := tlsConfig.Clone()
		quicTLSCfg.NextProtos = []string{"frp"}
		svr.quicListener, err = quic.ListenAddr(address, quicTLSCfg, &quic.Config{
			MaxIdleTimeout:     time.Duration(cfg.Transport.QUIC.MaxIdleTimeout) * time.Second,
			MaxIncomingStreams: int64(cfg.Transport.QUIC.MaxIncomingStreams),
			KeepAlivePeriod:    time.Duration(cfg.Transport.QUIC.KeepalivePeriod) * time.Second,
		})
		if err != nil {
			err = fmt.Errorf("listen on quic udp address %s error: %v", address, err)
			return
		}
		log.Info("frps quic listen on quic %s", address)
	}

	// Listen for accepting connections from client using websocket protocol.
	websocketPrefix := []byte("GET " + utilnet.FrpWebsocketPath)
	websocketLn := svr.muxer.Listen(0, uint32(len(websocketPrefix)), func(data []byte) bool {
		return bytes.Equal(data, websocketPrefix)
	})
	svr.websocketListener = utilnet.NewWebsocketListener(websocketLn)

	// Create http vhost muxer.
	if cfg.VhostHTTPPort > 0 {
		rp := vhost.NewHTTPReverseProxy(vhost.HTTPReverseProxyOptions{
			ResponseHeaderTimeoutS: cfg.VhostHTTPTimeout,
		}, svr.httpVhostRouter)
		svr.rc.HTTPReverseProxy = rp

		address := net.JoinHostPort(cfg.ProxyBindAddr, strconv.Itoa(cfg.VhostHTTPPort))
		server := &http.Server{
			Addr:    address,
			Handler: rp,
		}
		var l net.Listener
		if httpMuxOn {
			l = svr.muxer.ListenHttp(1)
		} else {
			l, err = net.Listen("tcp", address)
			if err != nil {
				err = fmt.Errorf("create vhost http listener error, %v", err)
				return
			}
		}
		go func() {
			_ = server.Serve(l)
		}()
		log.Info("http service listen on %s", address)
	}

	// Create https vhost muxer.
	if cfg.VhostHTTPSPort > 0 {
		var l net.Listener
		if httpsMuxOn {
			l = svr.muxer.ListenHttps(1)
		} else {
			address := net.JoinHostPort(cfg.ProxyBindAddr, strconv.Itoa(cfg.VhostHTTPSPort))
			l, err = net.Listen("tcp", address)
			if err != nil {
				err = fmt.Errorf("create server listener error, %v", err)
				return
			}
			log.Info("https service listen on %s", address)
		}

		svr.rc.VhostHTTPSMuxer, err = vhost.NewHTTPSMuxer(l, vhostReadWriteTimeout)
		if err != nil {
			err = fmt.Errorf("create vhost httpsMuxer error, %v", err)
			return
		}
	}

	// frp tls listener
	svr.tlsListener = svr.muxer.Listen(2, 1, func(data []byte) bool {
		// tls first byte can be 0x16 only when vhost https port is not same with bind port
		return int(data[0]) == utilnet.FRPTLSHeadByte || int(data[0]) == 0x16
	})

	// Create nat hole controller.
	nc, err := nathole.NewController(time.Duration(cfg.NatHoleAnalysisDataReserveHours) * time.Hour)
	if err != nil {
		err = fmt.Errorf("create nat hole controller error, %v", err)
		return
	}
	svr.rc.NatHoleController = nc

	var statsEnable bool
	// Create dashboard web server.
	if cfg.WebServer.Port > 0 {
		// Init dashboard assets
		assets.Load(cfg.WebServer.AssetsDir)

		address := net.JoinHostPort(cfg.WebServer.Addr, strconv.Itoa(cfg.WebServer.Port))
		err = svr.RunDashboardServer(address)
		if err != nil {
			err = fmt.Errorf("create dashboard web server error, %v", err)
			return
		}
		log.Info("Dashboard listen on %s", address)
		statsEnable = true
	}
	if statsEnable {
		modelmetrics.EnableMem()
		if cfg.EnablePrometheus {
			modelmetrics.EnablePrometheus()
		}
	}
	return
}

func (svr *Service) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	svr.ctx = ctx
	svr.cancel = cancel

	if svr.sshListener != nil {
		go svr.HandleSSHListener(svr.sshListener)
	}

	if svr.kcpListener != nil {
		go svr.HandleListener(svr.kcpListener)
	}
	if svr.quicListener != nil {
		go svr.HandleQUICListener(svr.quicListener)
	}
	go svr.HandleListener(svr.websocketListener)
	go svr.HandleListener(svr.tlsListener)

	if svr.rc.NatHoleController != nil {
		go svr.rc.NatHoleController.CleanWorker(svr.ctx)
	}
	svr.HandleListener(svr.listener)

	<-svr.ctx.Done()
	// service context may not be canceled by svr.Close(), we should call it here to release resources
	if svr.listener != nil {
		svr.Close()
	}
}

func (svr *Service) Close() error {
	if svr.sshListener != nil {
		svr.sshListener.Close()
		svr.sshListener = nil
	}
	if svr.kcpListener != nil {
		svr.kcpListener.Close()
		svr.kcpListener = nil
	}
	if svr.quicListener != nil {
		svr.quicListener.Close()
		svr.quicListener = nil
	}
	if svr.websocketListener != nil {
		svr.websocketListener.Close()
		svr.websocketListener = nil
	}
	if svr.tlsListener != nil {
		svr.tlsListener.Close()
		svr.tlsConfig = nil
	}
	if svr.listener != nil {
		svr.listener.Close()
		svr.listener = nil
	}
	svr.ctlManager.Close()
	if svr.cancel != nil {
		svr.cancel()
	}
	return nil
}

func (svr *Service) handleConnection(ctx context.Context, conn net.Conn) {
	xl := xlog.FromContextSafe(ctx)

	var (
		rawMsg msg.Message
		err    error
	)

	_ = conn.SetReadDeadline(time.Now().Add(connReadTimeout))
	if rawMsg, err = msg.ReadMsg(conn); err != nil {
		log.Trace("Failed to read message: %v", err)
		conn.Close()
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	switch m := rawMsg.(type) {
	case *msg.Login:
		// server plugin hook
		content := &plugin.LoginContent{
			Login:         *m,
			ClientAddress: conn.RemoteAddr().String(),
		}
		retContent, err := svr.pluginManager.Login(content)
		if err == nil {
			m = &retContent.Login
			err = svr.RegisterControl(conn, m)
		}

		// If login failed, send error message there.
		// Otherwise send success message in control's work goroutine.
		if err != nil {
			xl.Warn("register control error: %v", err)
			_ = msg.WriteMsg(conn, &msg.LoginResp{
				Version: version.Full(),
				Error:   util.GenerateResponseErrorString("register control error", err, lo.FromPtr(svr.cfg.DetailedErrorsToClient)),
			})
			conn.Close()
		}
	case *msg.NewWorkConn:
		if err := svr.RegisterWorkConn(conn, m); err != nil {
			conn.Close()
		}
	case *msg.NewVisitorConn:
		if err = svr.RegisterVisitorConn(conn, m); err != nil {
			xl.Warn("register visitor conn error: %v", err)
			_ = msg.WriteMsg(conn, &msg.NewVisitorConnResp{
				ProxyName: m.ProxyName,
				Error:     util.GenerateResponseErrorString("register visitor conn error", err, lo.FromPtr(svr.cfg.DetailedErrorsToClient)),
			})
			conn.Close()
		} else {
			_ = msg.WriteMsg(conn, &msg.NewVisitorConnResp{
				ProxyName: m.ProxyName,
				Error:     "",
			})
		}
	default:
		log.Warn("Error message type for the new connection [%s]", conn.RemoteAddr().String())
		conn.Close()
	}
}

func (svr *Service) HandleListener(l net.Listener) {
	// Listen for incoming connections from client.
	for {
		c, err := l.Accept()
		if err != nil {
			log.Warn("Listener for incoming connections from client closed")
			return
		}
		// inject xlog object into net.Conn context
		xl := xlog.New()
		ctx := context.Background()

		c = utilnet.NewContextConn(xlog.NewContext(ctx, xl), c)

		log.Trace("start check TLS connection...")
		originConn := c
		var isTLS, custom bool
		c, isTLS, custom, err = utilnet.CheckAndEnableTLSServerConnWithTimeout(c, svr.tlsConfig, svr.cfg.Transport.TLS.Force, connReadTimeout)
		if err != nil {
			log.Warn("CheckAndEnableTLSServerConnWithTimeout error: %v", err)
			originConn.Close()
			continue
		}
		log.Trace("check TLS connection success, isTLS: %v custom: %v", isTLS, custom)

		// Start a new goroutine to handle connection.
		go func(ctx context.Context, frpConn net.Conn) {
			if lo.FromPtr(svr.cfg.Transport.TCPMux) {
				fmuxCfg := fmux.DefaultConfig()
				fmuxCfg.KeepAliveInterval = time.Duration(svr.cfg.Transport.TCPMuxKeepaliveInterval) * time.Second
				fmuxCfg.LogOutput = io.Discard
				fmuxCfg.MaxStreamWindowSize = 6 * 1024 * 1024
				session, err := fmux.Server(frpConn, fmuxCfg)
				if err != nil {
					log.Warn("Failed to create mux connection: %v", err)
					frpConn.Close()
					return
				}

				for {
					stream, err := session.AcceptStream()
					if err != nil {
						log.Debug("Accept new mux stream error: %v", err)
						session.Close()
						return
					}
					go svr.handleConnection(ctx, stream)
				}
			} else {
				svr.handleConnection(ctx, frpConn)
			}
		}(ctx, c)
	}
}

func (svr *Service) HandleSSHListener(listener net.Listener) {
	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Error("failed to accept incoming ssh connection (%s)", err)
			return
		}
		log.Info("new tcp conn connected: %v", tcpConn.RemoteAddr().String())

		pxyPayloadCh := make(chan v1.ProxyConfigurer)
		replyCh := make(chan interface{})

		ss, err := frpssh.NewSSHService(tcpConn, svr.sshConfig, pxyPayloadCh, replyCh)
		if err != nil {
			log.Error("new ssh service error: %v", err)
			continue
		}
		ss.Run()

		go func() {
			for {
				pxyCfg := <-pxyPayloadCh

				ctx := context.Background()

				// TODO fill client common config and login msg
				vs, err := frpssh.NewVirtualService(ctx, v1.ClientCommonConfig{}, *svr.cfg,
					msg.Login{User: v1.SSHClientLoginUserPrefix + tcpConn.RemoteAddr().String()},
					svr.rc, pxyCfg, ss, replyCh)
				if err != nil {
					log.Error("new virtual service error: %v", err)
					ss.Close()
					return
				}

				err = vs.Run(ctx)
				if err != nil {
					log.Error("proxy run error: %v", err)
					vs.Close()
					return
				}
			}
		}()
	}
}

func (svr *Service) HandleQUICListener(l *quic.Listener) {
	// Listen for incoming connections from client.
	for {
		c, err := l.Accept(context.Background())
		if err != nil {
			log.Warn("QUICListener for incoming connections from client closed")
			return
		}
		// Start a new goroutine to handle connection.
		go func(ctx context.Context, frpConn quic.Connection) {
			for {
				stream, err := frpConn.AcceptStream(context.Background())
				if err != nil {
					log.Debug("Accept new quic mux stream error: %v", err)
					_ = frpConn.CloseWithError(0, "")
					return
				}
				go svr.handleConnection(ctx, utilnet.QuicStreamToNetConn(stream, frpConn))
			}
		}(context.Background(), c)
	}
}

func (svr *Service) RegisterControl(ctlConn net.Conn, loginMsg *msg.Login) error {
	// If client's RunID is empty, it's a new client, we just create a new controller.
	// Otherwise, we check if there is one controller has the same run id. If so, we release previous controller and start new one.
	var err error
	if loginMsg.RunID == "" {
		loginMsg.RunID, err = util.RandID()
		if err != nil {
			return err
		}
	}

	ctx := utilnet.NewContextFromConn(ctlConn)
	xl := xlog.FromContextSafe(ctx)
	xl.AppendPrefix(loginMsg.RunID)
	ctx = xlog.NewContext(ctx, xl)
	xl.Info("client login info: ip [%s] version [%s] hostname [%s] os [%s] arch [%s]",
		ctlConn.RemoteAddr().String(), loginMsg.Version, loginMsg.Hostname, loginMsg.Os, loginMsg.Arch)

	// Check auth.
	if err := svr.authVerifier.VerifyLogin(loginMsg); err != nil {
		return err
	}

	ctl, err := NewControl(ctx, svr.rc, svr.pxyManager, svr.pluginManager, svr.authVerifier, ctlConn, loginMsg, svr.cfg)
	if err != nil {
		xl.Warn("create new controller error: %v", err)
		// don't return detailed errors to client
		return fmt.Errorf("unexpected error when creating new controller")
	}
	if oldCtl := svr.ctlManager.Add(loginMsg.RunID, ctl); oldCtl != nil {
		oldCtl.WaitClosed()
	}

	ctl.Start()

	// for statistics
	metrics.Server.NewClient()

	go func() {
		// block until control closed
		ctl.WaitClosed()
		svr.ctlManager.Del(loginMsg.RunID, ctl)
	}()
	return nil
}

// RegisterWorkConn register a new work connection to control and proxies need it.
func (svr *Service) RegisterWorkConn(workConn net.Conn, newMsg *msg.NewWorkConn) error {
	xl := utilnet.NewLogFromConn(workConn)
	ctl, exist := svr.ctlManager.GetByID(newMsg.RunID)
	if !exist {
		xl.Warn("No client control found for run id [%s]", newMsg.RunID)
		return fmt.Errorf("no client control found for run id [%s]", newMsg.RunID)
	}
	// server plugin hook
	content := &plugin.NewWorkConnContent{
		User: plugin.UserInfo{
			User:  ctl.loginMsg.User,
			Metas: ctl.loginMsg.Metas,
			RunID: ctl.loginMsg.RunID,
		},
		NewWorkConn: *newMsg,
	}
	retContent, err := svr.pluginManager.NewWorkConn(content)
	if err == nil {
		newMsg = &retContent.NewWorkConn
		// Check auth.
		err = svr.authVerifier.VerifyNewWorkConn(newMsg)
	}
	if err != nil {
		xl.Warn("invalid NewWorkConn with run id [%s]", newMsg.RunID)
		_ = msg.WriteMsg(workConn, &msg.StartWorkConn{
			Error: util.GenerateResponseErrorString("invalid NewWorkConn", err, lo.FromPtr(svr.cfg.DetailedErrorsToClient)),
		})
		return fmt.Errorf("invalid NewWorkConn with run id [%s]", newMsg.RunID)
	}
	return ctl.RegisterWorkConn(workConn)
}

func (svr *Service) RegisterVisitorConn(visitorConn net.Conn, newMsg *msg.NewVisitorConn) error {
	visitorUser := ""
	// TODO(deprecation): Compatible with old versions, can be without runID, user is empty. In later versions, it will be mandatory to include runID.
	// If runID is required, it is not compatible with versions prior to v0.50.0.
	if newMsg.RunID != "" {
		ctl, exist := svr.ctlManager.GetByID(newMsg.RunID)
		if !exist {
			return fmt.Errorf("no client control found for run id [%s]", newMsg.RunID)
		}
		visitorUser = ctl.loginMsg.User
	}
	return svr.rc.VisitorManager.NewConn(newMsg.ProxyName, visitorConn, newMsg.Timestamp, newMsg.SignKey,
		newMsg.UseEncryption, newMsg.UseCompression, visitorUser)
}

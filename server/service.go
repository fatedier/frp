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
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/fatedier/frp/assets"
	"github.com/fatedier/frp/g"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/models/nathole"
	"github.com/fatedier/frp/server/controller"
	"github.com/fatedier/frp/server/group"
	"github.com/fatedier/frp/server/ports"
	"github.com/fatedier/frp/server/proxy"
	"github.com/fatedier/frp/server/stats"
	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/util"
	"github.com/fatedier/frp/utils/version"
	"github.com/fatedier/frp/utils/vhost"

	"github.com/fatedier/golib/net/mux"
	fmux "github.com/hashicorp/yamux"
)

const (
	connReadTimeout time.Duration = 10 * time.Second
)

var ServerService *Service

// Server service
type Service struct {
	// Dispatch connections to different handlers listen on same port
	muxer *mux.Mux

	// Accept connections from client
	listener frpNet.Listener

	// Accept connections using kcp
	kcpListener frpNet.Listener

	// Accept connections using websocket
	websocketListener frpNet.Listener

	// Manage all controllers
	ctlManager *ControlManager

	// Manage all proxies
	pxyManager *proxy.ProxyManager

	// All resource managers and controllers
	rc *controller.ResourceController

	// stats collector to store server and proxies stats info
	statsCollector stats.Collector
}

func NewService() (svr *Service, err error) {
	cfg := &g.GlbServerCfg.ServerCommonConf
	svr = &Service{
		ctlManager: NewControlManager(),
		pxyManager: proxy.NewProxyManager(),
		rc: &controller.ResourceController{
			VisitorManager: controller.NewVisitorManager(),
			TcpPortManager: ports.NewPortManager("tcp", cfg.ProxyBindAddr, cfg.AllowPorts),
			UdpPortManager: ports.NewPortManager("udp", cfg.ProxyBindAddr, cfg.AllowPorts),
		},
	}

	// Init group controller
	svr.rc.TcpGroupCtl = group.NewTcpGroupCtl(svr.rc.TcpPortManager)

	// Init assets
	err = assets.Load(cfg.AssetsDir)
	if err != nil {
		err = fmt.Errorf("Load assets error: %v", err)
		return
	}

	var (
		httpMuxOn  bool
		httpsMuxOn bool
	)
	if cfg.BindAddr == cfg.ProxyBindAddr {
		if cfg.BindPort == cfg.VhostHttpPort {
			httpMuxOn = true
		}
		if cfg.BindPort == cfg.VhostHttpsPort {
			httpsMuxOn = true
		}
	}

	// Listen for accepting connections from client.
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.BindPort))
	if err != nil {
		err = fmt.Errorf("Create server listener error, %v", err)
		return
	}

	svr.muxer = mux.NewMux(ln)
	go svr.muxer.Serve()
	ln = svr.muxer.DefaultListener()

	svr.listener = frpNet.WrapLogListener(ln)
	log.Info("frps tcp listen on %s:%d", cfg.BindAddr, cfg.BindPort)

	// Listen for accepting connections from client using kcp protocol.
	if cfg.KcpBindPort > 0 {
		svr.kcpListener, err = frpNet.ListenKcp(cfg.BindAddr, cfg.KcpBindPort)
		if err != nil {
			err = fmt.Errorf("Listen on kcp address udp [%s:%d] error: %v", cfg.BindAddr, cfg.KcpBindPort, err)
			return
		}
		log.Info("frps kcp listen on udp %s:%d", cfg.BindAddr, cfg.KcpBindPort)
	}

	// Listen for accepting connections from client using websocket protocol.
	websocketPrefix := []byte("GET " + frpNet.FrpWebsocketPath)
	websocketLn := svr.muxer.Listen(0, uint32(len(websocketPrefix)), func(data []byte) bool {
		return bytes.Equal(data, websocketPrefix)
	})
	svr.websocketListener = frpNet.NewWebsocketListener(websocketLn)

	// Create http vhost muxer.
	if cfg.VhostHttpPort > 0 {
		rp := vhost.NewHttpReverseProxy(vhost.HttpReverseProxyOptions{
			ResponseHeaderTimeoutS: cfg.VhostHttpTimeout,
		})
		svr.rc.HttpReverseProxy = rp

		address := fmt.Sprintf("%s:%d", cfg.ProxyBindAddr, cfg.VhostHttpPort)
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
				err = fmt.Errorf("Create vhost http listener error, %v", err)
				return
			}
		}
		go server.Serve(l)
		log.Info("http service listen on %s:%d", cfg.ProxyBindAddr, cfg.VhostHttpPort)
	}

	// Create https vhost muxer.
	if cfg.VhostHttpsPort > 0 {
		var l net.Listener
		if httpsMuxOn {
			l = svr.muxer.ListenHttps(1)
		} else {
			l, err = net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.ProxyBindAddr, cfg.VhostHttpsPort))
			if err != nil {
				err = fmt.Errorf("Create server listener error, %v", err)
				return
			}
		}

		svr.rc.VhostHttpsMuxer, err = vhost.NewHttpsMuxer(frpNet.WrapLogListener(l), 30*time.Second)
		if err != nil {
			err = fmt.Errorf("Create vhost httpsMuxer error, %v", err)
			return
		}
		log.Info("https service listen on %s:%d", cfg.ProxyBindAddr, cfg.VhostHttpsPort)
	}

	// Create nat hole controller.
	if cfg.BindUdpPort > 0 {
		var nc *nathole.NatHoleController
		addr := fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.BindUdpPort)
		nc, err = nathole.NewNatHoleController(addr)
		if err != nil {
			err = fmt.Errorf("Create nat hole controller error, %v", err)
			return
		}
		svr.rc.NatHoleController = nc
		log.Info("nat hole udp service listen on %s:%d", cfg.BindAddr, cfg.BindUdpPort)
	}

	var statsEnable bool
	// Create dashboard web server.
	if cfg.DashboardPort > 0 {
		err = svr.RunDashboardServer(cfg.DashboardAddr, cfg.DashboardPort)
		if err != nil {
			err = fmt.Errorf("Create dashboard web server error, %v", err)
			return
		}
		log.Info("Dashboard listen on %s:%d", cfg.DashboardAddr, cfg.DashboardPort)
		statsEnable = true
	}

	svr.statsCollector = stats.NewInternalCollector(statsEnable)
	return
}

func (svr *Service) Run() {
	if svr.rc.NatHoleController != nil {
		go svr.rc.NatHoleController.Run()
	}
	if g.GlbServerCfg.KcpBindPort > 0 {
		go svr.HandleListener(svr.kcpListener)
	}

	go svr.HandleListener(svr.websocketListener)

	svr.HandleListener(svr.listener)
}

func (svr *Service) HandleListener(l frpNet.Listener) {
	// Listen for incoming connections from client.
	for {
		c, err := l.Accept()
		if err != nil {
			log.Warn("Listener for incoming connections from client closed")
			return
		}

		// Start a new goroutine for dealing connections.
		go func(frpConn frpNet.Conn) {
			dealFn := func(conn frpNet.Conn) {
				var rawMsg msg.Message
				conn.SetReadDeadline(time.Now().Add(connReadTimeout))
				if rawMsg, err = msg.ReadMsg(conn); err != nil {
					log.Trace("Failed to read message: %v", err)
					conn.Close()
					return
				}
				conn.SetReadDeadline(time.Time{})

				switch m := rawMsg.(type) {
				case *msg.Login:
					err = svr.RegisterControl(conn, m)
					// If login failed, send error message there.
					// Otherwise send success message in control's work goroutine.
					if err != nil {
						conn.Warn("%v", err)
						msg.WriteMsg(conn, &msg.LoginResp{
							Version: version.Full(),
							Error:   err.Error(),
						})
						conn.Close()
					}
				case *msg.NewWorkConn:
					svr.RegisterWorkConn(conn, m)
				case *msg.NewVisitorConn:
					if err = svr.RegisterVisitorConn(conn, m); err != nil {
						conn.Warn("%v", err)
						msg.WriteMsg(conn, &msg.NewVisitorConnResp{
							ProxyName: m.ProxyName,
							Error:     err.Error(),
						})
						conn.Close()
					} else {
						msg.WriteMsg(conn, &msg.NewVisitorConnResp{
							ProxyName: m.ProxyName,
							Error:     "",
						})
					}
				default:
					log.Warn("Error message type for the new connection [%s]", conn.RemoteAddr().String())
					conn.Close()
				}
			}

			if g.GlbServerCfg.TcpMux {
				fmuxCfg := fmux.DefaultConfig()
				fmuxCfg.KeepAliveInterval = 20 * time.Second
				fmuxCfg.LogOutput = ioutil.Discard
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
					wrapConn := frpNet.WrapConn(stream)
					go dealFn(wrapConn)
				}
			} else {
				dealFn(frpConn)
			}
		}(c)
	}
}

func (svr *Service) RegisterControl(ctlConn frpNet.Conn, loginMsg *msg.Login) (err error) {
	ctlConn.Info("client login info: ip [%s] version [%s] hostname [%s] os [%s] arch [%s]",
		ctlConn.RemoteAddr().String(), loginMsg.Version, loginMsg.Hostname, loginMsg.Os, loginMsg.Arch)

	// Check client version.
	if ok, msg := version.Compat(loginMsg.Version); !ok {
		err = fmt.Errorf("%s", msg)
		return
	}

	// Check auth.
	if util.GetAuthKey(g.GlbServerCfg.Token, loginMsg.Timestamp) != loginMsg.PrivilegeKey {
		err = fmt.Errorf("authorization failed")
		return
	}

	// If client's RunId is empty, it's a new client, we just create a new controller.
	// Otherwise, we check if there is one controller has the same run id. If so, we release previous controller and start new one.
	if loginMsg.RunId == "" {
		loginMsg.RunId, err = util.RandId()
		if err != nil {
			return
		}
	}

	ctl := NewControl(svr.rc, svr.pxyManager, svr.statsCollector, ctlConn, loginMsg)

	if oldCtl := svr.ctlManager.Add(loginMsg.RunId, ctl); oldCtl != nil {
		oldCtl.allShutdown.WaitDone()
	}

	ctlConn.AddLogPrefix(loginMsg.RunId)
	ctl.Start()

	// for statistics
	svr.statsCollector.Mark(stats.TypeNewClient, &stats.NewClientPayload{})

	go func() {
		// block until control closed
		ctl.WaitClosed()
		svr.ctlManager.Del(loginMsg.RunId, ctl)
	}()
	return
}

// RegisterWorkConn register a new work connection to control and proxies need it.
func (svr *Service) RegisterWorkConn(workConn frpNet.Conn, newMsg *msg.NewWorkConn) {
	ctl, exist := svr.ctlManager.GetById(newMsg.RunId)
	if !exist {
		workConn.Warn("No client control found for run id [%s]", newMsg.RunId)
		return
	}
	ctl.RegisterWorkConn(workConn)
	return
}

func (svr *Service) RegisterVisitorConn(visitorConn frpNet.Conn, newMsg *msg.NewVisitorConn) error {
	return svr.rc.VisitorManager.NewConn(newMsg.ProxyName, visitorConn, newMsg.Timestamp, newMsg.SignKey,
		newMsg.UseEncryption, newMsg.UseCompression)
}

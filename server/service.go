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
	"fmt"
	"time"

	"github.com/fatedier/frp/assets"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/util"
	"github.com/fatedier/frp/utils/version"
	"github.com/fatedier/frp/utils/vhost"

	"github.com/xtaci/smux"
)

const (
	connReadTimeout time.Duration = 10 * time.Second
)

var ServerService *Service

// Server service.
type Service struct {
	// Accept connections from client.
	listener frpNet.Listener

	// For http proxies, route requests to different clients by hostname and other infomation.
	VhostHttpMuxer *vhost.HttpMuxer

	// For https proxies, route requests to different clients by hostname and other infomation.
	VhostHttpsMuxer *vhost.HttpsMuxer

	// Manage all controllers.
	ctlManager *ControlManager

	// Manage all proxies.
	pxyManager *ProxyManager
}

func NewService() (svr *Service, err error) {
	svr = &Service{
		ctlManager: NewControlManager(),
		pxyManager: NewProxyManager(),
	}

	// Init assets.
	err = assets.Load(config.ServerCommonCfg.AssetsDir)
	if err != nil {
		err = fmt.Errorf("Load assets error: %v", err)
		return
	}

	// Listen for accepting connections from client.
	svr.listener, err = frpNet.ListenTcp(config.ServerCommonCfg.BindAddr, config.ServerCommonCfg.BindPort)
	if err != nil {
		err = fmt.Errorf("Create server listener error, %v", err)
		return
	}

	// Create http vhost muxer.
	if config.ServerCommonCfg.VhostHttpPort != 0 {
		var l frpNet.Listener
		l, err = frpNet.ListenTcp(config.ServerCommonCfg.BindAddr, config.ServerCommonCfg.VhostHttpPort)
		if err != nil {
			err = fmt.Errorf("Create vhost http listener error, %v", err)
			return
		}
		svr.VhostHttpMuxer, err = vhost.NewHttpMuxer(l, 30*time.Second)
		if err != nil {
			err = fmt.Errorf("Create vhost httpMuxer error, %v", err)
			return
		}
	}

	// Create https vhost muxer.
	if config.ServerCommonCfg.VhostHttpsPort != 0 {
		var l frpNet.Listener
		l, err = frpNet.ListenTcp(config.ServerCommonCfg.BindAddr, config.ServerCommonCfg.VhostHttpsPort)
		if err != nil {
			err = fmt.Errorf("Create vhost https listener error, %v", err)
			return
		}
		svr.VhostHttpsMuxer, err = vhost.NewHttpsMuxer(l, 30*time.Second)
		if err != nil {
			err = fmt.Errorf("Create vhost httpsMuxer error, %v", err)
			return
		}
	}

	// Create dashboard web server.
	if config.ServerCommonCfg.DashboardPort != 0 {
		err = RunDashboardServer(config.ServerCommonCfg.BindAddr, config.ServerCommonCfg.DashboardPort)
		if err != nil {
			err = fmt.Errorf("Create dashboard web server error, %v", err)
			return
		}
	}
	return
}

func (svr *Service) Run() {
	// Listen for incoming connections from client.
	for {
		c, err := svr.listener.Accept()
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
					log.Warn("Failed to read message: %v", err)
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
				default:
					log.Warn("Error message type for the new connection [%s]", conn.RemoteAddr().String())
					conn.Close()
				}
			}

			if config.ServerCommonCfg.TcpMux {
				session, err := smux.Server(frpConn, nil)
				if err != nil {
					log.Warn("Failed to create mux connection: %v", err)
					frpConn.Close()
					return
				}

				for {
					stream, err := session.AcceptStream()
					if err != nil {
						log.Warn("Accept new mux stream error: %v", err)
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
	nowTime := time.Now().Unix()
	if config.ServerCommonCfg.AuthTimeout != 0 && nowTime-loginMsg.Timestamp > config.ServerCommonCfg.AuthTimeout {
		err = fmt.Errorf("authorization timeout")
		return
	}
	if util.GetAuthKey(config.ServerCommonCfg.PrivilegeToken, loginMsg.Timestamp) != loginMsg.PrivilegeKey {
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

	ctl := NewControl(svr, ctlConn, loginMsg)

	if oldCtl := svr.ctlManager.Add(loginMsg.RunId, ctl); oldCtl != nil {
		oldCtl.allShutdown.WaitDown()
	}

	ctlConn.AddLogPrefix(loginMsg.RunId)
	ctl.Start()

	// for statistics
	StatsNewClient()
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

func (svr *Service) RegisterProxy(name string, pxy Proxy) error {
	err := svr.pxyManager.Add(name, pxy)
	return err
}

func (svr *Service) DelProxy(name string) {
	svr.pxyManager.Del(name)
}

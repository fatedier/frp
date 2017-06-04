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
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/utils/crypto"
	"github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/util"
	"github.com/fatedier/frp/utils/version"
	"github.com/xtaci/smux"
)

const (
	connReadTimeout time.Duration = 10 * time.Second
)

type Control struct {
	// frpc service
	svr *Service

	// login message to server
	loginMsg *msg.Login

	// proxy configures
	pxyCfgs map[string]config.ProxyConf

	// proxies
	proxies map[string]Proxy

	// control connection
	conn net.Conn

	// tcp stream multiplexing, if enabled
	session *smux.Session

	// put a message in this channel to send it over control connection to server
	sendCh chan (msg.Message)

	// read from this channel to get the next message sent by server
	readCh chan (msg.Message)

	// run id got from server
	runId string

	// connection or other error happens , control will try to reconnect to server
	closed int32

	// goroutines can block by reading from this channel, it will be closed only in reader() when control connection is closed
	closedCh chan int

	// last time got the Pong message
	lastPong time.Time

	mu sync.RWMutex

	log.Logger
}

func NewControl(svr *Service, pxyCfgs map[string]config.ProxyConf) *Control {
	loginMsg := &msg.Login{
		Arch:      runtime.GOARCH,
		Os:        runtime.GOOS,
		PoolCount: config.ClientCommonCfg.PoolCount,
		User:      config.ClientCommonCfg.User,
		Version:   version.Full(),
	}
	return &Control{
		svr:      svr,
		loginMsg: loginMsg,
		pxyCfgs:  pxyCfgs,
		proxies:  make(map[string]Proxy),
		sendCh:   make(chan msg.Message, 10),
		readCh:   make(chan msg.Message, 10),
		closedCh: make(chan int),
		Logger:   log.NewPrefixLogger(""),
	}
}

// 1. login
// 2. start reader() writer() manager()
// 3. connection closed
// 4. In reader(): close closedCh and exit, controler() get it
// 5. In controler(): close readCh and sendCh, manager() and writer() will exit
// 6. In controler(): ini readCh, sendCh, closedCh
// 7. In controler(): start new reader(), writer(), manager()
// controler() will keep running
func (ctl *Control) Run() error {
	for {
		err := ctl.login()
		if err != nil {
			// if login_fail_exit is true, just exit this program
			// otherwise sleep a while and continues relogin to server
			if config.ClientCommonCfg.LoginFailExit {
				return err
			} else {
				ctl.Warn("login to server fail: %v", err)
				time.Sleep(30 * time.Second)
			}
		} else {
			break
		}
	}

	go ctl.controler()
	go ctl.manager()
	go ctl.writer()
	go ctl.reader()

	// send NewProxy message for all configured proxies
	for _, cfg := range ctl.pxyCfgs {
		var newProxyMsg msg.NewProxy
		cfg.UnMarshalToMsg(&newProxyMsg)
		ctl.sendCh <- &newProxyMsg
	}
	return nil
}

func (ctl *Control) NewWorkConn() {
	var (
		workConn net.Conn
		err      error
	)
	if config.ClientCommonCfg.TcpMux {
		stream, err := ctl.session.OpenStream()
		if err != nil {
			ctl.Warn("start new work connection error: %v", err)
			return
		}
		workConn = net.WrapConn(stream)

	} else {
		workConn, err = net.ConnectServerByHttpProxy(config.ClientCommonCfg.HttpProxy, config.ClientCommonCfg.Protocol,
			fmt.Sprintf("%s:%d", config.ClientCommonCfg.ServerAddr, config.ClientCommonCfg.ServerPort))
		if err != nil {
			ctl.Warn("start new work connection error: %v", err)
			return
		}
	}

	m := &msg.NewWorkConn{
		RunId: ctl.runId,
	}
	if err = msg.WriteMsg(workConn, m); err != nil {
		ctl.Warn("work connection write to server error: %v", err)
		workConn.Close()
		return
	}

	var startMsg msg.StartWorkConn
	if err = msg.ReadMsgInto(workConn, &startMsg); err != nil {
		ctl.Error("work connection closed, %v", err)
		workConn.Close()
		return
	}
	workConn.AddLogPrefix(startMsg.ProxyName)

	// dispatch this work connection to related proxy
	if pxy, ok := ctl.proxies[startMsg.ProxyName]; ok {
		workConn.Debug("start a new work connection, localAddr: %s remoteAddr: %s", workConn.LocalAddr().String(), workConn.RemoteAddr().String())
		go pxy.InWorkConn(workConn)
	} else {
		workConn.Close()
	}
}

func (ctl *Control) init() {
	ctl.sendCh = make(chan msg.Message, 10)
	ctl.readCh = make(chan msg.Message, 10)
	ctl.closedCh = make(chan int)
}

// login send a login message to server and wait for a loginResp message.
func (ctl *Control) login() (err error) {
	if ctl.conn != nil {
		ctl.conn.Close()
	}
	if ctl.session != nil {
		ctl.session.Close()
	}

	conn, err := net.ConnectServerByHttpProxy(config.ClientCommonCfg.HttpProxy, config.ClientCommonCfg.Protocol,
		fmt.Sprintf("%s:%d", config.ClientCommonCfg.ServerAddr, config.ClientCommonCfg.ServerPort))
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	if config.ClientCommonCfg.TcpMux {
		session, errRet := smux.Client(conn, nil)
		if errRet != nil {
			return errRet
		}
		stream, errRet := session.OpenStream()
		if errRet != nil {
			session.Close()
			return errRet
		}
		conn = net.WrapConn(stream)
		ctl.session = session
	}

	now := time.Now().Unix()
	ctl.loginMsg.PrivilegeKey = util.GetAuthKey(config.ClientCommonCfg.PrivilegeToken, now)
	ctl.loginMsg.Timestamp = now
	ctl.loginMsg.RunId = ctl.runId

	if err = msg.WriteMsg(conn, ctl.loginMsg); err != nil {
		return err
	}

	var loginRespMsg msg.LoginResp
	conn.SetReadDeadline(time.Now().Add(connReadTimeout))
	if err = msg.ReadMsgInto(conn, &loginRespMsg); err != nil {
		return err
	}
	conn.SetReadDeadline(time.Time{})

	if loginRespMsg.Error != "" {
		err = fmt.Errorf("%s", loginRespMsg.Error)
		ctl.Error("%s", loginRespMsg.Error)
		return err
	}

	ctl.conn = conn
	// update runId got from server
	ctl.runId = loginRespMsg.RunId
	ctl.ClearLogPrefix()
	ctl.AddLogPrefix(loginRespMsg.RunId)
	ctl.Info("login to server success, get run id [%s]", loginRespMsg.RunId)

	// login success, so we let closedCh available again
	ctl.closedCh = make(chan int)
	ctl.lastPong = time.Now()

	return nil
}

func (ctl *Control) reader() {
	defer func() {
		if err := recover(); err != nil {
			ctl.Error("panic error: %v", err)
		}
	}()
	defer close(ctl.closedCh)

	encReader := crypto.NewReader(ctl.conn, []byte(config.ClientCommonCfg.PrivilegeToken))
	for {
		if m, err := msg.ReadMsg(encReader); err != nil {
			if err == io.EOF {
				ctl.Debug("read from control connection EOF")
				return
			} else {
				ctl.Warn("read error: %v", err)
				return
			}
		} else {
			ctl.readCh <- m
		}
	}
}

func (ctl *Control) writer() {
	encWriter, err := crypto.NewWriter(ctl.conn, []byte(config.ClientCommonCfg.PrivilegeToken))
	if err != nil {
		ctl.conn.Error("crypto new writer error: %v", err)
		ctl.conn.Close()
		return
	}
	for {
		if m, ok := <-ctl.sendCh; !ok {
			ctl.Info("control writer is closing")
			return
		} else {
			if err := msg.WriteMsg(encWriter, m); err != nil {
				ctl.Warn("write message to control connection error: %v", err)
				return
			}
		}
	}
}

func (ctl *Control) manager() {
	defer func() {
		if err := recover(); err != nil {
			ctl.Error("panic error: %v", err)
		}
	}()

	hbSend := time.NewTicker(time.Duration(config.ClientCommonCfg.HeartBeatInterval) * time.Second)
	defer hbSend.Stop()
	hbCheck := time.NewTicker(time.Second)
	defer hbCheck.Stop()

	for {
		select {
		case <-hbSend.C:
			// send heartbeat to server
			ctl.Debug("send heartbeat to server")
			ctl.sendCh <- &msg.Ping{}
		case <-hbCheck.C:
			if time.Since(ctl.lastPong) > time.Duration(config.ClientCommonCfg.HeartBeatTimeout)*time.Second {
				ctl.Warn("heartbeat timeout")
				// let reader() stop
				ctl.conn.Close()
				return
			}
		case rawMsg, ok := <-ctl.readCh:
			if !ok {
				return
			}

			switch m := rawMsg.(type) {
			case *msg.ReqWorkConn:
				go ctl.NewWorkConn()
			case *msg.NewProxyResp:
				// Server will return NewProxyResp message to each NewProxy message.
				// Start a new proxy handler if no error got
				if m.Error != "" {
					ctl.Warn("[%s] start error: %s", m.ProxyName, m.Error)
					continue
				}
				cfg, ok := ctl.pxyCfgs[m.ProxyName]
				if !ok {
					// it will never go to this branch now
					ctl.Warn("[%s] no proxy conf found", m.ProxyName)
					continue
				}
				oldPxy, ok := ctl.proxies[m.ProxyName]
				if ok {
					oldPxy.Close()
				}
				pxy := NewProxy(ctl, cfg)
				if err := pxy.Run(); err != nil {
					ctl.Warn("[%s] proxy start running error: %v", m.ProxyName, err)
					continue
				}
				ctl.proxies[m.ProxyName] = pxy
				ctl.Info("[%s] start proxy success", m.ProxyName)
			case *msg.Pong:
				ctl.lastPong = time.Now()
				ctl.Debug("receive heartbeat from server")
			}
		}
	}
}

// control keep watching closedCh, start a new connection if previous control connection is closed
func (ctl *Control) controler() {
	var err error
	maxDelayTime := 30 * time.Second
	delayTime := time.Second

	checkInterval := 30 * time.Second
	checkProxyTicker := time.NewTicker(checkInterval)
	for {
		select {
		case <-checkProxyTicker.C:
			// Every 30 seconds, check which proxy registered failed and reregister it to server.
			for _, cfg := range ctl.pxyCfgs {
				if _, exist := ctl.proxies[cfg.GetName()]; !exist {
					ctl.Info("try to reregister proxy [%s]", cfg.GetName())
					var newProxyMsg msg.NewProxy
					cfg.UnMarshalToMsg(&newProxyMsg)
					ctl.sendCh <- &newProxyMsg
				}
			}
		case _, ok := <-ctl.closedCh:
			// we won't get any variable from this channel
			if !ok {
				// close related channels
				close(ctl.readCh)
				close(ctl.sendCh)

				for _, pxy := range ctl.proxies {
					pxy.Close()
				}
				time.Sleep(time.Second)

				// loop util reconnect to server success
				for {
					ctl.Info("try to reconnect to server...")
					err = ctl.login()
					if err != nil {
						ctl.Warn("reconnect to server error: %v", err)
						time.Sleep(delayTime)
						delayTime = delayTime * 2
						if delayTime > maxDelayTime {
							delayTime = maxDelayTime
						}
						continue
					}
					// reconnect success, init the delayTime
					delayTime = time.Second
					break
				}

				// init related channels and variables
				ctl.init()

				// previous work goroutines should be closed and start them here
				go ctl.manager()
				go ctl.writer()
				go ctl.reader()

				// send NewProxy message for all configured proxies
				for _, cfg := range ctl.pxyCfgs {
					var newProxyMsg msg.NewProxy
					cfg.UnMarshalToMsg(&newProxyMsg)
					ctl.sendCh <- &newProxyMsg
				}

				checkProxyTicker.Stop()
				checkProxyTicker = time.NewTicker(checkInterval)
			}
		}
	}
}

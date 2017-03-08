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
	"github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/util"
	"github.com/fatedier/frp/utils/version"
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
	err := ctl.login()
	if err != nil {
		return err
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
	workConn, err := net.ConnectTcpServerByHttpProxy(config.ClientCommonCfg.HttpProxy,
		fmt.Sprintf("%s:%d", config.ClientCommonCfg.ServerAddr, config.ClientCommonCfg.ServerPort))
	if err != nil {
		ctl.Warn("start new work connection error: %v", err)
		return
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
		ctl.Error("work connection closed and no response from server, %v", err)
		workConn.Close()
		return
	}
	workConn.AddLogPrefix(startMsg.ProxyName)

	// dispatch this work connection to related proxy
	if pxy, ok := ctl.proxies[startMsg.ProxyName]; ok {
		go pxy.InWorkConn(workConn)
		workConn.Info("start a new work connection")
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
	conn, err := net.ConnectTcpServerByHttpProxy(config.ClientCommonCfg.HttpProxy,
		fmt.Sprintf("%s:%d", config.ClientCommonCfg.ServerAddr, config.ClientCommonCfg.ServerPort))
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	ctl.loginMsg.PrivilegeKey = util.GetAuthKey(config.ClientCommonCfg.PrivilegeToken, now)
	ctl.loginMsg.Timestamp = now
	ctl.loginMsg.RunId = ctl.runId

	if err = msg.WriteMsg(conn, ctl.loginMsg); err != nil {
		return err
	}

	var loginRespMsg msg.LoginResp
	if err = msg.ReadMsgInto(conn, &loginRespMsg); err != nil {
		return err
	}

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

	for {
		if m, err := msg.ReadMsg(ctl.conn); err != nil {
			if err == io.EOF {
				ctl.Debug("read from control connection EOF")
				close(ctl.closedCh)
				return
			} else {
				ctl.Warn("read error: %v", err)
				continue
			}
		} else {
			ctl.readCh <- m
		}
	}
}

func (ctl *Control) writer() {
	for {
		if m, ok := <-ctl.sendCh; !ok {
			ctl.Info("control writer is closing")
			return
		} else {
			if err := msg.WriteMsg(ctl.conn, m); err != nil {
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
			ctl.sendCh <- &msg.Ping{}
		case <-hbCheck.C:
			if time.Since(ctl.lastPong) > time.Duration(config.ClientCommonCfg.HeartBeatTimeout)*time.Second {
				ctl.Warn("heartbeat timeout")
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
				oldPxy, ok := ctl.proxies[m.ProxyName]
				if ok {
					oldPxy.Close()
				}
				cfg, ok := ctl.pxyCfgs[m.ProxyName]
				if !ok {
					// it will never go to this branch
					ctl.Warn("[%s] no proxy conf found", m.ProxyName)
					continue
				}
				pxy := NewProxy(ctl, cfg)
				pxy.Run()
				ctl.proxies[m.ProxyName] = pxy
				ctl.Info("[%s] start proxy success", m.ProxyName)
			case *msg.Pong:
				ctl.lastPong = time.Now()
			}
		}
	}
}

// control keep watching closedCh, start a new connection if previous control connection is closed
func (ctl *Control) controler() {
	var err error
	maxDelayTime := 30 * time.Second
	delayTime := time.Second
	for {
		// we won't get any variable from this channel
		_, ok := <-ctl.closedCh
		if !ok {
			// close related channels
			close(ctl.readCh)
			close(ctl.sendCh)
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
		}
	}
}

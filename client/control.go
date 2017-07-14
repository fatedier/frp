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
	"github.com/fatedier/frp/utils/errors"
	"github.com/fatedier/frp/utils/log"
	frpNet "github.com/fatedier/frp/utils/net"
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

	// vistor configures
	vistorCfgs map[string]config.ProxyConf

	// vistors
	vistors map[string]Vistor

	// control connection
	conn frpNet.Conn

	// tcp stream multiplexing, if enabled
	session *smux.Session

	// put a message in this channel to send it over control connection to server
	sendCh chan (msg.Message)

	// read from this channel to get the next message sent by server
	readCh chan (msg.Message)

	// run id got from server
	runId string

	// if we call close() in control, do not reconnect to server
	exit bool

	// goroutines can block by reading from this channel, it will be closed only in reader() when control connection is closed
	closedCh chan int

	// last time got the Pong message
	lastPong time.Time

	mu sync.RWMutex

	log.Logger
}

func NewControl(svr *Service, pxyCfgs map[string]config.ProxyConf, vistorCfgs map[string]config.ProxyConf) *Control {
	loginMsg := &msg.Login{
		Arch:      runtime.GOARCH,
		Os:        runtime.GOOS,
		PoolCount: config.ClientCommonCfg.PoolCount,
		User:      config.ClientCommonCfg.User,
		Version:   version.Full(),
	}
	return &Control{
		svr:        svr,
		loginMsg:   loginMsg,
		pxyCfgs:    pxyCfgs,
		vistorCfgs: vistorCfgs,
		proxies:    make(map[string]Proxy),
		vistors:    make(map[string]Vistor),
		sendCh:     make(chan msg.Message, 10),
		readCh:     make(chan msg.Message, 10),
		closedCh:   make(chan int),
		Logger:     log.NewPrefixLogger(""),
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
func (ctl *Control) Run() (err error) {
	for {
		err = ctl.login()
		if err != nil {
			ctl.Warn("login to server failed: %v", err)

			// if login_fail_exit is true, just exit this program
			// otherwise sleep a while and continues relogin to server
			if config.ClientCommonCfg.LoginFailExit {
				return
			} else {
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

	// start all local vistors
	for _, cfg := range ctl.vistorCfgs {
		vistor := NewVistor(ctl, cfg)
		err = vistor.Run()
		if err != nil {
			vistor.Warn("start error: %v", err)
			continue
		}
		ctl.vistors[cfg.GetName()] = vistor
		vistor.Info("start vistor success")
	}

	// send NewProxy message for all configured proxies
	for _, cfg := range ctl.pxyCfgs {
		var newProxyMsg msg.NewProxy
		cfg.UnMarshalToMsg(&newProxyMsg)
		ctl.sendCh <- &newProxyMsg
	}
	return nil
}

func (ctl *Control) NewWorkConn() {
	workConn, err := ctl.connectServer()
	if err != nil {
		return
	}

	m := &msg.NewWorkConn{
		RunId: ctl.getRunId(),
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
	pxy, ok := ctl.getProxy(startMsg.ProxyName)
	if ok {
		workConn.Debug("start a new work connection, localAddr: %s remoteAddr: %s", workConn.LocalAddr().String(), workConn.RemoteAddr().String())
		go pxy.InWorkConn(workConn)
	} else {
		workConn.Close()
	}
}

func (ctl *Control) Close() error {
	ctl.mu.Lock()
	ctl.exit = true
	err := errors.PanicToError(func() {
		for name, _ := range ctl.proxies {
			ctl.sendCh <- &msg.CloseProxy{
				ProxyName: name,
			}
		}
	})
	ctl.mu.Unlock()
	return err
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

	conn, err := frpNet.ConnectServerByHttpProxy(config.ClientCommonCfg.HttpProxy, config.ClientCommonCfg.Protocol,
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
		conn = frpNet.WrapConn(stream)
		ctl.session = session
	}

	now := time.Now().Unix()
	ctl.loginMsg.PrivilegeKey = util.GetAuthKey(config.ClientCommonCfg.PrivilegeToken, now)
	ctl.loginMsg.Timestamp = now
	ctl.loginMsg.RunId = ctl.getRunId()

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
	ctl.setRunId(loginRespMsg.RunId)
	ctl.ClearLogPrefix()
	ctl.AddLogPrefix(loginRespMsg.RunId)
	ctl.Info("login to server success, get run id [%s]", loginRespMsg.RunId)

	// login success, so we let closedCh available again
	ctl.closedCh = make(chan int)
	ctl.lastPong = time.Now()

	return nil
}

func (ctl *Control) connectServer() (conn frpNet.Conn, err error) {
	if config.ClientCommonCfg.TcpMux {
		stream, errRet := ctl.session.OpenStream()
		if errRet != nil {
			err = errRet
			ctl.Warn("start new connection to server error: %v", err)
			return
		}
		conn = frpNet.WrapConn(stream)

	} else {
		conn, err = frpNet.ConnectServerByHttpProxy(config.ClientCommonCfg.HttpProxy, config.ClientCommonCfg.Protocol,
			fmt.Sprintf("%s:%d", config.ClientCommonCfg.ServerAddr, config.ClientCommonCfg.ServerPort))
		if err != nil {
			ctl.Warn("start new connection to server error: %v", err)
			return
		}
	}
	return
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

// manager handles all channel events and do corresponding process
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
				cfg, ok := ctl.getProxyConf(m.ProxyName)
				if !ok {
					// it will never go to this branch now
					ctl.Warn("[%s] no proxy conf found", m.ProxyName)
					continue
				}

				oldPxy, ok := ctl.getProxy(m.ProxyName)
				if ok {
					oldPxy.Close()
				}
				pxy := NewProxy(ctl, cfg)
				if err := pxy.Run(); err != nil {
					ctl.Warn("[%s] proxy start running error: %v", m.ProxyName, err)
					ctl.sendCh <- &msg.CloseProxy{
						ProxyName: m.ProxyName,
					}
					continue
				}
				ctl.addProxy(m.ProxyName, pxy)
				ctl.Info("[%s] start proxy success", m.ProxyName)
			case *msg.Pong:
				ctl.lastPong = time.Now()
				ctl.Debug("receive heartbeat from server")
			}
		}
	}
}

// controler keep watching closedCh, start a new connection if previous control connection is closed.
// If controler is notified by closedCh, reader and writer and manager will exit, then recall these functions.
func (ctl *Control) controler() {
	var err error
	maxDelayTime := 30 * time.Second
	delayTime := time.Second

	checkInterval := 10 * time.Second
	checkProxyTicker := time.NewTicker(checkInterval)
	for {
		select {
		case <-checkProxyTicker.C:
			// Every 10 seconds, check which proxy registered failed and reregister it to server.
			ctl.mu.RLock()
			for _, cfg := range ctl.pxyCfgs {
				if _, exist := ctl.proxies[cfg.GetName()]; !exist {
					ctl.Info("try to register proxy [%s]", cfg.GetName())
					var newProxyMsg msg.NewProxy
					cfg.UnMarshalToMsg(&newProxyMsg)
					ctl.sendCh <- &newProxyMsg
				}
			}

			for _, cfg := range ctl.vistorCfgs {
				if _, exist := ctl.vistors[cfg.GetName()]; !exist {
					ctl.Info("try to start vistor [%s]", cfg.GetName())
					vistor := NewVistor(ctl, cfg)
					err = vistor.Run()
					if err != nil {
						vistor.Warn("start error: %v", err)
						continue
					}
					ctl.vistors[cfg.GetName()] = vistor
					vistor.Info("start vistor success")
				}
			}
			ctl.mu.RUnlock()
		case _, ok := <-ctl.closedCh:
			// we won't get any variable from this channel
			if !ok {
				// close related channels
				close(ctl.readCh)
				close(ctl.sendCh)

				for _, pxy := range ctl.proxies {
					pxy.Close()
				}
				// if ctl.exit is true, just exit
				ctl.mu.RLock()
				exit := ctl.exit
				ctl.mu.RUnlock()
				if exit {
					return
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
				ctl.mu.RLock()
				for _, cfg := range ctl.pxyCfgs {
					var newProxyMsg msg.NewProxy
					cfg.UnMarshalToMsg(&newProxyMsg)
					ctl.sendCh <- &newProxyMsg
				}
				ctl.mu.RUnlock()

				checkProxyTicker.Stop()
				checkProxyTicker = time.NewTicker(checkInterval)
			}
		}
	}
}

func (ctl *Control) setRunId(runId string) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()
	ctl.runId = runId
}

func (ctl *Control) getRunId() string {
	ctl.mu.RLock()
	defer ctl.mu.RUnlock()
	return ctl.runId
}

func (ctl *Control) getProxy(name string) (pxy Proxy, ok bool) {
	ctl.mu.RLock()
	defer ctl.mu.RUnlock()
	pxy, ok = ctl.proxies[name]
	return
}

func (ctl *Control) addProxy(name string, pxy Proxy) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()
	ctl.proxies[name] = pxy
}

func (ctl *Control) getProxyConf(name string) (conf config.ProxyConf, ok bool) {
	ctl.mu.RLock()
	defer ctl.mu.RUnlock()
	conf, ok = ctl.pxyCfgs[name]
	return
}

func (ctl *Control) reloadConf(pxyCfgs map[string]config.ProxyConf, vistorCfgs map[string]config.ProxyConf) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	removedPxyNames := make([]string, 0)
	for name, oldCfg := range ctl.pxyCfgs {
		del := false
		cfg, ok := pxyCfgs[name]
		if !ok {
			del = true
		} else {
			if !oldCfg.Compare(cfg) {
				del = true
			}
		}

		if del {
			removedPxyNames = append(removedPxyNames, name)
			delete(ctl.pxyCfgs, name)
			if pxy, ok := ctl.proxies[name]; ok {
				pxy.Close()
			}
			delete(ctl.proxies, name)
			ctl.sendCh <- &msg.CloseProxy{
				ProxyName: name,
			}
		}
	}
	ctl.Info("proxy removed: %v", removedPxyNames)

	addedPxyNames := make([]string, 0)
	for name, cfg := range pxyCfgs {
		if _, ok := ctl.pxyCfgs[name]; !ok {
			ctl.pxyCfgs[name] = cfg
			addedPxyNames = append(addedPxyNames, name)
		}
	}
	ctl.Info("proxy added: %v", addedPxyNames)

	removedVistorName := make([]string, 0)
	for name, oldVistorCfg := range ctl.vistorCfgs {
		del := false
		cfg, ok := vistorCfgs[name]
		if !ok {
			del = true
		} else {
			if !oldVistorCfg.Compare(cfg) {
				del = true
			}
		}

		if del {
			removedVistorName = append(removedVistorName, name)
			delete(ctl.vistorCfgs, name)
			if vistor, ok := ctl.vistors[name]; ok {
				vistor.Close()
			}
			delete(ctl.vistors, name)
		}
	}
	ctl.Info("vistor removed: %v", removedVistorName)

	addedVistorName := make([]string, 0)
	for name, vistorCfg := range vistorCfgs {
		if _, ok := ctl.vistorCfgs[name]; !ok {
			ctl.vistorCfgs[name] = vistorCfg
			addedVistorName = append(addedVistorName, name)
		}
	}
	ctl.Info("vistor added: %v", addedVistorName)
}

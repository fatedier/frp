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
	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/shutdown"
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

	// login message to server, only used
	loginMsg *msg.Login

	pm *ProxyManager

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

	readerShutdown     *shutdown.Shutdown
	writerShutdown     *shutdown.Shutdown
	msgHandlerShutdown *shutdown.Shutdown

	mu sync.RWMutex

	log.Logger
}

func NewControl(svr *Service, pxyCfgs map[string]config.ProxyConf, visitorCfgs map[string]config.ProxyConf) *Control {
	loginMsg := &msg.Login{
		Arch:      runtime.GOARCH,
		Os:        runtime.GOOS,
		PoolCount: config.ClientCommonCfg.PoolCount,
		User:      config.ClientCommonCfg.User,
		Version:   version.Full(),
	}
	ctl := &Control{
		svr:                svr,
		loginMsg:           loginMsg,
		sendCh:             make(chan msg.Message, 10),
		readCh:             make(chan msg.Message, 10),
		closedCh:           make(chan int),
		readerShutdown:     shutdown.New(),
		writerShutdown:     shutdown.New(),
		msgHandlerShutdown: shutdown.New(),
		Logger:             log.NewPrefixLogger(""),
	}
	ctl.pm = NewProxyManager(ctl, ctl.sendCh, "")
	ctl.pm.Reload(pxyCfgs, visitorCfgs)
	return ctl
}

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
				time.Sleep(10 * time.Second)
			}
		} else {
			break
		}
	}

	go ctl.worker()

	// start all local visitors and send NewProxy message for all configured proxies
	ctl.pm.Reset(ctl.sendCh, ctl.runId)
	ctl.pm.CheckAndStartProxy()
	return nil
}

func (ctl *Control) HandleReqWorkConn(inMsg *msg.ReqWorkConn) {
	workConn, err := ctl.connectServer()
	if err != nil {
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
		ctl.Error("work connection closed, %v", err)
		workConn.Close()
		return
	}
	workConn.AddLogPrefix(startMsg.ProxyName)

	// dispatch this work connection to related proxy
	ctl.pm.HandleWorkConn(startMsg.ProxyName, workConn)
}

func (ctl *Control) HandleNewProxyResp(inMsg *msg.NewProxyResp) {
	// Server will return NewProxyResp message to each NewProxy message.
	// Start a new proxy handler if no error got
	err := ctl.pm.StartProxy(inMsg.ProxyName, inMsg.RemoteAddr, inMsg.Error)
	if err != nil {
		ctl.Warn("[%s] start error: %v", inMsg.ProxyName, err)
	} else {
		ctl.Info("[%s] start proxy success", inMsg.ProxyName)
	}
}

func (ctl *Control) Close() error {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()
	ctl.exit = true
	ctl.pm.CloseProxies()
	return nil
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
	config.ClientCommonCfg.ServerUdpPort = loginRespMsg.ServerUdpPort
	ctl.ClearLogPrefix()
	ctl.AddLogPrefix(loginRespMsg.RunId)
	ctl.Info("login to server success, get run id [%s], server udp port [%d]", loginRespMsg.RunId, loginRespMsg.ServerUdpPort)
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

// reader read all messages from frps and send to readCh
func (ctl *Control) reader() {
	defer func() {
		if err := recover(); err != nil {
			ctl.Error("panic error: %v", err)
		}
	}()
	defer ctl.readerShutdown.Done()
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

// writer writes messages got from sendCh to frps
func (ctl *Control) writer() {
	defer ctl.writerShutdown.Done()
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

// msgHandler handles all channel events and do corresponding operations.
func (ctl *Control) msgHandler() {
	defer func() {
		if err := recover(); err != nil {
			ctl.Error("panic error: %v", err)
		}
	}()
	defer ctl.msgHandlerShutdown.Done()

	hbSend := time.NewTicker(time.Duration(config.ClientCommonCfg.HeartBeatInterval) * time.Second)
	defer hbSend.Stop()
	hbCheck := time.NewTicker(time.Second)
	defer hbCheck.Stop()

	ctl.lastPong = time.Now()

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
				go ctl.HandleReqWorkConn(m)
			case *msg.NewProxyResp:
				ctl.HandleNewProxyResp(m)
			case *msg.Pong:
				ctl.lastPong = time.Now()
				ctl.Debug("receive heartbeat from server")
			}
		}
	}
}

// controler keep watching closedCh, start a new connection if previous control connection is closed.
// If controler is notified by closedCh, reader and writer and handler will exit, then recall these functions.
func (ctl *Control) worker() {
	go ctl.msgHandler()
	go ctl.writer()
	go ctl.reader()

	var err error
	maxDelayTime := 20 * time.Second
	delayTime := time.Second

	checkInterval := 10 * time.Second
	checkProxyTicker := time.NewTicker(checkInterval)
	for {
		select {
		case <-checkProxyTicker.C:
			// every 10 seconds, check which proxy registered failed and reregister it to server
			ctl.pm.CheckAndStartProxy()
		case _, ok := <-ctl.closedCh:
			// we won't get any variable from this channel
			if !ok {
				// close related channels and wait until other goroutines done
				close(ctl.readCh)
				ctl.readerShutdown.WaitDone()
				ctl.msgHandlerShutdown.WaitDone()

				close(ctl.sendCh)
				ctl.writerShutdown.WaitDone()

				ctl.pm.CloseProxies()
				// if ctl.exit is true, just exit
				ctl.mu.RLock()
				exit := ctl.exit
				ctl.mu.RUnlock()
				if exit {
					return
				}

				// loop util reconnecting to server success
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
					// reconnect success, init delayTime
					delayTime = time.Second
					break
				}

				// init related channels and variables
				ctl.sendCh = make(chan msg.Message, 10)
				ctl.readCh = make(chan msg.Message, 10)
				ctl.closedCh = make(chan int)
				ctl.readerShutdown = shutdown.New()
				ctl.writerShutdown = shutdown.New()
				ctl.msgHandlerShutdown = shutdown.New()
				ctl.pm.Reset(ctl.sendCh, ctl.runId)

				// previous work goroutines should be closed and start them here
				go ctl.msgHandler()
				go ctl.writer()
				go ctl.reader()

				// start all configured proxies
				ctl.pm.CheckAndStartProxy()

				checkProxyTicker.Stop()
				checkProxyTicker = time.NewTicker(checkInterval)
			}
		}
	}
}

func (ctl *Control) reloadConf(pxyCfgs map[string]config.ProxyConf, visitorCfgs map[string]config.ProxyConf) error {
	err := ctl.pm.Reload(pxyCfgs, visitorCfgs)
	return err
}

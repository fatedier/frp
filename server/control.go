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
	"io"
	"sync"
	"time"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/consts"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/utils/crypto"
	"github.com/fatedier/frp/utils/errors"
	"github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/shutdown"
	"github.com/fatedier/frp/utils/version"
)

type Control struct {
	// frps service
	svr *Service

	// login message
	loginMsg *msg.Login

	// control connection
	conn net.Conn

	// put a message in this channel to send it over control connection to client
	sendCh chan (msg.Message)

	// read from this channel to get the next message sent by client
	readCh chan (msg.Message)

	// work connections
	workConnCh chan net.Conn

	// proxies in one client
	proxies map[string]Proxy

	// pool count
	poolCount int

	// last time got the Ping message
	lastPing time.Time

	// A new run id will be generated when a new client login.
	// If run id got from login message has same run id, it means it's the same client, so we can
	// replace old controller instantly.
	runId string

	// control status
	status string

	readerShutdown  *shutdown.Shutdown
	writerShutdown  *shutdown.Shutdown
	managerShutdown *shutdown.Shutdown
	allShutdown     *shutdown.Shutdown

	mu sync.RWMutex
}

func NewControl(svr *Service, ctlConn net.Conn, loginMsg *msg.Login) *Control {
	return &Control{
		svr:             svr,
		conn:            ctlConn,
		loginMsg:        loginMsg,
		sendCh:          make(chan msg.Message, 10),
		readCh:          make(chan msg.Message, 10),
		workConnCh:      make(chan net.Conn, loginMsg.PoolCount+10),
		proxies:         make(map[string]Proxy),
		poolCount:       loginMsg.PoolCount,
		lastPing:        time.Now(),
		runId:           loginMsg.RunId,
		status:          consts.Working,
		readerShutdown:  shutdown.New(),
		writerShutdown:  shutdown.New(),
		managerShutdown: shutdown.New(),
		allShutdown:     shutdown.New(),
	}
}

// Start send a login success message to client and start working.
func (ctl *Control) Start() {
	loginRespMsg := &msg.LoginResp{
		Version:       version.Full(),
		RunId:         ctl.runId,
		ServerUdpPort: config.ServerCommonCfg.BindUdpPort,
		Error:         "",
	}
	msg.WriteMsg(ctl.conn, loginRespMsg)

	go ctl.writer()
	for i := 0; i < ctl.poolCount; i++ {
		ctl.sendCh <- &msg.ReqWorkConn{}
	}

	go ctl.manager()
	go ctl.reader()
	go ctl.stoper()
}

func (ctl *Control) RegisterWorkConn(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			ctl.conn.Error("panic error: %v", err)
		}
	}()

	select {
	case ctl.workConnCh <- conn:
		ctl.conn.Debug("new work connection registered")
	default:
		ctl.conn.Debug("work connection pool is full, discarding")
		conn.Close()
	}
}

// When frps get one user connection, we get one work connection from the pool and return it.
// If no workConn available in the pool, send message to frpc to get one or more
// and wait until it is available.
// return an error if wait timeout
func (ctl *Control) GetWorkConn() (workConn net.Conn, err error) {
	defer func() {
		if err := recover(); err != nil {
			ctl.conn.Error("panic error: %v", err)
		}
	}()

	var ok bool
	// get a work connection from the pool
	select {
	case workConn, ok = <-ctl.workConnCh:
		if !ok {
			err = errors.ErrCtlClosed
			return
		}
		ctl.conn.Debug("get work connection from pool")
	default:
		// no work connections available in the poll, send message to frpc to get more
		err = errors.PanicToError(func() {
			ctl.sendCh <- &msg.ReqWorkConn{}
		})
		if err != nil {
			ctl.conn.Error("%v", err)
			return
		}

		select {
		case workConn, ok = <-ctl.workConnCh:
			if !ok {
				err = errors.ErrCtlClosed
				ctl.conn.Warn("no work connections avaiable, %v", err)
				return
			}

		case <-time.After(time.Duration(config.ServerCommonCfg.UserConnTimeout) * time.Second):
			err = fmt.Errorf("timeout trying to get work connection")
			ctl.conn.Warn("%v", err)
			return
		}
	}

	// When we get a work connection from pool, replace it with a new one.
	errors.PanicToError(func() {
		ctl.sendCh <- &msg.ReqWorkConn{}
	})
	return
}

func (ctl *Control) Replaced(newCtl *Control) {
	ctl.conn.Info("Replaced by client [%s]", newCtl.runId)
	ctl.runId = ""
	ctl.allShutdown.Start()
}

func (ctl *Control) writer() {
	defer func() {
		if err := recover(); err != nil {
			ctl.conn.Error("panic error: %v", err)
		}
	}()

	defer ctl.allShutdown.Start()
	defer ctl.writerShutdown.Done()

	encWriter, err := crypto.NewWriter(ctl.conn, []byte(config.ServerCommonCfg.PrivilegeToken))
	if err != nil {
		ctl.conn.Error("crypto new writer error: %v", err)
		ctl.allShutdown.Start()
		return
	}
	for {
		if m, ok := <-ctl.sendCh; !ok {
			ctl.conn.Info("control writer is closing")
			return
		} else {
			if err := msg.WriteMsg(encWriter, m); err != nil {
				ctl.conn.Warn("write message to control connection error: %v", err)
				return
			}
		}
	}
}

func (ctl *Control) reader() {
	defer func() {
		if err := recover(); err != nil {
			ctl.conn.Error("panic error: %v", err)
		}
	}()

	defer ctl.allShutdown.Start()
	defer ctl.readerShutdown.Done()

	encReader := crypto.NewReader(ctl.conn, []byte(config.ServerCommonCfg.PrivilegeToken))
	for {
		if m, err := msg.ReadMsg(encReader); err != nil {
			if err == io.EOF {
				ctl.conn.Debug("control connection closed")
				return
			} else {
				ctl.conn.Warn("read error: %v", err)
				return
			}
		} else {
			ctl.readCh <- m
		}
	}
}

func (ctl *Control) stoper() {
	defer func() {
		if err := recover(); err != nil {
			ctl.conn.Error("panic error: %v", err)
		}
	}()

	ctl.allShutdown.WaitStart()

	close(ctl.readCh)
	ctl.managerShutdown.WaitDone()

	close(ctl.sendCh)
	ctl.writerShutdown.WaitDone()

	ctl.conn.Close()
	ctl.readerShutdown.WaitDone()

	close(ctl.workConnCh)
	for workConn := range ctl.workConnCh {
		workConn.Close()
	}

	ctl.mu.Lock()
	defer ctl.mu.Unlock()
	for _, pxy := range ctl.proxies {
		pxy.Close()
		ctl.svr.DelProxy(pxy.GetName())
		StatsCloseProxy(pxy.GetName(), pxy.GetConf().GetBaseInfo().ProxyType)
	}

	ctl.allShutdown.Done()
	ctl.conn.Info("client exit success")

	StatsCloseClient()
}

func (ctl *Control) manager() {
	defer func() {
		if err := recover(); err != nil {
			ctl.conn.Error("panic error: %v", err)
		}
	}()

	defer ctl.allShutdown.Start()
	defer ctl.managerShutdown.Done()

	heartbeat := time.NewTicker(time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-heartbeat.C:
			if time.Since(ctl.lastPing) > time.Duration(config.ServerCommonCfg.HeartBeatTimeout)*time.Second {
				ctl.conn.Warn("heartbeat timeout")
				ctl.allShutdown.Start()
			}
		case rawMsg, ok := <-ctl.readCh:
			if !ok {
				return
			}

			switch m := rawMsg.(type) {
			case *msg.NewProxy:
				// register proxy in this control
				remoteAddr, err := ctl.RegisterProxy(m)
				resp := &msg.NewProxyResp{
					ProxyName: m.ProxyName,
				}
				if err != nil {
					resp.Error = err.Error()
					ctl.conn.Warn("new proxy [%s] error: %v", m.ProxyName, err)
				} else {
					resp.RemoteAddr = remoteAddr
					ctl.conn.Info("new proxy [%s] success", m.ProxyName)
					StatsNewProxy(m.ProxyName, m.ProxyType)
				}
				ctl.sendCh <- resp
			case *msg.CloseProxy:
				ctl.CloseProxy(m)
				ctl.conn.Info("close proxy [%s] success", m.ProxyName)
			case *msg.Ping:
				ctl.lastPing = time.Now()
				ctl.conn.Debug("receive heartbeat")
				ctl.sendCh <- &msg.Pong{}
			}
		}
	}
}

func (ctl *Control) RegisterProxy(pxyMsg *msg.NewProxy) (remoteAddr string, err error) {
	var pxyConf config.ProxyConf
	// Load configures from NewProxy message and check.
	pxyConf, err = config.NewProxyConf(pxyMsg)
	if err != nil {
		return
	}

	// NewProxy will return a interface Proxy.
	// In fact it create different proxies by different proxy type, we just call run() here.
	pxy, err := NewProxy(ctl, pxyConf)
	if err != nil {
		return remoteAddr, err
	}

	remoteAddr, err = pxy.Run()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			pxy.Close()
		}
	}()

	err = ctl.svr.RegisterProxy(pxyMsg.ProxyName, pxy)
	if err != nil {
		return
	}

	ctl.mu.Lock()
	ctl.proxies[pxy.GetName()] = pxy
	ctl.mu.Unlock()
	return
}

func (ctl *Control) CloseProxy(closeMsg *msg.CloseProxy) (err error) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	pxy, ok := ctl.proxies[closeMsg.ProxyName]
	if !ok {
		return
	}

	pxy.Close()
	ctl.svr.DelProxy(pxy.GetName())
	delete(ctl.proxies, closeMsg.ProxyName)
	StatsCloseProxy(pxy.GetName(), pxy.GetConf().GetBaseInfo().ProxyType)
	return
}

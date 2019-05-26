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
	"runtime/debug"
	"sync"
	"time"

	"github.com/fatedier/frp/g"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/consts"
	frpErr "github.com/fatedier/frp/models/errors"
	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/server/controller"
	"github.com/fatedier/frp/server/proxy"
	"github.com/fatedier/frp/server/stats"
	"github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/version"

	"github.com/fatedier/golib/control/shutdown"
	"github.com/fatedier/golib/crypto"
	"github.com/fatedier/golib/errors"
)

type ControlManager struct {
	// controls indexed by run id
	ctlsByRunId map[string]*Control

	mu sync.RWMutex
}

func NewControlManager() *ControlManager {
	return &ControlManager{
		ctlsByRunId: make(map[string]*Control),
	}
}

func (cm *ControlManager) Add(runId string, ctl *Control) (oldCtl *Control) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	oldCtl, ok := cm.ctlsByRunId[runId]
	if ok {
		oldCtl.Replaced(ctl)
	}
	cm.ctlsByRunId[runId] = ctl
	return
}

// we should make sure if it's the same control to prevent delete a new one
func (cm *ControlManager) Del(runId string, ctl *Control) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if c, ok := cm.ctlsByRunId[runId]; ok && c == ctl {
		delete(cm.ctlsByRunId, runId)
	}
}

func (cm *ControlManager) GetById(runId string) (ctl *Control, ok bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	ctl, ok = cm.ctlsByRunId[runId]
	return
}

type Control struct {
	// all resource managers and controllers
	rc *controller.ResourceController

	// proxy manager
	pxyManager *proxy.ProxyManager

	// stats collector to store stats info of clients and proxies
	statsCollector stats.Collector

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
	proxies map[string]proxy.Proxy

	// pool count
	poolCount int

	// ports used, for limitations
	portsUsedNum int

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

func NewControl(rc *controller.ResourceController, pxyManager *proxy.ProxyManager,
	statsCollector stats.Collector, ctlConn net.Conn, loginMsg *msg.Login) *Control {

	return &Control{
		rc:              rc,
		pxyManager:      pxyManager,
		statsCollector:  statsCollector,
		conn:            ctlConn,
		loginMsg:        loginMsg,
		sendCh:          make(chan msg.Message, 10),
		readCh:          make(chan msg.Message, 10),
		workConnCh:      make(chan net.Conn, loginMsg.PoolCount+10),
		proxies:         make(map[string]proxy.Proxy),
		poolCount:       loginMsg.PoolCount,
		portsUsedNum:    0,
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
		ServerUdpPort: g.GlbServerCfg.BindUdpPort,
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
			ctl.conn.Error(string(debug.Stack()))
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
			ctl.conn.Error(string(debug.Stack()))
		}
	}()

	var ok bool
	// get a work connection from the pool
	select {
	case workConn, ok = <-ctl.workConnCh:
		if !ok {
			err = frpErr.ErrCtlClosed
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
				err = frpErr.ErrCtlClosed
				ctl.conn.Warn("no work connections avaiable, %v", err)
				return
			}

		case <-time.After(time.Duration(g.GlbServerCfg.UserConnTimeout) * time.Second):
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
			ctl.conn.Error(string(debug.Stack()))
		}
	}()

	defer ctl.allShutdown.Start()
	defer ctl.writerShutdown.Done()

	encWriter, err := crypto.NewWriter(ctl.conn, []byte(g.GlbServerCfg.Token))
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
			ctl.conn.Error(string(debug.Stack()))
		}
	}()

	defer ctl.allShutdown.Start()
	defer ctl.readerShutdown.Done()

	encReader := crypto.NewReader(ctl.conn, []byte(g.GlbServerCfg.Token))
	for {
		if m, err := msg.ReadMsg(encReader); err != nil {
			if err == io.EOF {
				ctl.conn.Debug("control connection closed")
				return
			} else {
				ctl.conn.Warn("read error: %v", err)
				ctl.conn.Close()
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
			ctl.conn.Error(string(debug.Stack()))
		}
	}()

	ctl.allShutdown.WaitStart()

	close(ctl.readCh)
	ctl.managerShutdown.WaitDone()

	close(ctl.sendCh)
	ctl.writerShutdown.WaitDone()

	ctl.conn.Close()
	ctl.readerShutdown.WaitDone()

	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	close(ctl.workConnCh)
	for workConn := range ctl.workConnCh {
		workConn.Close()
	}

	for _, pxy := range ctl.proxies {
		pxy.Close()
		ctl.pxyManager.Del(pxy.GetName())
		ctl.statsCollector.Mark(stats.TypeCloseProxy, &stats.CloseProxyPayload{
			Name:      pxy.GetName(),
			ProxyType: pxy.GetConf().GetBaseInfo().ProxyType,
		})
	}

	ctl.allShutdown.Done()
	ctl.conn.Info("client exit success")

	ctl.statsCollector.Mark(stats.TypeCloseClient, &stats.CloseClientPayload{})
}

// block until Control closed
func (ctl *Control) WaitClosed() {
	ctl.allShutdown.WaitDone()
}

func (ctl *Control) manager() {
	defer func() {
		if err := recover(); err != nil {
			ctl.conn.Error("panic error: %v", err)
			ctl.conn.Error(string(debug.Stack()))
		}
	}()

	defer ctl.allShutdown.Start()
	defer ctl.managerShutdown.Done()

	heartbeat := time.NewTicker(time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-heartbeat.C:
			if time.Since(ctl.lastPing) > time.Duration(g.GlbServerCfg.HeartBeatTimeout)*time.Second {
				ctl.conn.Warn("heartbeat timeout")
				return
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
					ctl.statsCollector.Mark(stats.TypeNewProxy, &stats.NewProxyPayload{
						Name:      m.ProxyName,
						ProxyType: m.ProxyType,
					})
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
	pxyConf, err = config.NewProxyConfFromMsg(pxyMsg)
	if err != nil {
		return
	}

	// NewProxy will return a interface Proxy.
	// In fact it create different proxies by different proxy type, we just call run() here.
	pxy, err := proxy.NewProxy(ctl.runId, ctl.rc, ctl.statsCollector, ctl.poolCount, ctl.GetWorkConn, pxyConf)
	if err != nil {
		return remoteAddr, err
	}

	// Check ports used number in each client
	if g.GlbServerCfg.MaxPortsPerClient > 0 {
		ctl.mu.Lock()
		if ctl.portsUsedNum+pxy.GetUsedPortsNum() > int(g.GlbServerCfg.MaxPortsPerClient) {
			ctl.mu.Unlock()
			err = fmt.Errorf("exceed the max_ports_per_client")
			return
		}
		ctl.portsUsedNum = ctl.portsUsedNum + pxy.GetUsedPortsNum()
		ctl.mu.Unlock()

		defer func() {
			if err != nil {
				ctl.mu.Lock()
				ctl.portsUsedNum = ctl.portsUsedNum - pxy.GetUsedPortsNum()
				ctl.mu.Unlock()
			}
		}()
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

	err = ctl.pxyManager.Add(pxyMsg.ProxyName, pxy)
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
	pxy, ok := ctl.proxies[closeMsg.ProxyName]
	if !ok {
		ctl.mu.Unlock()
		return
	}

	if g.GlbServerCfg.MaxPortsPerClient > 0 {
		ctl.portsUsedNum = ctl.portsUsedNum - pxy.GetUsedPortsNum()
	}
	pxy.Close()
	ctl.pxyManager.Del(pxy.GetName())
	delete(ctl.proxies, closeMsg.ProxyName)
	ctl.mu.Unlock()

	ctl.statsCollector.Mark(stats.TypeCloseProxy, &stats.CloseProxyPayload{
		Name:      pxy.GetName(),
		ProxyType: pxy.GetConf().GetBaseInfo().ProxyType,
	})
	return
}

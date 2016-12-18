// Copyright 2016 fatedier, fatedier@gmail.com
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
	"sync"
	"time"

	"github.com/fatedier/frp/src/models/config"
	"github.com/fatedier/frp/src/models/consts"
	"github.com/fatedier/frp/src/models/metric"
	"github.com/fatedier/frp/src/models/msg"
	"github.com/fatedier/frp/src/utils/conn"
	"github.com/fatedier/frp/src/utils/log"
)

type Listener interface {
	Accept() (*conn.Conn, error)
	Close() error
}

type ProxyServer struct {
	*config.ProxyServerConf

	CtlConn      *conn.Conn      `json:"-"` // control connection with frpc
	listeners    []Listener      // accept new connection from remote users
	ctlMsgChan   chan int64      // every time accept a new user conn, put "1" to the channel
	workConnChan chan *conn.Conn // get new work conns from control goroutine
	mutex        sync.RWMutex
	closeChan    chan struct{} // for notify other goroutines that the proxy is closed by close this channel
}

func NewProxyServer() (p *ProxyServer) {
	psc := &config.ProxyServerConf{CustomDomains: make([]string, 0)}
	p = &ProxyServer{
		ProxyServerConf: psc,
	}
	return p
}

func NewProxyServerFromCtlMsg(req *msg.ControlReq) (p *ProxyServer) {
	p = &ProxyServer{}
	p.Name = req.ProxyName
	p.Type = req.ProxyType
	p.UseEncryption = req.UseEncryption
	p.UseGzip = req.UseGzip
	p.PrivilegeMode = req.PrivilegeMode
	p.PrivilegeToken = PrivilegeToken
	p.BindAddr = BindAddr
	if p.Type == "tcp" {
		p.ListenPort = req.RemotePort
	} else if p.Type == "http" {
		p.ListenPort = VhostHttpPort
	} else if p.Type == "https" {
		p.ListenPort = VhostHttpsPort
	}
	p.CustomDomains = req.CustomDomains
	p.HostHeaderRewrite = req.HostHeaderRewrite
	p.HttpUserName = req.HttpUserName
	p.HttpPassWord = req.HttpPassWord
	return
}

func (p *ProxyServer) Init() {
	p.Lock()
	p.Status = consts.Idle
	metric.SetStatus(p.Name, p.Status)
	p.workConnChan = make(chan *conn.Conn, p.PoolCount+10)
	p.ctlMsgChan = make(chan int64, p.PoolCount+10)
	p.listeners = make([]Listener, 0)
	p.closeChan = make(chan struct{})
	p.Unlock()
}

func (p *ProxyServer) Compare(p2 *ProxyServer) bool {
	if p.Name != p2.Name || p.AuthToken != p2.AuthToken || p.Type != p2.Type ||
		p.BindAddr != p2.BindAddr || p.ListenPort != p2.ListenPort || p.HostHeaderRewrite != p2.HostHeaderRewrite {
		return false
	}
	if len(p.CustomDomains) != len(p2.CustomDomains) {
		return false
	}
	for i, _ := range p.CustomDomains {
		if p.CustomDomains[i] != p2.CustomDomains[i] {
			return false
		}
	}

	if len(p.Locations) != len(p2.Locations) {
		return false
	}
	for i, _ := range p.Locations {
		if p.Locations[i] != p2.Locations[i] {
			return false
		}
	}
	return true
}

func (p *ProxyServer) Lock() {
	p.mutex.Lock()
}

func (p *ProxyServer) Unlock() {
	p.mutex.Unlock()
}

// start listening for user conns
func (p *ProxyServer) Start(c *conn.Conn) (err error) {
	p.CtlConn = c
	p.Init()
	if p.Type == "tcp" {
		l, err := conn.Listen(p.BindAddr, p.ListenPort)
		if err != nil {
			return err
		}
		p.listeners = append(p.listeners, l)
	} else if p.Type == "http" {
		ls := VhostHttpMuxer.Listen(p.ProxyServerConf)
		for _, l := range ls {
			p.listeners = append(p.listeners, l)
		}
	} else if p.Type == "https" {
		ls := VhostHttpsMuxer.Listen(p.ProxyServerConf)
		for _, l := range ls {
			p.listeners = append(p.listeners, l)
		}
	}

	p.Lock()
	p.Status = consts.Working
	p.Unlock()
	metric.SetStatus(p.Name, p.Status)

	// create connection pool if needed
	if p.PoolCount > 0 {
		go p.connectionPoolManager(p.closeChan)
	}

	// start a goroutine for every listener to accept user connection
	for _, listener := range p.listeners {
		go func(l Listener) {
			for {
				// block
				// if listener is closed, err returned
				c, err := l.Accept()
				if err != nil {
					log.Info("ProxyName [%s], listener is closed", p.Name)
					return
				}
				log.Debug("ProxyName [%s], get one new user conn [%s]", p.Name, c.GetRemoteAddr())

				if p.Status != consts.Working {
					log.Debug("ProxyName [%s] is not working, new user conn close", p.Name)
					c.Close()
					return
				}

				go func(userConn *conn.Conn) {
					workConn, err := p.getWorkConn()
					if err != nil {
						return
					}

					// message will be transferred to another without modifying
					// l means local, r means remote
					log.Debug("Join two connections, (l[%s] r[%s]) (l[%s] r[%s])", workConn.GetLocalAddr(), workConn.GetRemoteAddr(),
						userConn.GetLocalAddr(), userConn.GetRemoteAddr())

					needRecord := true
					go msg.JoinMore(userConn, workConn, p.BaseConf, needRecord)
				}(c)
			}
		}(listener)
	}
	return nil
}

func (p *ProxyServer) Close() {
	p.Lock()
	if p.Status != consts.Closed {
		p.Status = consts.Closed
		for _, l := range p.listeners {
			if l != nil {
				l.Close()
			}
		}
		close(p.ctlMsgChan)
		close(p.workConnChan)
		close(p.closeChan)
		if p.CtlConn != nil {
			p.CtlConn.Close()
		}
	}
	metric.SetStatus(p.Name, p.Status)
	// if the proxy created by PrivilegeMode, delete it when closed
	if p.PrivilegeMode {
		DeleteProxy(p.Name)
	}
	p.Unlock()
}

func (p *ProxyServer) WaitUserConn() (closeFlag bool) {
	closeFlag = false

	_, ok := <-p.ctlMsgChan
	if !ok {
		closeFlag = true
	}
	return
}

func (p *ProxyServer) RegisterNewWorkConn(c *conn.Conn) {
	select {
	case p.workConnChan <- c:
	default:
		log.Debug("ProxyName [%s], workConnChan is full, so close this work connection", p.Name)
	}
}

// When frps get one user connection, we get one work connection from the pool and return it.
// If no workConn available in the pool, send message to frpc to get one or more
// and wait until it is available.
// return an error if wait timeout
func (p *ProxyServer) getWorkConn() (workConn *conn.Conn, err error) {
	var ok bool
	// get a work connection from the pool
	for {
		select {
		case workConn, ok = <-p.workConnChan:
			if !ok {
				err = fmt.Errorf("ProxyName [%s], no work connections available, control is closing", p.Name)
				return
			}
		default:
			// no work connections available in the poll, send message to frpc to get more
			p.ctlMsgChan <- 1

			select {
			case workConn, ok = <-p.workConnChan:
				if !ok {
					err = fmt.Errorf("ProxyName [%s], no work connections available, control is closing", p.Name)
					return
				}

			case <-time.After(time.Duration(UserConnTimeout) * time.Second):
				log.Warn("ProxyName [%s], timeout trying to get work connection", p.Name)
				err = fmt.Errorf("ProxyName [%s], timeout trying to get work connection", p.Name)
				return
			}
		}

		// if connection pool is not used, we don't check the status
		// function CheckClosed will consume at least 1 millisecond if the connection isn't closed
		if p.PoolCount == 0 || !workConn.CheckClosed() {
			break
		} else {
			log.Warn("ProxyName [%s], connection got from pool, but it's already closed", p.Name)
		}
	}
	return
}

func (p *ProxyServer) connectionPoolManager(closeCh <-chan struct{}) {
	for {
		// check if we need more work connections and send messages to frpc to get more
		time.Sleep(time.Duration(2) * time.Second)
		select {
		// if the channel closed, it means the proxy is closed, so just return
		case <-closeCh:
			log.Info("ProxyName [%s], connectionPoolManager exit", p.Name)
			return
		default:
			curWorkConnNum := int64(len(p.workConnChan))
			diff := p.PoolCount - curWorkConnNum
			if diff > 0 {
				if diff < p.PoolCount/5 {
					diff = p.PoolCount*4/5 + 1
				} else if diff < p.PoolCount/2 {
					diff = p.PoolCount/4 + 1
				} else if diff < p.PoolCount*4/5 {
					diff = p.PoolCount/5 + 1
				} else {
					diff = p.PoolCount/10 + 1
				}
				if diff+curWorkConnNum > p.PoolCount {
					diff = p.PoolCount - curWorkConnNum
				}
				for i := 0; i < int(diff); i++ {
					p.ctlMsgChan <- 1
				}
			}
		}
	}
}

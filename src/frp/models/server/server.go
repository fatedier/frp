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

	"frp/models/consts"
	"frp/utils/conn"
	"frp/utils/log"
)

type Listener interface {
	Accept() (*conn.Conn, error)
	Close() error
}

type ProxyServer struct {
	Name          string
	AuthToken     string
	Type          string
	BindAddr      string
	ListenPort    int64
	CustomDomains []string

	// configure in frpc.ini
	UseEncryption bool

	Status       int64
	CtlConn      *conn.Conn      // control connection with frpc
	listeners    []Listener      // accept new connection from remote users
	ctlMsgChan   chan int64      // every time accept a new user conn, put "1" to the channel
	workConnChan chan *conn.Conn // get new work conns from control goroutine
	mutex        sync.Mutex
}

func NewProxyServer() (p *ProxyServer) {
	p = &ProxyServer{
		CustomDomains: make([]string, 0),
	}
	return p
}

func (p *ProxyServer) Init() {
	p.Lock()
	p.Status = consts.Idle
	p.workConnChan = make(chan *conn.Conn, 100)
	p.ctlMsgChan = make(chan int64)
	p.listeners = make([]Listener, 0)
	p.Unlock()
}

func (p *ProxyServer) Compare(p2 *ProxyServer) bool {
	if p.Name != p2.Name || p.AuthToken != p2.AuthToken || p.Type != p2.Type ||
		p.BindAddr != p2.BindAddr || p.ListenPort != p2.ListenPort {
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
		for _, domain := range p.CustomDomains {
			l, err := VhostHttpMuxer.Listen(domain)
			if err != nil {
				return err
			}
			p.listeners = append(p.listeners, l)
		}
	} else if p.Type == "https" {
		for _, domain := range p.CustomDomains {
			l, err := VhostHttpsMuxer.Listen(domain)
			if err != nil {
				return err
			}
			p.listeners = append(p.listeners, l)
		}
	}

	p.Lock()
	p.Status = consts.Working
	p.Unlock()

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

				// start another goroutine for join two conns from frpc and user
				go func() {
					workConn, err := p.getWorkConn()
					if err != nil {
						return
					}

					userConn := c
					// msg will transfer to another without modifying
					// l means local, r means remote
					log.Debug("Join two connections, (l[%s] r[%s]) (l[%s] r[%s])", workConn.GetLocalAddr(), workConn.GetRemoteAddr(),
						userConn.GetLocalAddr(), userConn.GetRemoteAddr())

					if p.UseEncryption {
						go conn.JoinMore(userConn, workConn, p.AuthToken)
					} else {
						go conn.Join(userConn, workConn)
					}
				}()
			}
		}(listener)
	}
	return nil
}

func (p *ProxyServer) Close() {
	p.Lock()
	if p.Status != consts.Closed {
		p.Status = consts.Closed
		if len(p.listeners) != 0 {
			for _, l := range p.listeners {
				if l != nil {
					l.Close()
				}
			}
		}
		close(p.ctlMsgChan)
		close(p.workConnChan)
		if p.CtlConn != nil {
			p.CtlConn.Close()
		}
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
	p.workConnChan <- c
}

// when frps get one user connection, we get one work connection from the pool and return it
// if no workConn available in the pool, send message to frpc to get one or more
// and wait until it is available
// return an error if wait timeout
func (p *ProxyServer) getWorkConn() (workConn *conn.Conn, err error) {
	var ok bool

	// get a work connection from the pool
	select {
	case workConn, ok = <-p.workConnChan:
		if !ok {
			err = fmt.Errorf("ProxyName [%s], no work connections available, control is closing", p.Name)
			return
		}
	default:
		// no work connections available in the poll, send message to frpc to get one
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
	return
}

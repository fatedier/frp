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
	"container/list"
	"sync"
	"time"

	"frp/models/consts"
	"frp/utils/conn"
	"frp/utils/log"
)

type ProxyServer struct {
	Name          string
	AuthToken     string
	UseEncryption bool
	BindAddr      string
	ListenPort    int64
	Status        int64

	listener     *conn.Listener  // accept new connection from remote users
	ctlMsgChan   chan int64      // every time accept a new user conn, put "1" to the channel
	workConnChan chan *conn.Conn // get new work conns from control goroutine
	userConnList *list.List      // store user conns
	mutex        sync.Mutex
}

func (p *ProxyServer) Init() {
	p.Status = consts.Idle
	p.workConnChan = make(chan *conn.Conn)
	p.ctlMsgChan = make(chan int64)
	p.userConnList = list.New()
}

func (p *ProxyServer) Lock() {
	p.mutex.Lock()
}

func (p *ProxyServer) Unlock() {
	p.mutex.Unlock()
}

// start listening for user conns
func (p *ProxyServer) Start() (err error) {
	p.Init()
	p.listener, err = conn.Listen(p.BindAddr, p.ListenPort)
	if err != nil {
		return err
	}

	p.Status = consts.Working

	// start a goroutine for listener to accept user connection
	go func() {
		for {
			// block
			// if listener is closed, err returned
			c, err := p.listener.GetConn()
			if err != nil {
				log.Info("ProxyName [%s], listener is closed", p.Name)
				return
			}
			log.Debug("ProxyName [%s], get one new user conn [%s]", p.Name, c.GetRemoteAddr())

			// insert into list
			p.Lock()
			if p.Status != consts.Working {
				log.Debug("ProxyName [%s] is not working, new user conn close", p.Name)
				c.Close()
				p.Unlock()
				return
			}
			p.userConnList.PushBack(c)
			p.Unlock()

			// put msg to control conn
			p.ctlMsgChan <- 1

			// set timeout
			time.AfterFunc(time.Duration(UserConnTimeout)*time.Second, func() {
				p.Lock()
				defer p.Unlock()
				element := p.userConnList.Front()
				if element == nil {
					return
				}

				userConn := element.Value.(*conn.Conn)
				if userConn == c {
					log.Warn("ProxyName [%s], user conn [%s] timeout", p.Name, c.GetRemoteAddr())
				}
			})
		}
	}()

	// start another goroutine for join two conns from client and user
	go func() {
		for {
			workConn, ok := <-p.workConnChan
			if !ok {
				return
			}

			p.Lock()
			element := p.userConnList.Front()

			var userConn *conn.Conn
			if element != nil {
				userConn = element.Value.(*conn.Conn)
				p.userConnList.Remove(element)
			} else {
				workConn.Close()
				p.Unlock()
				continue
			}
			p.Unlock()

			// msg will transfer to another without modifying
			// l means local, r means remote
			log.Debug("Join two connections, (l[%s] r[%s]) (l[%s] r[%s])", workConn.GetLocalAddr(), workConn.GetRemoteAddr(),
				userConn.GetLocalAddr(), userConn.GetRemoteAddr())

			if p.UseEncryption {
				go conn.JoinMore(userConn, workConn, p.AuthToken)
			} else {
				go conn.Join(userConn, workConn)
			}
		}
	}()

	return nil
}

func (p *ProxyServer) Close() {
	p.Lock()
	if p.Status != consts.Closed {
		p.Status = consts.Closed
		if p.listener != nil {
			p.listener.Close()
		}
		close(p.ctlMsgChan)
		close(p.workConnChan)
		p.userConnList = list.New()
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

func (p *ProxyServer) RecvNewWorkConn(c *conn.Conn) {
	p.workConnChan <- c
}

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
	"net"
	"sync"
	"time"

	"github.com/fatedier/frp/src/models/config"
	"github.com/fatedier/frp/src/models/consts"
	"github.com/fatedier/frp/src/models/metric"
	"github.com/fatedier/frp/src/models/msg"
	"github.com/fatedier/frp/src/utils/conn"
	"github.com/fatedier/frp/src/utils/log"
	"github.com/fatedier/frp/src/utils/pool"
)

type Listener interface {
	Accept() (*conn.Conn, error)
	Close() error
}

type ProxyServer struct {
	config.BaseConf
	BindAddr      string
	ListenPort    int64
	CustomDomains []string

	Status      int64
	CtlConn     *conn.Conn // control connection with frpc
	WorkConnUdp *conn.Conn // work connection for udp

	udpConn       *net.UDPConn
	listeners     []Listener      // accept new connection from remote users
	ctlMsgChan    chan int64      // every time accept a new user conn, put "1" to the channel
	workConnChan  chan *conn.Conn // get new work conns from control goroutine
	udpSenderChan chan *msg.UdpPacket
	mutex         sync.RWMutex
	closeChan     chan struct{} // close this channel for notifying other goroutines that the proxy is closed
}

func NewProxyServer() (p *ProxyServer) {
	p = &ProxyServer{
		CustomDomains: make([]string, 0),
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
	if p.Type == "tcp" || p.Type == "udp" {
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
	p.udpSenderChan = make(chan *msg.UdpPacket, 1024)
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
			l, err := VhostHttpMuxer.Listen(domain, p.HostHeaderRewrite, p.HttpUserName, p.HttpPassWord)
			if err != nil {
				return err
			}
			p.listeners = append(p.listeners, l)
		}
	} else if p.Type == "https" {
		for _, domain := range p.CustomDomains {
			l, err := VhostHttpsMuxer.Listen(domain, p.HostHeaderRewrite, p.HttpUserName, p.HttpPassWord)
			if err != nil {
				return err
			}
			p.listeners = append(p.listeners, l)
		}
	}

	p.Lock()
	p.Status = consts.Working
	p.Unlock()
	metric.SetStatus(p.Name, p.Status)

	if p.Type == "udp" {
		// udp is special
		p.udpConn, err = conn.ListenUDP(p.BindAddr, p.ListenPort)
		if err != nil {
			log.Warn("ProxyName [%s], listen udp port error: %v", p.Name, err)
			return err
		}
		go func() {
			for {
				buf := pool.GetBuf(2048)
				n, remoteAddr, err := p.udpConn.ReadFromUDP(buf)
				if err != nil {
					log.Info("ProxyName [%s], udp listener is closed", p.Name)
					return
				}
				localAddr, _ := net.ResolveUDPAddr("udp", p.udpConn.LocalAddr().String())
				udpPacket := msg.NewUdpPacket(buf[0:n], remoteAddr, localAddr)
				select {
				case p.udpSenderChan <- udpPacket:
				default:
					log.Warn("ProxyName [%s], udp sender channel is full", p.Name)
				}
				pool.PutBuf(buf)
			}
		}()
	} else {
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
		close(p.udpSenderChan)
		close(p.closeChan)
		if p.CtlConn != nil {
			p.CtlConn.Close()
		}
		if p.WorkConnUdp != nil {
			p.WorkConnUdp.Close()
		}
		if p.udpConn != nil {
			p.udpConn.Close()
			p.udpConn = nil
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
		c.Close()
	}
}

// create a tcp connection for forwarding udp packages
func (p *ProxyServer) RegisterNewWorkConnUdp(c *conn.Conn) {
	if p.WorkConnUdp != nil && !p.WorkConnUdp.IsClosed() {
		p.WorkConnUdp.Close()
	}
	p.WorkConnUdp = c

	// read
	go func() {
		var (
			buf string
			err error
		)
		for {
			buf, err = c.ReadLine()
			if err != nil {
				log.Warn("ProxyName [%s], work connection for udp closed", p.Name)
				return
			}
			udpPacket := &msg.UdpPacket{}
			err = udpPacket.UnPack([]byte(buf))
			if err != nil {
				log.Warn("ProxyName [%s], unpack udp packet error: %v", p.Name, err)
				continue
			}

			// send to user
			_, err = p.udpConn.WriteToUDP(udpPacket.Content, udpPacket.Dst)
			if err != nil {
				continue
			}
		}
	}()

	// write
	go func() {
		for {
			udpPacket, ok := <-p.udpSenderChan
			if !ok {
				return
			}
			err := c.WriteString(string(udpPacket.Pack()) + "\n")
			if err != nil {
				log.Debug("ProxyName [%s], write to work connection for udp error: %v", p.Name, err)
				return
			}
		}
	}()
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

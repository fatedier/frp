// Copyright 2018 fatedier, fatedier@gmail.com
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

package group

import (
	"fmt"
	"net"
	"sync"

	"github.com/fatedier/frp/server/ports"

	gerr "github.com/fatedier/golib/errors"
)

type TcpGroupListener struct {
	groupName string
	group     *TcpGroup

	addr    net.Addr
	closeCh chan struct{}
}

func newTcpGroupListener(name string, group *TcpGroup, addr net.Addr) *TcpGroupListener {
	return &TcpGroupListener{
		groupName: name,
		group:     group,
		addr:      addr,
		closeCh:   make(chan struct{}),
	}
}

func (ln *TcpGroupListener) Accept() (c net.Conn, err error) {
	var ok bool
	select {
	case <-ln.closeCh:
		return nil, ErrListenerClosed
	case c, ok = <-ln.group.Accept():
		if !ok {
			return nil, ErrListenerClosed
		}
		return c, nil
	}
}

func (ln *TcpGroupListener) Addr() net.Addr {
	return ln.addr
}

func (ln *TcpGroupListener) Close() (err error) {
	close(ln.closeCh)
	ln.group.CloseListener(ln)
	return
}

type TcpGroup struct {
	group    string
	groupKey string
	addr     string
	port     int
	realPort int

	acceptCh chan net.Conn
	index    uint64
	tcpLn    net.Listener
	lns      []*TcpGroupListener
	ctl      *TcpGroupCtl
	mu       sync.Mutex
}

func NewTcpGroup(ctl *TcpGroupCtl) *TcpGroup {
	return &TcpGroup{
		lns:      make([]*TcpGroupListener, 0),
		ctl:      ctl,
		acceptCh: make(chan net.Conn),
	}
}

func (tg *TcpGroup) Listen(proxyName string, group string, groupKey string, addr string, port int) (ln *TcpGroupListener, realPort int, err error) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	if len(tg.lns) == 0 {
		realPort, err = tg.ctl.portManager.Acquire(proxyName, port)
		if err != nil {
			return
		}
		tcpLn, errRet := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
		if errRet != nil {
			err = errRet
			return
		}
		ln = newTcpGroupListener(group, tg, tcpLn.Addr())

		tg.group = group
		tg.groupKey = groupKey
		tg.addr = addr
		tg.port = port
		tg.realPort = realPort
		tg.tcpLn = tcpLn
		tg.lns = append(tg.lns, ln)
		if tg.acceptCh == nil {
			tg.acceptCh = make(chan net.Conn)
		}
		go tg.worker()
	} else {
		if tg.group != group || tg.addr != addr {
			err = ErrGroupParamsInvalid
			return
		}
		if tg.port != port {
			err = ErrGroupDifferentPort
			return
		}
		if tg.groupKey != groupKey {
			err = ErrGroupAuthFailed
			return
		}
		ln = newTcpGroupListener(group, tg, tg.lns[0].Addr())
		realPort = tg.realPort
		tg.lns = append(tg.lns, ln)
	}
	return
}

func (tg *TcpGroup) worker() {
	for {
		c, err := tg.tcpLn.Accept()
		if err != nil {
			return
		}
		err = gerr.PanicToError(func() {
			tg.acceptCh <- c
		})
		if err != nil {
			return
		}
	}
}

func (tg *TcpGroup) Accept() <-chan net.Conn {
	return tg.acceptCh
}

func (tg *TcpGroup) CloseListener(ln *TcpGroupListener) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	for i, tmpLn := range tg.lns {
		if tmpLn == ln {
			tg.lns = append(tg.lns[:i], tg.lns[i+1:]...)
			break
		}
	}
	if len(tg.lns) == 0 {
		close(tg.acceptCh)
		tg.tcpLn.Close()
		tg.ctl.portManager.Release(tg.realPort)
		tg.ctl.RemoveGroup(tg.group)
	}
}

type TcpGroupCtl struct {
	groups map[string]*TcpGroup

	portManager *ports.PortManager
	mu          sync.Mutex
}

func NewTcpGroupCtl(portManager *ports.PortManager) *TcpGroupCtl {
	return &TcpGroupCtl{
		groups:      make(map[string]*TcpGroup),
		portManager: portManager,
	}
}

func (tgc *TcpGroupCtl) Listen(proxyNanme string, group string, groupKey string,
	addr string, port int) (l net.Listener, realPort int, err error) {

	tgc.mu.Lock()
	defer tgc.mu.Unlock()
	if tcpGroup, ok := tgc.groups[group]; ok {
		return tcpGroup.Listen(proxyNanme, group, groupKey, addr, port)
	} else {
		tcpGroup = NewTcpGroup(tgc)
		tgc.groups[group] = tcpGroup
		return tcpGroup.Listen(proxyNanme, group, groupKey, addr, port)
	}
}

func (tgc *TcpGroupCtl) RemoveGroup(group string) {
	tgc.mu.Lock()
	defer tgc.mu.Unlock()
	delete(tgc.groups, group)
}

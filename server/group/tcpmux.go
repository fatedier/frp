// Copyright 2020 guylewin, guy@lewin.co.il
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
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/fatedier/frp/models/consts"
	"github.com/fatedier/frp/utils/tcpmux"
	"github.com/fatedier/frp/utils/vhost"

	gerr "github.com/fatedier/golib/errors"
)

// TcpMuxGroupCtl manage all TcpMuxGroups
type TcpMuxGroupCtl struct {
	groups map[string]*TcpMuxGroup

	// portManager is used to manage port
	tcpMuxHttpConnectMuxer *tcpmux.HttpConnectTcpMuxer
	mu                     sync.Mutex
}

// NewTcpMuxGroupCtl return a new TcpMuxGroupCtl
func NewTcpMuxGroupCtl(tcpMuxHttpConnectMuxer *tcpmux.HttpConnectTcpMuxer) *TcpMuxGroupCtl {
	return &TcpMuxGroupCtl{
		groups:                 make(map[string]*TcpMuxGroup),
		tcpMuxHttpConnectMuxer: tcpMuxHttpConnectMuxer,
	}
}

// Listen is the wrapper for TcpMuxGroup's Listen
// If there are no group, we will create one here
func (tmgc *TcpMuxGroupCtl) Listen(multiplexer string, group string, groupKey string,
	domain string, ctx context.Context) (l net.Listener, err error) {
	tmgc.mu.Lock()
	tcpMuxGroup, ok := tmgc.groups[group]
	if !ok {
		tcpMuxGroup = NewTcpMuxGroup(tmgc)
		tmgc.groups[group] = tcpMuxGroup
	}
	tmgc.mu.Unlock()

	switch multiplexer {
	case consts.HttpConnectTcpMultiplexer:
		return tcpMuxGroup.HttpConnectListen(group, groupKey, domain, ctx)
	default:
		err = fmt.Errorf("unknown multiplexer [%s]", multiplexer)
		return
	}
}

// RemoveGroup remove TcpMuxGroup from controller
func (tmgc *TcpMuxGroupCtl) RemoveGroup(group string) {
	tmgc.mu.Lock()
	defer tmgc.mu.Unlock()
	delete(tmgc.groups, group)
}

// TcpMuxGroup route connections to different proxies
type TcpMuxGroup struct {
	group    string
	groupKey string
	domain   string

	acceptCh chan net.Conn
	index    uint64
	tcpMuxLn net.Listener
	lns      []*TcpMuxGroupListener
	ctl      *TcpMuxGroupCtl
	mu       sync.Mutex
}

// NewTcpMuxGroup return a new TcpMuxGroup
func NewTcpMuxGroup(ctl *TcpMuxGroupCtl) *TcpMuxGroup {
	return &TcpMuxGroup{
		lns:      make([]*TcpMuxGroupListener, 0),
		ctl:      ctl,
		acceptCh: make(chan net.Conn),
	}
}

// Listen will return a new TcpMuxGroupListener
// if TcpMuxGroup already has a listener, just add a new TcpMuxGroupListener to the queues
// otherwise, listen on the real address
func (tmg *TcpMuxGroup) HttpConnectListen(group string, groupKey string, domain string, context context.Context) (ln *TcpMuxGroupListener, err error) {
	tmg.mu.Lock()
	defer tmg.mu.Unlock()
	if len(tmg.lns) == 0 {
		// the first listener, listen on the real address
		routeConfig := &vhost.VhostRouteConfig{
			Domain: domain,
		}
		tcpMuxLn, errRet := tmg.ctl.tcpMuxHttpConnectMuxer.Listen(context, routeConfig)
		if errRet != nil {
			return nil, errRet
		}
		ln = newTcpMuxGroupListener(group, tmg, tcpMuxLn.Addr())

		tmg.group = group
		tmg.groupKey = groupKey
		tmg.domain = domain
		tmg.tcpMuxLn = tcpMuxLn
		tmg.lns = append(tmg.lns, ln)
		if tmg.acceptCh == nil {
			tmg.acceptCh = make(chan net.Conn)
		}
		go tmg.worker()
	} else {
		// domain in the same group must be equal
		if tmg.group != group || tmg.domain != domain {
			return nil, ErrGroupParamsInvalid
		}
		if tmg.groupKey != groupKey {
			return nil, ErrGroupAuthFailed
		}
		ln = newTcpMuxGroupListener(group, tmg, tmg.lns[0].Addr())
		tmg.lns = append(tmg.lns, ln)
	}
	return
}

// worker is called when the real tcp listener has been created
func (tmg *TcpMuxGroup) worker() {
	for {
		c, err := tmg.tcpMuxLn.Accept()
		if err != nil {
			return
		}
		err = gerr.PanicToError(func() {
			tmg.acceptCh <- c
		})
		if err != nil {
			return
		}
	}
}

func (tmg *TcpMuxGroup) Accept() <-chan net.Conn {
	return tmg.acceptCh
}

// CloseListener remove the TcpMuxGroupListener from the TcpMuxGroup
func (tmg *TcpMuxGroup) CloseListener(ln *TcpMuxGroupListener) {
	tmg.mu.Lock()
	defer tmg.mu.Unlock()
	for i, tmpLn := range tmg.lns {
		if tmpLn == ln {
			tmg.lns = append(tmg.lns[:i], tmg.lns[i+1:]...)
			break
		}
	}
	if len(tmg.lns) == 0 {
		close(tmg.acceptCh)
		tmg.tcpMuxLn.Close()
		tmg.ctl.RemoveGroup(tmg.group)
	}
}

// TcpMuxGroupListener
type TcpMuxGroupListener struct {
	groupName string
	group     *TcpMuxGroup

	addr    net.Addr
	closeCh chan struct{}
}

func newTcpMuxGroupListener(name string, group *TcpMuxGroup, addr net.Addr) *TcpMuxGroupListener {
	return &TcpMuxGroupListener{
		groupName: name,
		group:     group,
		addr:      addr,
		closeCh:   make(chan struct{}),
	}
}

// Accept will accept connections from TcpMuxGroup
func (ln *TcpMuxGroupListener) Accept() (c net.Conn, err error) {
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

func (ln *TcpMuxGroupListener) Addr() net.Addr {
	return ln.addr
}

// Close close the listener
func (ln *TcpMuxGroupListener) Close() (err error) {
	close(ln.closeCh)

	// remove self from TcpMuxGroup
	ln.group.CloseListener(ln)
	return
}

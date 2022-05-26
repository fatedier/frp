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

	"github.com/fatedier/frp/pkg/consts"
	"github.com/fatedier/frp/pkg/util/tcpmux"
	"github.com/fatedier/frp/pkg/util/vhost"

	gerr "github.com/fatedier/golib/errors"
)

// TCPMuxGroupCtl manage all TCPMuxGroups
type TCPMuxGroupCtl struct {
	groups map[string]*TCPMuxGroup

	// portManager is used to manage port
	tcpMuxHTTPConnectMuxer *tcpmux.HTTPConnectTCPMuxer
	mu                     sync.Mutex
}

// NewTCPMuxGroupCtl return a new TCPMuxGroupCtl
func NewTCPMuxGroupCtl(tcpMuxHTTPConnectMuxer *tcpmux.HTTPConnectTCPMuxer) *TCPMuxGroupCtl {
	return &TCPMuxGroupCtl{
		groups:                 make(map[string]*TCPMuxGroup),
		tcpMuxHTTPConnectMuxer: tcpMuxHTTPConnectMuxer,
	}
}

// Listen is the wrapper for TCPMuxGroup's Listen
// If there are no group, we will create one here
func (tmgc *TCPMuxGroupCtl) Listen(
	ctx context.Context,
	multiplexer, group, groupKey string,
	routeConfig vhost.RouteConfig,
) (l net.Listener, err error) {

	tmgc.mu.Lock()
	tcpMuxGroup, ok := tmgc.groups[group]
	if !ok {
		tcpMuxGroup = NewTCPMuxGroup(tmgc)
		tmgc.groups[group] = tcpMuxGroup
	}
	tmgc.mu.Unlock()

	switch multiplexer {
	case consts.HTTPConnectTCPMultiplexer:
		return tcpMuxGroup.HTTPConnectListen(ctx, group, groupKey, routeConfig)
	default:
		err = fmt.Errorf("unknown multiplexer [%s]", multiplexer)
		return
	}
}

// RemoveGroup remove TCPMuxGroup from controller
func (tmgc *TCPMuxGroupCtl) RemoveGroup(group string) {
	tmgc.mu.Lock()
	defer tmgc.mu.Unlock()
	delete(tmgc.groups, group)
}

// TCPMuxGroup route connections to different proxies
type TCPMuxGroup struct {
	group           string
	groupKey        string
	domain          string
	routeByHTTPUser string

	acceptCh chan net.Conn
	index    uint64
	tcpMuxLn net.Listener
	lns      []*TCPMuxGroupListener
	ctl      *TCPMuxGroupCtl
	mu       sync.Mutex
}

// NewTCPMuxGroup return a new TCPMuxGroup
func NewTCPMuxGroup(ctl *TCPMuxGroupCtl) *TCPMuxGroup {
	return &TCPMuxGroup{
		lns:      make([]*TCPMuxGroupListener, 0),
		ctl:      ctl,
		acceptCh: make(chan net.Conn),
	}
}

// Listen will return a new TCPMuxGroupListener
// if TCPMuxGroup already has a listener, just add a new TCPMuxGroupListener to the queues
// otherwise, listen on the real address
func (tmg *TCPMuxGroup) HTTPConnectListen(
	ctx context.Context,
	group, groupKey string,
	routeConfig vhost.RouteConfig,
) (ln *TCPMuxGroupListener, err error) {

	tmg.mu.Lock()
	defer tmg.mu.Unlock()
	if len(tmg.lns) == 0 {
		// the first listener, listen on the real address
		tcpMuxLn, errRet := tmg.ctl.tcpMuxHTTPConnectMuxer.Listen(ctx, &routeConfig)
		if errRet != nil {
			return nil, errRet
		}
		ln = newTCPMuxGroupListener(group, tmg, tcpMuxLn.Addr())

		tmg.group = group
		tmg.groupKey = groupKey
		tmg.domain = routeConfig.Domain
		tmg.routeByHTTPUser = routeConfig.RouteByHTTPUser
		tmg.tcpMuxLn = tcpMuxLn
		tmg.lns = append(tmg.lns, ln)
		if tmg.acceptCh == nil {
			tmg.acceptCh = make(chan net.Conn)
		}
		go tmg.worker()
	} else {
		// route config in the same group must be equal
		if tmg.group != group || tmg.domain != routeConfig.Domain || tmg.routeByHTTPUser != routeConfig.RouteByHTTPUser {
			return nil, ErrGroupParamsInvalid
		}
		if tmg.groupKey != groupKey {
			return nil, ErrGroupAuthFailed
		}
		ln = newTCPMuxGroupListener(group, tmg, tmg.lns[0].Addr())
		tmg.lns = append(tmg.lns, ln)
	}
	return
}

// worker is called when the real TCP listener has been created
func (tmg *TCPMuxGroup) worker() {
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

func (tmg *TCPMuxGroup) Accept() <-chan net.Conn {
	return tmg.acceptCh
}

// CloseListener remove the TCPMuxGroupListener from the TCPMuxGroup
func (tmg *TCPMuxGroup) CloseListener(ln *TCPMuxGroupListener) {
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

// TCPMuxGroupListener
type TCPMuxGroupListener struct {
	groupName string
	group     *TCPMuxGroup

	addr    net.Addr
	closeCh chan struct{}
}

func newTCPMuxGroupListener(name string, group *TCPMuxGroup, addr net.Addr) *TCPMuxGroupListener {
	return &TCPMuxGroupListener{
		groupName: name,
		group:     group,
		addr:      addr,
		closeCh:   make(chan struct{}),
	}
}

// Accept will accept connections from TCPMuxGroup
func (ln *TCPMuxGroupListener) Accept() (c net.Conn, err error) {
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

func (ln *TCPMuxGroupListener) Addr() net.Addr {
	return ln.addr
}

// Close close the listener
func (ln *TCPMuxGroupListener) Close() (err error) {
	close(ln.closeCh)

	// remove self from TcpMuxGroup
	ln.group.CloseListener(ln)
	return
}

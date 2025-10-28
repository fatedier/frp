// Copyright 2025 The frp Authors
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
	"net"
	"sync"

	gerr "github.com/fatedier/golib/errors"

	"github.com/fatedier/frp/pkg/util/vhost"
)

type HTTPSGroupController struct {
	groups map[string]*HTTPSGroup

	httpsMuxer *vhost.HTTPSMuxer

	mu sync.Mutex
}

func NewHTTPSGroupController(httpsMuxer *vhost.HTTPSMuxer) *HTTPSGroupController {
	return &HTTPSGroupController{
		groups:     make(map[string]*HTTPSGroup),
		httpsMuxer: httpsMuxer,
	}
}

func (ctl *HTTPSGroupController) Listen(
	ctx context.Context,
	group, groupKey string,
	routeConfig vhost.RouteConfig,
) (l net.Listener, err error) {
	indexKey := group
	ctl.mu.Lock()
	g, ok := ctl.groups[indexKey]
	if !ok {
		g = NewHTTPSGroup(ctl)
		ctl.groups[indexKey] = g
	}
	ctl.mu.Unlock()

	return g.Listen(ctx, group, groupKey, routeConfig)
}

func (ctl *HTTPSGroupController) RemoveGroup(group string) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()
	delete(ctl.groups, group)
}

type HTTPSGroup struct {
	group    string
	groupKey string
	domain   string

	acceptCh chan net.Conn
	httpsLn  *vhost.Listener
	lns      []*HTTPSGroupListener
	ctl      *HTTPSGroupController
	mu       sync.Mutex
}

func NewHTTPSGroup(ctl *HTTPSGroupController) *HTTPSGroup {
	return &HTTPSGroup{
		lns:      make([]*HTTPSGroupListener, 0),
		ctl:      ctl,
		acceptCh: make(chan net.Conn),
	}
}

func (g *HTTPSGroup) Listen(
	ctx context.Context,
	group, groupKey string,
	routeConfig vhost.RouteConfig,
) (ln *HTTPSGroupListener, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.lns) == 0 {
		// the first listener, listen on the real address
		httpsLn, errRet := g.ctl.httpsMuxer.Listen(ctx, &routeConfig)
		if errRet != nil {
			return nil, errRet
		}
		ln = newHTTPSGroupListener(group, g, httpsLn.Addr())

		g.group = group
		g.groupKey = groupKey
		g.domain = routeConfig.Domain
		g.httpsLn = httpsLn
		g.lns = append(g.lns, ln)
		go g.worker()
	} else {
		// route config in the same group must be equal
		if g.group != group || g.domain != routeConfig.Domain {
			return nil, ErrGroupParamsInvalid
		}
		if g.groupKey != groupKey {
			return nil, ErrGroupAuthFailed
		}
		ln = newHTTPSGroupListener(group, g, g.lns[0].Addr())
		g.lns = append(g.lns, ln)
	}
	return
}

func (g *HTTPSGroup) worker() {
	for {
		c, err := g.httpsLn.Accept()
		if err != nil {
			return
		}
		err = gerr.PanicToError(func() {
			g.acceptCh <- c
		})
		if err != nil {
			return
		}
	}
}

func (g *HTTPSGroup) Accept() <-chan net.Conn {
	return g.acceptCh
}

func (g *HTTPSGroup) CloseListener(ln *HTTPSGroupListener) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, tmpLn := range g.lns {
		if tmpLn == ln {
			g.lns = append(g.lns[:i], g.lns[i+1:]...)
			break
		}
	}
	if len(g.lns) == 0 {
		close(g.acceptCh)
		if g.httpsLn != nil {
			g.httpsLn.Close()
		}
		g.ctl.RemoveGroup(g.group)
	}
}

type HTTPSGroupListener struct {
	groupName string
	group     *HTTPSGroup

	addr    net.Addr
	closeCh chan struct{}
}

func newHTTPSGroupListener(name string, group *HTTPSGroup, addr net.Addr) *HTTPSGroupListener {
	return &HTTPSGroupListener{
		groupName: name,
		group:     group,
		addr:      addr,
		closeCh:   make(chan struct{}),
	}
}

func (ln *HTTPSGroupListener) Accept() (c net.Conn, err error) {
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

func (ln *HTTPSGroupListener) Addr() net.Addr {
	return ln.addr
}

func (ln *HTTPSGroupListener) Close() (err error) {
	close(ln.closeCh)

	// remove self from HTTPSGroup
	ln.group.CloseListener(ln)
	return
}

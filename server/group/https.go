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

	"github.com/fatedier/frp/pkg/util/vhost"
)

type HTTPSGroupController struct {
	groupRegistry[*HTTPSGroup]
	httpsMuxer *vhost.HTTPSMuxer
}

func NewHTTPSGroupController(httpsMuxer *vhost.HTTPSMuxer) *HTTPSGroupController {
	return &HTTPSGroupController{
		groupRegistry: newGroupRegistry[*HTTPSGroup](),
		httpsMuxer:    httpsMuxer,
	}
}

func (ctl *HTTPSGroupController) Listen(
	ctx context.Context,
	group, groupKey string,
	routeConfig vhost.RouteConfig,
) (l net.Listener, err error) {
	for {
		g := ctl.getOrCreate(group, func() *HTTPSGroup {
			return NewHTTPSGroup(ctl)
		})
		l, err = g.Listen(ctx, group, groupKey, routeConfig)
		if err == errGroupStale {
			continue
		}
		return
	}
}

type HTTPSGroup struct {
	baseGroup

	domain string
	ctl    *HTTPSGroupController
}

func NewHTTPSGroup(ctl *HTTPSGroupController) *HTTPSGroup {
	return &HTTPSGroup{
		ctl: ctl,
	}
}

func (g *HTTPSGroup) Listen(
	ctx context.Context,
	group, groupKey string,
	routeConfig vhost.RouteConfig,
) (ln *Listener, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.ctl.isCurrent(group, func(cur *HTTPSGroup) bool { return cur == g }) {
		return nil, errGroupStale
	}
	if len(g.lns) == 0 {
		// the first listener, listen on the real address
		httpsLn, errRet := g.ctl.httpsMuxer.Listen(ctx, &routeConfig)
		if errRet != nil {
			return nil, errRet
		}

		g.domain = routeConfig.Domain
		g.initBase(group, groupKey, httpsLn, func() {
			g.ctl.removeIf(g.group, func(cur *HTTPSGroup) bool {
				return cur == g
			})
		})
		ln = g.newListener(httpsLn.Addr())
		go g.worker(httpsLn, g.acceptCh)
	} else {
		// route config in the same group must be equal
		if g.group != group || g.domain != routeConfig.Domain {
			return nil, ErrGroupParamsInvalid
		}
		if g.groupKey != groupKey {
			return nil, ErrGroupAuthFailed
		}
		ln = g.newListener(g.lns[0].Addr())
	}
	return
}

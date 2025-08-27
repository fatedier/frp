// Copyright 2025 Satyajeet Singh, jeet.0733@gmail.com
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
	"sync/atomic"

	"github.com/fatedier/frp/pkg/util/vhost"
)

type HTTPSGroupController struct {
	// groups indexed by group name
	groups map[string]*HTTPSGroup

	// register createConn for each group to vhostRouter.
	// createConn will get a connection from one proxy of the group
	vhostRouter *vhost.Routers

	mu sync.Mutex
}

func NewHTTPSGroupController(vhostRouter *vhost.Routers) *HTTPSGroupController {
	return &HTTPSGroupController{
		groups:      make(map[string]*HTTPSGroup),
		vhostRouter: vhostRouter,
	}
}

func (ctl *HTTPSGroupController) Register(
	proxyName, group, groupKey string,
	routeConfig vhost.RouteConfig,
) (err error) {
	indexKey := group
	ctl.mu.Lock()
	g, ok := ctl.groups[indexKey]
	if !ok {
		g = NewHTTPSGroup(ctl)
		ctl.groups[indexKey] = g
	}
	ctl.mu.Unlock()

	return g.Register(proxyName, group, groupKey, routeConfig)
}

func (ctl *HTTPSGroupController) UnRegister(proxyName, group string, _ vhost.RouteConfig) {
	indexKey := group
	ctl.mu.Lock()
	defer ctl.mu.Unlock()
	g, ok := ctl.groups[indexKey]
	if !ok {
		return
	}

	isEmpty := g.UnRegister(proxyName)
	if isEmpty {
		delete(ctl.groups, indexKey)
	}
}

type HTTPSGroup struct {
	group    string
	groupKey string
	domain   string

	// CreateConnFuncs indexed by proxy name
	createFuncs map[string]vhost.CreateConnFunc
	pxyNames    []string
	index       uint64
	ctl         *HTTPSGroupController
	mu          sync.RWMutex
}

func NewHTTPSGroup(ctl *HTTPSGroupController) *HTTPSGroup {
	return &HTTPSGroup{
		createFuncs: make(map[string]vhost.CreateConnFunc),
		pxyNames:    make([]string, 0),
		ctl:         ctl,
	}
}

func (g *HTTPSGroup) Register(
	proxyName, group, groupKey string,
	routeConfig vhost.RouteConfig,
) (err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.createFuncs) == 0 {
		// the first proxy in this group
		// Create a deep copy to avoid shared state
		tmp := vhost.RouteConfig{
			Domain:          routeConfig.Domain,
			Location:        routeConfig.Location,
			RewriteHost:     routeConfig.RewriteHost,
			Username:        routeConfig.Username,
			Password:        routeConfig.Password,
			RouteByHTTPUser: routeConfig.RouteByHTTPUser,
		}
		// Deep copy maps to avoid shared state
		if routeConfig.Headers != nil {
			tmp.Headers = make(map[string]string, len(routeConfig.Headers))
			for k, v := range routeConfig.Headers {
				tmp.Headers[k] = v
			}
		}
		if routeConfig.ResponseHeaders != nil {
			tmp.ResponseHeaders = make(map[string]string, len(routeConfig.ResponseHeaders))
			for k, v := range routeConfig.ResponseHeaders {
				tmp.ResponseHeaders[k] = v
			}
		}
		tmp.CreateConnFn = g.createConn
		tmp.ChooseEndpointFn = g.chooseEndpoint
		tmp.CreateConnByEndpointFn = g.createConnByEndpoint
		err = g.ctl.vhostRouter.Add(routeConfig.Domain, "", "", &tmp)
		if err != nil {
			return
		}

		g.group = group
		g.groupKey = groupKey
		g.domain = routeConfig.Domain
	} else {
		if g.group != group || g.domain != routeConfig.Domain {
			err = ErrGroupParamsInvalid
			return
		}
		if g.groupKey != groupKey {
			err = ErrGroupAuthFailed
			return
		}
	}
	if _, ok := g.createFuncs[proxyName]; ok {
		err = ErrProxyRepeated
		return
	}
	g.createFuncs[proxyName] = routeConfig.CreateConnFn
	g.pxyNames = append(g.pxyNames, proxyName)
	return nil
}

func (g *HTTPSGroup) UnRegister(proxyName string) (isEmpty bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.createFuncs, proxyName)
	for i, name := range g.pxyNames {
		if name == proxyName {
			g.pxyNames = append(g.pxyNames[:i], g.pxyNames[i+1:]...)
			break
		}
	}

	if len(g.createFuncs) == 0 {
		isEmpty = true
		g.ctl.vhostRouter.Del(g.domain, "", "")
	}
	return
}

func (g *HTTPSGroup) createConn(remoteAddr string) (net.Conn, error) {
	var f vhost.CreateConnFunc
	newIndex := atomic.AddUint64(&g.index, 1)

	g.mu.RLock()
	group := g.group
	domain := g.domain
	if len(g.pxyNames) > 0 {
		name := g.pxyNames[int(newIndex)%len(g.pxyNames)]
		f = g.createFuncs[name]
	}
	g.mu.RUnlock()

	if f == nil {
		return nil, fmt.Errorf("no CreateConnFunc for https group [%s], domain [%s]",
			group, domain)
	}

	return f(remoteAddr)
}

func (g *HTTPSGroup) chooseEndpoint() (string, error) {
	newIndex := atomic.AddUint64(&g.index, 1)
	name := ""

	g.mu.RLock()
	group := g.group
	domain := g.domain
	if len(g.pxyNames) > 0 {
		name = g.pxyNames[int(newIndex)%len(g.pxyNames)]
	}
	g.mu.RUnlock()

	if name == "" {
		return "", fmt.Errorf("no healthy endpoint for https group [%s], domain [%s]",
			group, domain)
	}
	return name, nil
}

func (g *HTTPSGroup) createConnByEndpoint(endpoint, remoteAddr string) (net.Conn, error) {
	var f vhost.CreateConnFunc
	g.mu.RLock()
	f = g.createFuncs[endpoint]
	g.mu.RUnlock()

	if f == nil {
		return nil, fmt.Errorf("no CreateConnFunc for endpoint [%s] in group [%s]", endpoint, g.group)
	}
	return f(remoteAddr)
}

package group

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/fatedier/frp/utils/vhost"
)

type HTTPGroupController struct {
	groups map[string]*HTTPGroup

	vhostRouter *vhost.VhostRouters

	mu sync.Mutex
}

func NewHTTPGroupController(vhostRouter *vhost.VhostRouters) *HTTPGroupController {
	return &HTTPGroupController{
		groups:      make(map[string]*HTTPGroup),
		vhostRouter: vhostRouter,
	}
}

func (ctl *HTTPGroupController) Register(proxyName, group, groupKey string,
	routeConfig vhost.VhostRouteConfig) (err error) {

	indexKey := httpGroupIndex(group, routeConfig.Domain, routeConfig.Location)
	ctl.mu.Lock()
	g, ok := ctl.groups[indexKey]
	if !ok {
		g = NewHTTPGroup(ctl)
		ctl.groups[indexKey] = g
	}
	ctl.mu.Unlock()

	return g.Register(proxyName, group, groupKey, routeConfig)
}

func (ctl *HTTPGroupController) UnRegister(proxyName, group, domain, location string) {
	indexKey := httpGroupIndex(group, domain, location)
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

type HTTPGroup struct {
	group    string
	groupKey string
	domain   string
	location string

	createFuncs map[string]vhost.CreateConnFunc
	pxyNames    []string
	index       uint64
	ctl         *HTTPGroupController
	mu          sync.RWMutex
}

func NewHTTPGroup(ctl *HTTPGroupController) *HTTPGroup {
	return &HTTPGroup{
		createFuncs: make(map[string]vhost.CreateConnFunc),
		pxyNames:    make([]string, 0),
		ctl:         ctl,
	}
}

func (g *HTTPGroup) Register(proxyName, group, groupKey string,
	routeConfig vhost.VhostRouteConfig) (err error) {

	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.createFuncs) == 0 {
		// the first proxy in this group
		tmp := routeConfig // copy object
		tmp.CreateConnFn = g.createConn
		err = g.ctl.vhostRouter.Add(routeConfig.Domain, routeConfig.Location, &tmp)
		if err != nil {
			return
		}

		g.group = group
		g.groupKey = groupKey
		g.domain = routeConfig.Domain
		g.location = routeConfig.Location
	} else {
		if g.group != group || g.domain != routeConfig.Domain || g.location != routeConfig.Location {
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

func (g *HTTPGroup) UnRegister(proxyName string) (isEmpty bool) {
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
		g.ctl.vhostRouter.Del(g.domain, g.location)
	}
	return
}

func (g *HTTPGroup) createConn(remoteAddr string) (net.Conn, error) {
	var f vhost.CreateConnFunc
	newIndex := atomic.AddUint64(&g.index, 1)

	g.mu.RLock()
	group := g.group
	domain := g.domain
	location := g.location
	if len(g.pxyNames) > 0 {
		name := g.pxyNames[int(newIndex)%len(g.pxyNames)]
		f, _ = g.createFuncs[name]
	}
	g.mu.RUnlock()

	if f == nil {
		return nil, fmt.Errorf("no CreateConnFunc for http group [%s], domain [%s], location [%s]", group, domain, location)
	}

	return f(remoteAddr)
}

func httpGroupIndex(group, domain, location string) string {
	return fmt.Sprintf("%s_%s_%s", group, domain, location)
}

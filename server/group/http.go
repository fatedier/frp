package group

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/fatedier/frp/pkg/util/vhost"
)

// HTTPGroupController manages HTTP groups that use round-robin
// callback routing (fundamentally different from listener-based groups).
type HTTPGroupController struct {
	groupRegistry[*HTTPGroup]
	vhostRouter *vhost.Routers
}

func NewHTTPGroupController(vhostRouter *vhost.Routers) *HTTPGroupController {
	return &HTTPGroupController{
		groupRegistry: newGroupRegistry[*HTTPGroup](),
		vhostRouter:   vhostRouter,
	}
}

func (ctl *HTTPGroupController) Register(
	proxyName, group, groupKey string,
	routeConfig vhost.RouteConfig,
) error {
	for {
		g := ctl.getOrCreate(group, func() *HTTPGroup {
			return NewHTTPGroup(ctl)
		})
		err := g.Register(proxyName, group, groupKey, routeConfig)
		if err == errGroupStale {
			continue
		}
		return err
	}
}

func (ctl *HTTPGroupController) UnRegister(proxyName, group string, _ vhost.RouteConfig) {
	g, ok := ctl.get(group)
	if !ok {
		return
	}
	g.UnRegister(proxyName)
}

type HTTPGroup struct {
	group           string
	groupKey        string
	domain          string
	location        string
	routeByHTTPUser string

	// CreateConnFuncs indexed by proxy name
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

func (g *HTTPGroup) Register(
	proxyName, group, groupKey string,
	routeConfig vhost.RouteConfig,
) (err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.ctl.isCurrent(group, func(cur *HTTPGroup) bool { return cur == g }) {
		return errGroupStale
	}
	if len(g.createFuncs) == 0 {
		// the first proxy in this group
		tmp := routeConfig // copy object
		tmp.CreateConnFn = g.createConn
		tmp.ChooseEndpointFn = g.chooseEndpoint
		tmp.CreateConnByEndpointFn = g.createConnByEndpoint
		err = g.ctl.vhostRouter.Add(routeConfig.Domain, routeConfig.Location, routeConfig.RouteByHTTPUser, &tmp)
		if err != nil {
			return
		}

		g.group = group
		g.groupKey = groupKey
		g.domain = routeConfig.Domain
		g.location = routeConfig.Location
		g.routeByHTTPUser = routeConfig.RouteByHTTPUser
	} else {
		if g.group != group || g.domain != routeConfig.Domain ||
			g.location != routeConfig.Location || g.routeByHTTPUser != routeConfig.RouteByHTTPUser {
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

func (g *HTTPGroup) UnRegister(proxyName string) {
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
		g.ctl.vhostRouter.Del(g.domain, g.location, g.routeByHTTPUser)
		g.ctl.removeIf(g.group, func(cur *HTTPGroup) bool {
			return cur == g
		})
	}
}

func (g *HTTPGroup) createConn(remoteAddr string) (net.Conn, error) {
	var f vhost.CreateConnFunc
	newIndex := atomic.AddUint64(&g.index, 1)

	g.mu.RLock()
	group := g.group
	domain := g.domain
	location := g.location
	routeByHTTPUser := g.routeByHTTPUser
	if len(g.pxyNames) > 0 {
		name := g.pxyNames[newIndex%uint64(len(g.pxyNames))]
		f = g.createFuncs[name]
	}
	g.mu.RUnlock()

	if f == nil {
		return nil, fmt.Errorf("no CreateConnFunc for http group [%s], domain [%s], location [%s], routeByHTTPUser [%s]",
			group, domain, location, routeByHTTPUser)
	}

	return f(remoteAddr)
}

func (g *HTTPGroup) chooseEndpoint() (string, error) {
	newIndex := atomic.AddUint64(&g.index, 1)
	name := ""

	g.mu.RLock()
	group := g.group
	domain := g.domain
	location := g.location
	routeByHTTPUser := g.routeByHTTPUser
	if len(g.pxyNames) > 0 {
		name = g.pxyNames[newIndex%uint64(len(g.pxyNames))]
	}
	g.mu.RUnlock()

	if name == "" {
		return "", fmt.Errorf("no healthy endpoint for http group [%s], domain [%s], location [%s], routeByHTTPUser [%s]",
			group, domain, location, routeByHTTPUser)
	}
	return name, nil
}

func (g *HTTPGroup) createConnByEndpoint(endpoint, remoteAddr string) (net.Conn, error) {
	var f vhost.CreateConnFunc
	g.mu.RLock()
	f = g.createFuncs[endpoint]
	g.mu.RUnlock()

	if f == nil {
		return nil, fmt.Errorf("no CreateConnFunc for endpoint [%s] in group [%s]", endpoint, g.group)
	}
	return f(remoteAddr)
}

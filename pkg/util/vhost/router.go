package vhost

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"sync"
)

var ErrRouterConfigConflict = errors.New("router config conflict")

type routerByHTTPUser map[string][]*Router

type Routers struct {
	indexByDomain map[string]routerByHTTPUser

	mutex sync.RWMutex
}

type Router struct {
	domain   string
	location string
	httpUser string

	// store any object here
	payload any
}

func NewRouters() *Routers {
	return &Routers{
		indexByDomain: make(map[string]routerByHTTPUser),
	}
}

func (r *Routers) Add(domain, location, httpUser string, payload any) error {
	domain = strings.ToLower(domain)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exist := r.exist(domain, location, httpUser); exist {
		return ErrRouterConfigConflict
	}

	routersByHTTPUser, found := r.indexByDomain[domain]
	if !found {
		routersByHTTPUser = make(map[string][]*Router)
	}
	vrs, found := routersByHTTPUser[httpUser]
	if !found {
		vrs = make([]*Router, 0, 1)
	}

	vr := &Router{
		domain:   domain,
		location: location,
		httpUser: httpUser,
		payload:  payload,
	}
	vrs = append(vrs, vr)

	slices.SortFunc(vrs, func(a, b *Router) int {
		return -cmp.Compare(a.location, b.location)
	})

	routersByHTTPUser[httpUser] = vrs
	r.indexByDomain[domain] = routersByHTTPUser
	return nil
}

func (r *Routers) Del(domain, location, httpUser string) {
	domain = strings.ToLower(domain)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	routersByHTTPUser, found := r.indexByDomain[domain]
	if !found {
		return
	}

	vrs, found := routersByHTTPUser[httpUser]
	if !found {
		return
	}
	newVrs := make([]*Router, 0)
	for _, vr := range vrs {
		if vr.location != location {
			newVrs = append(newVrs, vr)
		}
	}
	routersByHTTPUser[httpUser] = newVrs
}

// Get returns the best location match for an exact host and exact HTTP user.
// It does not apply all-users, wildcard-domain, or catch-all-domain fallback.
func (r *Routers) Get(host, path, httpUser string) (vr *Router, exist bool) {
	host = strings.ToLower(host)

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.getLocked(host, path, httpUser)
}

// getLocked performs an exact-host lookup; host must already be lower-cased.
func (r *Routers) getLocked(host, path, httpUser string) (vr *Router, exist bool) {
	routersByHTTPUser, found := r.indexByDomain[host]
	if !found {
		return
	}

	vrs, found := routersByHTTPUser[httpUser]
	if !found {
		return
	}

	for _, vr = range vrs {
		if strings.HasPrefix(path, vr.location) {
			return vr, true
		}
	}
	return
}

func (r *Routers) getByRoute(host, path, httpUser string) (*Router, bool) {
	host = strings.ToLower(host)

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// First we check the full hostname; if it doesn't exist, then check wildcard domains.
	// For example, test.example.com checks *.example.com before falling back to "*".
	vr, ok := r.getExactOrAllUsersLocked(host, path, httpUser)
	if ok {
		return vr, true
	}

	hostSplit := strings.Split(host, ".")
	// Keep two-label hosts out of the wildcard walk, so example.com does not match *.com.
	for len(hostSplit) >= 3 {
		// Replace the leftmost remaining label with the wildcard marker.
		hostSplit[0] = "*"
		host = strings.Join(hostSplit, ".")
		vr, ok = r.getExactOrAllUsersLocked(host, path, httpUser)
		if ok {
			return vr, true
		}
		hostSplit = hostSplit[1:]
	}

	// Finally, try to check if there is one proxy whose domain is "*", which means match all domains.
	return r.getExactOrAllUsersLocked("*", path, httpUser)
}

func (r *Routers) getExactOrAllUsersLocked(host, path, httpUser string) (*Router, bool) {
	vr, ok := r.getLocked(host, path, httpUser)
	if ok {
		return vr, true
	}
	// Try to check if there is one proxy that doesn't specify routeByHTTPUser, it means match all.
	vr, ok = r.getLocked(host, path, "")
	if ok {
		return vr, true
	}
	return nil, false
}

func (r *Routers) exist(host, path, httpUser string) (route *Router, exist bool) {
	routersByHTTPUser, found := r.indexByDomain[host]
	if !found {
		return
	}
	routers, found := routersByHTTPUser[httpUser]
	if !found {
		return
	}

	for _, route = range routers {
		if path == route.location {
			return route, true
		}
	}
	return
}

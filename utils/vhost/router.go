package vhost

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

var (
	ErrRouterConfigConflict = errors.New("router config conflict")
)

type VhostRouters struct {
	RouterByDomain map[string][]*VhostRouter
	mutex          sync.RWMutex
}

type VhostRouter struct {
	domain   string
	location string

	payload interface{}
}

func NewVhostRouters() *VhostRouters {
	return &VhostRouters{
		RouterByDomain: make(map[string][]*VhostRouter),
	}
}

func (r *VhostRouters) Add(domain, location string, payload interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exist := r.exist(domain, location); exist {
		return ErrRouterConfigConflict
	}

	vrs, found := r.RouterByDomain[domain]
	if !found {
		vrs = make([]*VhostRouter, 0, 1)
	}

	vr := &VhostRouter{
		domain:   domain,
		location: location,
		payload:  payload,
	}
	vrs = append(vrs, vr)

	sort.Sort(sort.Reverse(ByLocation(vrs)))
	r.RouterByDomain[domain] = vrs
	return nil
}

func (r *VhostRouters) Del(domain, location string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	vrs, found := r.RouterByDomain[domain]
	if !found {
		return
	}
	newVrs := make([]*VhostRouter, 0)
	for _, vr := range vrs {
		if vr.location != location {
			newVrs = append(newVrs, vr)
		}
	}
	r.RouterByDomain[domain] = newVrs
}

func (r *VhostRouters) Get(host, path string) (vr *VhostRouter, exist bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	vrs, found := r.RouterByDomain[host]
	if !found {
		return
	}

	// can't support load balance, will to do
	for _, vr = range vrs {
		if strings.HasPrefix(path, vr.location) {
			return vr, true
		}
	}

	return
}

func (r *VhostRouters) exist(host, path string) (vr *VhostRouter, exist bool) {
	vrs, found := r.RouterByDomain[host]
	if !found {
		return
	}

	for _, vr = range vrs {
		if path == vr.location {
			return vr, true
		}
	}

	return
}

// sort by location
type ByLocation []*VhostRouter

func (a ByLocation) Len() int {
	return len(a)
}
func (a ByLocation) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByLocation) Less(i, j int) bool {
	return strings.Compare(a[i].location, a[j].location) < 0
}

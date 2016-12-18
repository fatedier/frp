package vhost

import (
	"sort"
	"strings"
	"sync"
)

type VhostRouters struct {
	RouterByDomain map[string][]*VhostRouter
	mutex          sync.RWMutex
}

type VhostRouter struct {
	name     string
	domain   string
	location string
	listener *Listener
}

func NewVhostRouters() *VhostRouters {
	return &VhostRouters{
		RouterByDomain: make(map[string][]*VhostRouter),
	}
}

//sort by location
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

func (r *VhostRouters) add(name, domain string, locations []string, l *Listener) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	vrs, found := r.RouterByDomain[domain]
	if !found {
		vrs = make([]*VhostRouter, 0)
	}

	for _, loc := range locations {
		vr := &VhostRouter{
			name:     name,
			domain:   domain,
			location: loc,
			listener: l,
		}
		vrs = append(vrs, vr)
	}

	sort.Reverse(ByLocation(vrs))
	r.RouterByDomain[domain] = vrs
}

func (r *VhostRouters) del(l *Listener) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	vrs, found := r.RouterByDomain[l.domain]
	if !found {
		return
	}

	for i, vr := range vrs {
		if vr.listener == l {
			if len(vrs) > i+1 {
				r.RouterByDomain[l.domain] = append(vrs[:i], vrs[i+1:]...)
			} else {
				r.RouterByDomain[l.domain] = vrs[:i]
			}
		}
	}
}

func (r *VhostRouters) get(rname string) (vr *VhostRouter, exist bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var domain, url string
	tmparray := strings.SplitN(rname, ":", 2)
	if len(tmparray) == 2 {
		domain = tmparray[0]
		url = tmparray[1]
	}

	vrs, exist := r.RouterByDomain[domain]
	if !exist {
		return
	}

	//can't support load balance,will to do
	for _, vr = range vrs {
		if strings.HasPrefix(url, vr.location) {
			return vr, true
		}
	}

	return
}

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
	Name     string
	Domain   string
	Location string
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
	return strings.Compare(a[i].Location, a[j].Location) < 0
}

func (r *VhostRouters) Add(name string, domains, locations []string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, domain := range domains {
		vrs, found := r.RouterByDomain[name]
		if !found {
			vrs = make([]*VhostRouter, 0)
		}

		for _, loc := range locations {
			vr := &VhostRouter{
				Name:     name,
				Domain:   domain,
				Location: loc,
			}
			vrs = append(vrs, vr)
		}

		sort_vrs := sort.Reverse(ByLocation(vrs))
		r.RouterByDomain[name] = sort_vrs.(ByLocation)
	}
}

func (r *VhostRouters) getName(domain, url string) (name string, exist bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	vrs, exist := r.RouterByDomain[domain]
	if !exist {
		return
	}

	for _, vr := range vrs {
		if strings.HasPrefix(url, vr.Location+"/") {
			return vr.Name, true
		}
	}

	return
}

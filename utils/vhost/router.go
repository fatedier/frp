package vhost

import (
	"errors"
	"math/rand"
	"sort"
	"strings"
	"sync"
)

var (
	ErrRouterConfigConflict = errors.New("router config conflict")
)

type VhostRouters struct {
	RouterByDomain  map[string][]*VhostRouter
	allowDuplicates bool
	mutex           sync.RWMutex
}

type VhostRouter struct {
	domain   string
	location string

	allowDuplicates bool
	payloads        []interface{}
}

func (vr *VhostRouter) getPayload() interface{} {
	if !vr.allowDuplicates {
		return vr.payloads[0]
	}
	return vr.payloads[rand.Intn(len(vr.payloads))]
}

func NewVhostRouters(allowDuplicates bool) *VhostRouters {
	return &VhostRouters{
		allowDuplicates: allowDuplicates,
		RouterByDomain:  make(map[string][]*VhostRouter),
	}
}

func (r *VhostRouters) Add(domain, location string, payload interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if vr, exist := r.exist(domain, location); exist {
		if !r.allowDuplicates {
			return ErrRouterConfigConflict
		}
		vr.payloads = append(vr.payloads, payload)
		return nil
	}

	vrs, found := r.RouterByDomain[domain]
	if !found {
		vrs = make([]*VhostRouter, 0, 1)
	}

	vr := &VhostRouter{
		domain:          domain,
		location:        location,
		allowDuplicates: r.allowDuplicates,
		payloads:        make([]interface{}, 1),
	}
	vr.payloads[0] = payload
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
	if len(newVrs) == 0 {
		delete(r.RouterByDomain, domain)
	} else {
		r.RouterByDomain[domain] = newVrs
	}
}

func (r *VhostRouters) DelPayloadFromLocation(domain, location string, payload interface{}) {
	r.mutex.Lock()

	vrs, found := r.RouterByDomain[domain]
	if !found {
		r.mutex.Unlock()
		return
	}
	newPayloadsLen := -1
	for _, vr := range vrs {
		if vr.location == location {
			newPayloads := make([]interface{}, 0)
			for _, payloadIter := range vr.payloads {
				if payloadIter != payload {
					newPayloads = append(newPayloads, payloadIter)
				}
			}
			vr.payloads = newPayloads
			newPayloadsLen = len(newPayloads)
			break
		}
	}

	r.mutex.Unlock()

	if newPayloadsLen == 0 {
		r.Del(domain, location)
	}
}

func (r *VhostRouters) Get(host, path string) (vr *VhostRouter, exist bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	vrs, found := r.RouterByDomain[host]
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

package group

import (
	"sync"
)

// groupRegistry is a concurrent map of named groups with
// automatic creation on first access.
type groupRegistry[G any] struct {
	groups map[string]G
	mu     sync.Mutex
}

func newGroupRegistry[G any]() groupRegistry[G] {
	return groupRegistry[G]{
		groups: make(map[string]G),
	}
}

func (r *groupRegistry[G]) getOrCreate(key string, newFn func() G) G {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.groups[key]
	if !ok {
		g = newFn()
		r.groups[key] = g
	}
	return g
}

func (r *groupRegistry[G]) get(key string) (G, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.groups[key]
	return g, ok
}

// isCurrent returns true if key exists in the registry and matchFn
// returns true for the stored value.
func (r *groupRegistry[G]) isCurrent(key string, matchFn func(G) bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.groups[key]
	return ok && matchFn(g)
}

// removeIf atomically looks up the group for key, calls fn on it,
// and removes the entry if fn returns true.
func (r *groupRegistry[G]) removeIf(key string, fn func(G) bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.groups[key]
	if !ok {
		return
	}
	if fn(g) {
		delete(r.groups, key)
	}
}

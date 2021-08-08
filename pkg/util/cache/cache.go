package cache

import (
	lru "github.com/hashicorp/golang-lru"
)

type Cacher interface {
	Add(key, value interface{}) (evicted bool)
	Contains(key interface{}) bool
	Get(key interface{}) (value interface{}, ok bool)
	Keys() []interface{}
	Len() int
	Purge()
	Remove(key interface{}) bool
}

type lruCache struct {
	*lru.Cache
}

func NewCache(size int) Cacher {
	l, _ := lru.New(size)
	return l
}

var DefaultCache = NewCache(1024)

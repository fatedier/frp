// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package memcache for session provider
//
// depend on github.com/bradfitz/gomemcache/memcache
//
// go install github.com/bradfitz/gomemcache/memcache
//
// Usage:
// import(
//   _ "github.com/astaxie/beego/session/memcache"
//   "github.com/astaxie/beego/session"
// )
//
//	func init() {
//		globalSessions, _ = session.NewManager("memcache", ``{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"127.0.0.1:11211"}``)
//		go globalSessions.GC()
//	}
//
// more docs: http://beego.me/docs/module/session.md
package memcache

import (
	"net/http"
	"strings"
	"sync"

	"github.com/astaxie/beego/session"

	"github.com/bradfitz/gomemcache/memcache"
)

var mempder = &MemProvider{}
var client *memcache.Client

// SessionStore memcache session store
type SessionStore struct {
	sid         string
	lock        sync.RWMutex
	values      map[interface{}]interface{}
	maxlifetime int64
}

// Set value in memcache session
func (rs *SessionStore) Set(key, value interface{}) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	rs.values[key] = value
	return nil
}

// Get value in memcache session
func (rs *SessionStore) Get(key interface{}) interface{} {
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	if v, ok := rs.values[key]; ok {
		return v
	}
	return nil
}

// Delete value in memcache session
func (rs *SessionStore) Delete(key interface{}) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	delete(rs.values, key)
	return nil
}

// Flush clear all values in memcache session
func (rs *SessionStore) Flush() error {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	rs.values = make(map[interface{}]interface{})
	return nil
}

// SessionID get memcache session id
func (rs *SessionStore) SessionID() string {
	return rs.sid
}

// SessionRelease save session values to memcache
func (rs *SessionStore) SessionRelease(w http.ResponseWriter) {
	b, err := session.EncodeGob(rs.values)
	if err != nil {
		return
	}
	item := memcache.Item{Key: rs.sid, Value: b, Expiration: int32(rs.maxlifetime)}
	client.Set(&item)
}

// MemProvider memcache session provider
type MemProvider struct {
	maxlifetime int64
	conninfo    []string
	poolsize    int
	password    string
}

// SessionInit init memcache session
// savepath like
// e.g. 127.0.0.1:9090
func (rp *MemProvider) SessionInit(maxlifetime int64, savePath string) error {
	rp.maxlifetime = maxlifetime
	rp.conninfo = strings.Split(savePath, ";")
	client = memcache.New(rp.conninfo...)
	return nil
}

// SessionRead read memcache session by sid
func (rp *MemProvider) SessionRead(sid string) (session.Store, error) {
	if client == nil {
		if err := rp.connectInit(); err != nil {
			return nil, err
		}
	}
	item, err := client.Get(sid)
	if err != nil && err == memcache.ErrCacheMiss {
		rs := &SessionStore{sid: sid, values: make(map[interface{}]interface{}), maxlifetime: rp.maxlifetime}
		return rs, nil
	}
	var kv map[interface{}]interface{}
	if len(item.Value) == 0 {
		kv = make(map[interface{}]interface{})
	} else {
		kv, err = session.DecodeGob(item.Value)
		if err != nil {
			return nil, err
		}
	}
	rs := &SessionStore{sid: sid, values: kv, maxlifetime: rp.maxlifetime}
	return rs, nil
}

// SessionExist check memcache session exist by sid
func (rp *MemProvider) SessionExist(sid string) bool {
	if client == nil {
		if err := rp.connectInit(); err != nil {
			return false
		}
	}
	if item, err := client.Get(sid); err != nil || len(item.Value) == 0 {
		return false
	}
	return true
}

// SessionRegenerate generate new sid for memcache session
func (rp *MemProvider) SessionRegenerate(oldsid, sid string) (session.Store, error) {
	if client == nil {
		if err := rp.connectInit(); err != nil {
			return nil, err
		}
	}
	var contain []byte
	if item, err := client.Get(sid); err != nil || len(item.Value) == 0 {
		// oldsid doesn't exists, set the new sid directly
		// ignore error here, since if it return error
		// the existed value will be 0
		item.Key = sid
		item.Value = []byte("")
		item.Expiration = int32(rp.maxlifetime)
		client.Set(item)
	} else {
		client.Delete(oldsid)
		item.Key = sid
		item.Expiration = int32(rp.maxlifetime)
		client.Set(item)
		contain = item.Value
	}

	var kv map[interface{}]interface{}
	if len(contain) == 0 {
		kv = make(map[interface{}]interface{})
	} else {
		var err error
		kv, err = session.DecodeGob(contain)
		if err != nil {
			return nil, err
		}
	}

	rs := &SessionStore{sid: sid, values: kv, maxlifetime: rp.maxlifetime}
	return rs, nil
}

// SessionDestroy delete memcache session by id
func (rp *MemProvider) SessionDestroy(sid string) error {
	if client == nil {
		if err := rp.connectInit(); err != nil {
			return err
		}
	}

	err := client.Delete(sid)
	if err != nil {
		return err
	}
	return nil
}

func (rp *MemProvider) connectInit() error {
	client = memcache.New(rp.conninfo...)
	return nil
}

// SessionGC Impelment method, no used.
func (rp *MemProvider) SessionGC() {
	return
}

// SessionAll return all activeSession
func (rp *MemProvider) SessionAll() int {
	return 0
}

func init() {
	session.Register("memcache", mempder)
}

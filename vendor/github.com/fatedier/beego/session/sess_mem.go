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

package session

import (
	"container/list"
	"net/http"
	"sync"
	"time"
)

var mempder = &MemProvider{list: list.New(), sessions: make(map[string]*list.Element)}

// MemSessionStore memory session store.
// it saved sessions in a map in memory.
type MemSessionStore struct {
	sid          string                      //session id
	timeAccessed time.Time                   //last access time
	value        map[interface{}]interface{} //session store
	lock         sync.RWMutex
}

// Set value to memory session
func (st *MemSessionStore) Set(key, value interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.value[key] = value
	return nil
}

// Get value from memory session by key
func (st *MemSessionStore) Get(key interface{}) interface{} {
	st.lock.RLock()
	defer st.lock.RUnlock()
	if v, ok := st.value[key]; ok {
		return v
	}
	return nil
}

// Delete in memory session by key
func (st *MemSessionStore) Delete(key interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	delete(st.value, key)
	return nil
}

// Flush clear all values in memory session
func (st *MemSessionStore) Flush() error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.value = make(map[interface{}]interface{})
	return nil
}

// SessionID get this id of memory session store
func (st *MemSessionStore) SessionID() string {
	return st.sid
}

// SessionRelease Implement method, no used.
func (st *MemSessionStore) SessionRelease(w http.ResponseWriter) {
}

// MemProvider Implement the provider interface
type MemProvider struct {
	lock        sync.RWMutex             // locker
	sessions    map[string]*list.Element // map in memory
	list        *list.List               // for gc
	maxlifetime int64
	savePath    string
}

// SessionInit init memory session
func (pder *MemProvider) SessionInit(maxlifetime int64, savePath string) error {
	pder.maxlifetime = maxlifetime
	pder.savePath = savePath
	return nil
}

// SessionRead get memory session store by sid
func (pder *MemProvider) SessionRead(sid string) (Store, error) {
	pder.lock.RLock()
	if element, ok := pder.sessions[sid]; ok {
		go pder.SessionUpdate(sid)
		pder.lock.RUnlock()
		return element.Value.(*MemSessionStore), nil
	}
	pder.lock.RUnlock()
	pder.lock.Lock()
	newsess := &MemSessionStore{sid: sid, timeAccessed: time.Now(), value: make(map[interface{}]interface{})}
	element := pder.list.PushFront(newsess)
	pder.sessions[sid] = element
	pder.lock.Unlock()
	return newsess, nil
}

// SessionExist check session store exist in memory session by sid
func (pder *MemProvider) SessionExist(sid string) bool {
	pder.lock.RLock()
	defer pder.lock.RUnlock()
	if _, ok := pder.sessions[sid]; ok {
		return true
	}
	return false
}

// SessionRegenerate generate new sid for session store in memory session
func (pder *MemProvider) SessionRegenerate(oldsid, sid string) (Store, error) {
	pder.lock.RLock()
	if element, ok := pder.sessions[oldsid]; ok {
		go pder.SessionUpdate(oldsid)
		pder.lock.RUnlock()
		pder.lock.Lock()
		element.Value.(*MemSessionStore).sid = sid
		pder.sessions[sid] = element
		delete(pder.sessions, oldsid)
		pder.lock.Unlock()
		return element.Value.(*MemSessionStore), nil
	}
	pder.lock.RUnlock()
	pder.lock.Lock()
	newsess := &MemSessionStore{sid: sid, timeAccessed: time.Now(), value: make(map[interface{}]interface{})}
	element := pder.list.PushFront(newsess)
	pder.sessions[sid] = element
	pder.lock.Unlock()
	return newsess, nil
}

// SessionDestroy delete session store in memory session by id
func (pder *MemProvider) SessionDestroy(sid string) error {
	pder.lock.Lock()
	defer pder.lock.Unlock()
	if element, ok := pder.sessions[sid]; ok {
		delete(pder.sessions, sid)
		pder.list.Remove(element)
		return nil
	}
	return nil
}

// SessionGC clean expired session stores in memory session
func (pder *MemProvider) SessionGC() {
	pder.lock.RLock()
	for {
		element := pder.list.Back()
		if element == nil {
			break
		}
		if (element.Value.(*MemSessionStore).timeAccessed.Unix() + pder.maxlifetime) < time.Now().Unix() {
			pder.lock.RUnlock()
			pder.lock.Lock()
			pder.list.Remove(element)
			delete(pder.sessions, element.Value.(*MemSessionStore).sid)
			pder.lock.Unlock()
			pder.lock.RLock()
		} else {
			break
		}
	}
	pder.lock.RUnlock()
}

// SessionAll get count number of memory session
func (pder *MemProvider) SessionAll() int {
	return pder.list.Len()
}

// SessionUpdate expand time of session store by id in memory session
func (pder *MemProvider) SessionUpdate(sid string) error {
	pder.lock.Lock()
	defer pder.lock.Unlock()
	if element, ok := pder.sessions[sid]; ok {
		element.Value.(*MemSessionStore).timeAccessed = time.Now()
		pder.list.MoveToFront(element)
		return nil
	}
	return nil
}

func init() {
	Register("memory", mempder)
}

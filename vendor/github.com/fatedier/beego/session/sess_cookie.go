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
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
)

var cookiepder = &CookieProvider{}

// CookieSessionStore Cookie SessionStore
type CookieSessionStore struct {
	sid    string
	values map[interface{}]interface{} // session data
	lock   sync.RWMutex
}

// Set value to cookie session.
// the value are encoded as gob with hash block string.
func (st *CookieSessionStore) Set(key, value interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.values[key] = value
	return nil
}

// Get value from cookie session
func (st *CookieSessionStore) Get(key interface{}) interface{} {
	st.lock.RLock()
	defer st.lock.RUnlock()
	if v, ok := st.values[key]; ok {
		return v
	}
	return nil
}

// Delete value in cookie session
func (st *CookieSessionStore) Delete(key interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	delete(st.values, key)
	return nil
}

// Flush Clean all values in cookie session
func (st *CookieSessionStore) Flush() error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.values = make(map[interface{}]interface{})
	return nil
}

// SessionID Return id of this cookie session
func (st *CookieSessionStore) SessionID() string {
	return st.sid
}

// SessionRelease Write cookie session to http response cookie
func (st *CookieSessionStore) SessionRelease(w http.ResponseWriter) {
	str, err := encodeCookie(cookiepder.block,
		cookiepder.config.SecurityKey,
		cookiepder.config.SecurityName,
		st.values)
	if err != nil {
		return
	}
	cookie := &http.Cookie{Name: cookiepder.config.CookieName,
		Value:    url.QueryEscape(str),
		Path:     "/",
		HttpOnly: true,
		Secure:   cookiepder.config.Secure,
		MaxAge:   cookiepder.config.Maxage}
	http.SetCookie(w, cookie)
	return
}

type cookieConfig struct {
	SecurityKey  string `json:"securityKey"`
	BlockKey     string `json:"blockKey"`
	SecurityName string `json:"securityName"`
	CookieName   string `json:"cookieName"`
	Secure       bool   `json:"secure"`
	Maxage       int    `json:"maxage"`
}

// CookieProvider Cookie session provider
type CookieProvider struct {
	maxlifetime int64
	config      *cookieConfig
	block       cipher.Block
}

// SessionInit Init cookie session provider with max lifetime and config json.
// maxlifetime is ignored.
// json config:
// 	securityKey - hash string
// 	blockKey - gob encode hash string. it's saved as aes crypto.
// 	securityName - recognized name in encoded cookie string
// 	cookieName - cookie name
// 	maxage - cookie max life time.
func (pder *CookieProvider) SessionInit(maxlifetime int64, config string) error {
	pder.config = &cookieConfig{}
	err := json.Unmarshal([]byte(config), pder.config)
	if err != nil {
		return err
	}
	if pder.config.BlockKey == "" {
		pder.config.BlockKey = string(generateRandomKey(16))
	}
	if pder.config.SecurityName == "" {
		pder.config.SecurityName = string(generateRandomKey(20))
	}
	pder.block, err = aes.NewCipher([]byte(pder.config.BlockKey))
	if err != nil {
		return err
	}
	pder.maxlifetime = maxlifetime
	return nil
}

// SessionRead Get SessionStore in cooke.
// decode cooke string to map and put into SessionStore with sid.
func (pder *CookieProvider) SessionRead(sid string) (Store, error) {
	maps, _ := decodeCookie(pder.block,
		pder.config.SecurityKey,
		pder.config.SecurityName,
		sid, pder.maxlifetime)
	if maps == nil {
		maps = make(map[interface{}]interface{})
	}
	rs := &CookieSessionStore{sid: sid, values: maps}
	return rs, nil
}

// SessionExist Cookie session is always existed
func (pder *CookieProvider) SessionExist(sid string) bool {
	return true
}

// SessionRegenerate Implement method, no used.
func (pder *CookieProvider) SessionRegenerate(oldsid, sid string) (Store, error) {
	return nil, nil
}

// SessionDestroy Implement method, no used.
func (pder *CookieProvider) SessionDestroy(sid string) error {
	return nil
}

// SessionGC Implement method, no used.
func (pder *CookieProvider) SessionGC() {
	return
}

// SessionAll Implement method, return 0.
func (pder *CookieProvider) SessionAll() int {
	return 0
}

// SessionUpdate Implement method, no used.
func (pder *CookieProvider) SessionUpdate(sid string) error {
	return nil
}

func init() {
	Register("cookie", cookiepder)
}

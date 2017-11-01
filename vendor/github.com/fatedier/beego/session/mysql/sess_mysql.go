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

// Package mysql for session provider
//
// depends on github.com/go-sql-driver/mysql:
//
// go install github.com/go-sql-driver/mysql
//
// mysql session support need create table as sql:
//	CREATE TABLE `session` (
//	`session_key` char(64) NOT NULL,
//	`session_data` blob,
//	`session_expiry` int(11) unsigned NOT NULL,
//	PRIMARY KEY (`session_key`)
//	) ENGINE=MyISAM DEFAULT CHARSET=utf8;
//
// Usage:
// import(
//   _ "github.com/astaxie/beego/session/mysql"
//   "github.com/astaxie/beego/session"
// )
//
//	func init() {
//		globalSessions, _ = session.NewManager("mysql", ``{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]"}``)
//		go globalSessions.GC()
//	}
//
// more docs: http://beego.me/docs/module/session.md
package mysql

import (
	"database/sql"
	"net/http"
	"sync"
	"time"

	"github.com/astaxie/beego/session"
	// import mysql driver
	_ "github.com/go-sql-driver/mysql"
)

var (
	// TableName store the session in MySQL
	TableName = "session"
	mysqlpder = &Provider{}
)

// SessionStore mysql session store
type SessionStore struct {
	c      *sql.DB
	sid    string
	lock   sync.RWMutex
	values map[interface{}]interface{}
}

// Set value in mysql session.
// it is temp value in map.
func (st *SessionStore) Set(key, value interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.values[key] = value
	return nil
}

// Get value from mysql session
func (st *SessionStore) Get(key interface{}) interface{} {
	st.lock.RLock()
	defer st.lock.RUnlock()
	if v, ok := st.values[key]; ok {
		return v
	}
	return nil
}

// Delete value in mysql session
func (st *SessionStore) Delete(key interface{}) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	delete(st.values, key)
	return nil
}

// Flush clear all values in mysql session
func (st *SessionStore) Flush() error {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.values = make(map[interface{}]interface{})
	return nil
}

// SessionID get session id of this mysql session store
func (st *SessionStore) SessionID() string {
	return st.sid
}

// SessionRelease save mysql session values to database.
// must call this method to save values to database.
func (st *SessionStore) SessionRelease(w http.ResponseWriter) {
	defer st.c.Close()
	b, err := session.EncodeGob(st.values)
	if err != nil {
		return
	}
	st.c.Exec("UPDATE "+TableName+" set `session_data`=?, `session_expiry`=? where session_key=?",
		b, time.Now().Unix(), st.sid)
}

// Provider mysql session provider
type Provider struct {
	maxlifetime int64
	savePath    string
}

// connect to mysql
func (mp *Provider) connectInit() *sql.DB {
	db, e := sql.Open("mysql", mp.savePath)
	if e != nil {
		return nil
	}
	return db
}

// SessionInit init mysql session.
// savepath is the connection string of mysql.
func (mp *Provider) SessionInit(maxlifetime int64, savePath string) error {
	mp.maxlifetime = maxlifetime
	mp.savePath = savePath
	return nil
}

// SessionRead get mysql session by sid
func (mp *Provider) SessionRead(sid string) (session.Store, error) {
	c := mp.connectInit()
	defer c.Close()
	row := c.QueryRow("select session_data from "+TableName+" where session_key=?", sid)
	var sessiondata []byte
	err := row.Scan(&sessiondata)
	if err == sql.ErrNoRows {
		c.Exec("insert into "+TableName+"(`session_key`,`session_data`,`session_expiry`) values(?,?,?)",
			sid, "", time.Now().Unix())
	}
	var kv map[interface{}]interface{}
	if len(sessiondata) == 0 {
		kv = make(map[interface{}]interface{})
	} else {
		kv, err = session.DecodeGob(sessiondata)
		if err != nil {
			return nil, err
		}
	}
	rs := &SessionStore{c: c, sid: sid, values: kv}
	return rs, nil
}

// SessionExist check mysql session exist
func (mp *Provider) SessionExist(sid string) bool {
	c := mp.connectInit()
	defer c.Close()
	row := c.QueryRow("select session_data from "+TableName+" where session_key=?", sid)
	var sessiondata []byte
	err := row.Scan(&sessiondata)
	if err == sql.ErrNoRows {
		return false
	}
	return true
}

// SessionRegenerate generate new sid for mysql session
func (mp *Provider) SessionRegenerate(oldsid, sid string) (session.Store, error) {
	c := mp.connectInit()
	defer c.Close()
	row := c.QueryRow("select session_data from "+TableName+" where session_key=?", oldsid)
	var sessiondata []byte
	err := row.Scan(&sessiondata)
	if err == sql.ErrNoRows {
		c.Exec("insert into "+TableName+"(`session_key`,`session_data`,`session_expiry`) values(?,?,?)", oldsid, "", time.Now().Unix())
	}
	c.Exec("update "+TableName+" set `session_key`=? where session_key=?", sid, oldsid)
	var kv map[interface{}]interface{}
	if len(sessiondata) == 0 {
		kv = make(map[interface{}]interface{})
	} else {
		kv, err = session.DecodeGob(sessiondata)
		if err != nil {
			return nil, err
		}
	}
	rs := &SessionStore{c: c, sid: sid, values: kv}
	return rs, nil
}

// SessionDestroy delete mysql session by sid
func (mp *Provider) SessionDestroy(sid string) error {
	c := mp.connectInit()
	c.Exec("DELETE FROM "+TableName+" where session_key=?", sid)
	c.Close()
	return nil
}

// SessionGC delete expired values in mysql session
func (mp *Provider) SessionGC() {
	c := mp.connectInit()
	c.Exec("DELETE from "+TableName+" where session_expiry < ?", time.Now().Unix()-mp.maxlifetime)
	c.Close()
	return
}

// SessionAll count values in mysql session
func (mp *Provider) SessionAll() int {
	c := mp.connectInit()
	defer c.Close()
	var total int
	err := c.QueryRow("SELECT count(*) as num from " + TableName).Scan(&total)
	if err != nil {
		return 0
	}
	return total
}

func init() {
	session.Register("mysql", mysqlpder)
}

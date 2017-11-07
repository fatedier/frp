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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMem(t *testing.T) {
	config := `{"cookieName":"gosessionid","gclifetime":10, "enableSetCookie":true}`
	conf := new(ManagerConfig)
	if err := json.Unmarshal([]byte(config), conf); err != nil {
		t.Fatal("json decode error", err)
	}
	globalSessions, _ := NewManager("memory", conf)
	go globalSessions.GC()
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	sess, err := globalSessions.SessionStart(w, r)
	if err != nil {
		t.Fatal("set error,", err)
	}
	defer sess.SessionRelease(w)
	err = sess.Set("username", "astaxie")
	if err != nil {
		t.Fatal("set error,", err)
	}
	if username := sess.Get("username"); username != "astaxie" {
		t.Fatal("get username error")
	}
	if cookiestr := w.Header().Get("Set-Cookie"); cookiestr == "" {
		t.Fatal("setcookie error")
	} else {
		parts := strings.Split(strings.TrimSpace(cookiestr), ";")
		for k, v := range parts {
			nameval := strings.Split(v, "=")
			if k == 0 && nameval[0] != "gosessionid" {
				t.Fatal("error")
			}
		}
	}
}

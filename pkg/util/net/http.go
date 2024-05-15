// Copyright 2017 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package net

import (
	"compress/gzip"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fatedier/frp/pkg/util/util"

	"github.com/fatedier/frp/pkg/util/log"
)

type HTTPAuthMiddleware struct {
	user          string
	passwd        string
	authFailDelay time.Duration

	expires  time.Duration
	sessions map[string]time.Time
}

func NewHTTPAuthMiddleware(user, passwd string) *HTTPAuthMiddleware {
	middleware := &HTTPAuthMiddleware{
		user:   user,
		passwd: passwd,

		expires:  10 * time.Minute,
		sessions: make(map[string]time.Time),
	}
	middleware.cleanSession()
	return middleware
}

func (authMid *HTTPAuthMiddleware) SetAuthFailDelay(delay time.Duration) *HTTPAuthMiddleware {
	authMid.authFailDelay = delay
	return authMid
}

func (authMid *HTTPAuthMiddleware) signIn(w http.ResponseWriter, r *http.Request) bool {
	reqUser, reqPasswd, hasAuth := r.BasicAuth()
	if (authMid.user == "" && authMid.passwd == "") ||
		(hasAuth && util.ConstantTimeEqString(reqUser, authMid.user) &&
			util.ConstantTimeEqString(reqPasswd, authMid.passwd)) {
		sessionToken := uuid.NewString()
		expiresAt := time.Now().Add(authMid.expires)

		authMid.sessions[sessionToken] = expiresAt
		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   sessionToken,
			Expires: expiresAt,
		})
		log.Debugf("signIn success and set cookie %s", sessionToken)
		return true
	} else {
		log.Debugf("signIn fail")
		return false
	}
}

func (authMid *HTTPAuthMiddleware) auth(r *http.Request) bool {
	c, err := r.Cookie("session_token")
	if err != nil {
		log.Debugf("get cookie error: %v", err)
		return false
	}
	_, exists := authMid.sessions[c.Value]
	if exists {
		log.Debugf("exist session %s and refresh it", c.Value)
		authMid.sessions[c.Value] = time.Now().Add(authMid.expires)
	}
	return exists
}

func (authMid *HTTPAuthMiddleware) cleanSession() {
	ticker := time.NewTicker(authMid.expires)
	go func() {
		for {
			<-ticker.C
			log.Debugf("start clean session...")
			for k, v := range authMid.sessions {
				if v.Before(time.Now()) {
					log.Debugf("delete session %s", k)
					delete(authMid.sessions, k)
				}
			}
		}
	}()
}

func (authMid *HTTPAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authMid.auth(r) {
			next.ServeHTTP(w, r)
		} else if authMid.signIn(w, r) {
			next.ServeHTTP(w, r)
		} else {
			if authMid.authFailDelay > 0 {
				time.Sleep(authMid.authFailDelay)
			}
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	})
}

type HTTPGzipWrapper struct {
	h http.Handler
}

func (gw *HTTPGzipWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		gw.h.ServeHTTP(w, r)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
	gw.h.ServeHTTP(gzr, r)
}

func MakeHTTPGzipHandler(h http.Handler) http.Handler {
	return &HTTPGzipWrapper{
		h: h,
	}
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

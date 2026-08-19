// Copyright 2025 The frp Authors
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

package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fatedier/frp/pkg/util/util"
)

const (
	// defaultSessionCookieName is used when no specific name is provided.
	// frps and frpc pass distinct names ("frps_session" / "frpc_session") so
	// that both dashboards can be logged into simultaneously when they run on
	// the same host: cookies are scoped by host name only, not by port.
	defaultSessionCookieName = "frp_session"
	sessionDuration          = 7 * 24 * time.Hour
)

// sessionManager issues and verifies stateless HMAC-signed session tokens.
// Tokens are signed with a random per-process secret, so all sessions become
// invalid after a restart.
type sessionManager struct {
	secret []byte
}

func newSessionManager() (*sessionManager, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	return &sessionManager{secret: secret}, nil
}

func (m *sessionManager) sign(expireAt int64, nonce []byte) string {
	payload := fmt.Sprintf("%d:%s", expireAt, hex.EncodeToString(nonce))
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))
	return fmt.Sprintf("%d.%s.%s", expireAt, hex.EncodeToString(nonce), hex.EncodeToString(mac.Sum(nil)))
}

// issue creates a signed token valid for sessionDuration from now.
func (m *sessionManager) issue() (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return m.sign(time.Now().Add(sessionDuration).Unix(), nonce), nil
}

// verify checks both the signature and the expiry of a token.
func (m *sessionManager) verify(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	expireAt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}
	nonce, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	if time.Now().Unix() >= expireAt {
		return false
	}
	return util.ConstantTimeEqString(token, m.sign(expireAt, nonce))
}

// SessionAuthMiddleware authenticates requests either by a signed session
// cookie (set by the login endpoint, used by the dashboard web UI) or by HTTP
// basic auth (kept for API clients such as the frp SDK).
type SessionAuthMiddleware struct {
	user          string
	passwd        string
	cookieName    string
	sessions      *sessionManager
	authFailDelay time.Duration
}

// NewSessionAuthMiddleware creates the middleware. cookieName scopes the
// session cookie and should be distinct per binary (frps/frpc) to avoid the
// two dashboards overwriting each other's session cookie when served from
// the same host.
func NewSessionAuthMiddleware(user, passwd, cookieName string, sessions *sessionManager) *SessionAuthMiddleware {
	if cookieName == "" {
		cookieName = defaultSessionCookieName
	}
	return &SessionAuthMiddleware{
		user:       user,
		passwd:     passwd,
		cookieName: cookieName,
		sessions:   sessions,
	}
}

func (m *SessionAuthMiddleware) SetAuthFailDelay(delay time.Duration) *SessionAuthMiddleware {
	m.authFailDelay = delay
	return m
}

func (m *SessionAuthMiddleware) authorized(r *http.Request) bool {
	if m.user == "" && m.passwd == "" {
		return true
	}
	reqUser, reqPasswd, hasAuth := r.BasicAuth()
	if hasAuth && util.ConstantTimeEqString(reqUser, m.user) &&
		util.ConstantTimeEqString(reqPasswd, m.passwd) {
		return true
	}
	return m.hasValidSession(r)
}

func (m *SessionAuthMiddleware) hasValidSession(r *http.Request) bool {
	if m.sessions == nil {
		return false
	}
	c, err := r.Cookie(m.cookieName)
	if err != nil || c.Value == "" {
		return false
	}
	return m.sessions.verify(c.Value)
}

func (m *SessionAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.authorized(r) {
			next.ServeHTTP(w, r)
			return
		}
		if m.authFailDelay > 0 {
			time.Sleep(m.authFailDelay)
		}
		// Only send the basic auth challenge to clients that already attempted
		// basic auth or to non-API requests, so that browser fetches from the
		// dashboard SPA handle 401 in JavaScript instead of triggering the
		// native popup.
		_, _, hasAuth := r.BasicAuth()
		if hasAuth || !strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	})
}

type loginRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// LoginHandler validates credentials from a JSON body and issues a session
// cookie on success.
func (m *SessionAuthMiddleware) LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
			m.writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if m.authorizedCredentials(req.User, req.Password) {
			token, err := m.sessions.issue()
			if err != nil {
				m.writeJSONError(w, http.StatusInternalServerError, "internal error")
				return
			}
			http.SetCookie(w, m.newSessionCookie(token, int(sessionDuration/time.Second), r))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(GeneralResponse{Code: http.StatusOK, Msg: "success"})
			return
		}
		if m.authFailDelay > 0 {
			time.Sleep(m.authFailDelay)
		}
		m.writeJSONError(w, http.StatusUnauthorized, "invalid user or password")
	})
}

func (m *SessionAuthMiddleware) authorizedCredentials(user, passwd string) bool {
	if m.user == "" && m.passwd == "" {
		return true
	}
	return util.ConstantTimeEqString(user, m.user) && util.ConstantTimeEqString(passwd, m.passwd)
}

// LogoutHandler clears the session cookie.
func (m *SessionAuthMiddleware) LogoutHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, m.newSessionCookie("", -1, r))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GeneralResponse{Code: http.StatusOK, Msg: "success"})
	})
}

// CheckHandler reports whether the SPA has a login session: 204 if a valid
// session cookie is present, 401 otherwise. It deliberately does NOT accept
// basic auth: browsers silently attach cached basic credentials from earlier
// popup logins, which would resurrect the dashboard after a logout.
func (m *SessionAuthMiddleware) CheckHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (m.user == "" && m.passwd == "") || m.hasValidSession(r) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func (m *SessionAuthMiddleware) newSessionCookie(value string, maxAge int, r *http.Request) *http.Cookie {
	cookie := &http.Cookie{
		Name:     m.cookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
	// Mark the cookie Secure both for direct TLS and for TLS terminated at a
	// reverse proxy in front of frp. A spoofed X-Forwarded-Proto can only
	// force the Secure flag on, which is harmless.
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		cookie.Secure = true
	}
	return cookie
}

func (m *SessionAuthMiddleware) writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(GeneralResponse{Code: code, Msg: msg})
}

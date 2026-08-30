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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

const testCookieName = "frps_session"

func newTestAuthMiddleware(t *testing.T, user, passwd string) *SessionAuthMiddleware {
	t.Helper()
	sessions, err := newSessionManager()
	require.NoError(t, err)
	return NewSessionAuthMiddleware(user, passwd, testCookieName, sessions)
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestSessionManagerIssueAndVerify(t *testing.T) {
	sessions, err := newSessionManager()
	require.NoError(t, err)

	token, err := sessions.issue()
	require.NoError(t, err)
	assert.True(t, sessions.verify(token))

	// Tampered token must fail.
	assert.False(t, sessions.verify(token+"x"))
	assert.False(t, sessions.verify("not-a-token"))
	assert.False(t, sessions.verify(""))

	// Token signed by another secret must fail.
	other, err := newSessionManager()
	require.NoError(t, err)
	assert.False(t, other.verify(token))

	// Expired token must fail.
	nonce := make([]byte, 16)
	expired := sessions.sign(time.Now().Add(-time.Second).Unix(), nonce)
	assert.False(t, sessions.verify(expired))
}

func TestSessionAuthMiddlewareAuthDisabled(t *testing.T) {
	m := newTestAuthMiddleware(t, "", "")
	req := httptest.NewRequest(http.MethodGet, "/api/serverinfo", nil)
	rec := httptest.NewRecorder()
	m.Middleware(okHandler()).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSessionAuthMiddlewareBasicAuth(t *testing.T) {
	m := newTestAuthMiddleware(t, "admin", "secret")

	req := httptest.NewRequest(http.MethodGet, "/api/serverinfo", nil)
	req.SetBasicAuth("admin", "secret")
	rec := httptest.NewRecorder()
	m.Middleware(okHandler()).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Wrong password: 401. Since an Authorization header was sent, the basic
	// auth challenge is included for API clients.
	req = httptest.NewRequest(http.MethodGet, "/api/serverinfo", nil)
	req.SetBasicAuth("admin", "wrong")
	rec = httptest.NewRecorder()
	m.Middleware(okHandler()).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, `Basic realm="Restricted"`, rec.Header().Get("WWW-Authenticate"))
}

func TestSessionAuthMiddlewareSessionCookie(t *testing.T) {
	m := newTestAuthMiddleware(t, "admin", "secret")

	// Obtain a session cookie from the login handler.
	login := m.LoginHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"user":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	login.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, testCookieName, cookies[0].Name)
	assert.True(t, cookies[0].HttpOnly)

	// The cookie must authenticate API requests without basic auth and the
	// response must not carry a challenge.
	req = httptest.NewRequest(http.MethodGet, "/api/serverinfo", nil)
	req.AddCookie(cookies[0])
	rec = httptest.NewRecorder()
	m.Middleware(okHandler()).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSessionAuthMiddlewareUnauthorizedAPI(t *testing.T) {
	m := newTestAuthMiddleware(t, "admin", "secret").SetAuthFailDelay(0)

	// API request without credentials: 401 without the WWW-Authenticate
	// challenge so browser fetches do not trigger the native popup.
	req := httptest.NewRequest(http.MethodGet, "/api/serverinfo", nil)
	rec := httptest.NewRecorder()
	m.Middleware(okHandler()).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, rec.Header().Get("WWW-Authenticate"))

	// Forged cookie must be rejected.
	req = httptest.NewRequest(http.MethodGet, "/api/serverinfo", nil)
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: "1.00.ff"})
	rec = httptest.NewRecorder()
	m.Middleware(okHandler()).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLoginHandler(t *testing.T) {
	m := newTestAuthMiddleware(t, "admin", "secret").SetAuthFailDelay(0)
	login := m.LoginHandler()

	// Bad request body.
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	login.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Wrong credentials.
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"user":"admin","password":"wrong"}`))
	rec = httptest.NewRecorder()
	login.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, rec.Result().Cookies())

	// Correct credentials.
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"user":"admin","password":"secret"}`))
	rec = httptest.NewRecorder()
	login.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, rec.Result().Cookies(), 1)
}

func TestLogoutHandler(t *testing.T) {
	m := newTestAuthMiddleware(t, "admin", "secret")

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	m.LogoutHandler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, testCookieName, cookies[0].Name)
	assert.LessOrEqual(t, cookies[0].MaxAge, 0)
}

func TestSessionCookieSecureBehindTLSProxy(t *testing.T) {
	m := newTestAuthMiddleware(t, "admin", "secret")

	// X-Forwarded-Proto: https from a TLS-terminating reverse proxy must mark
	// the cookie Secure even though the request itself is plain HTTP.
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"user":"admin","password":"secret"}`))
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	m.LoginHandler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.True(t, cookies[0].Secure)

	// Without the header and without TLS the cookie stays non-Secure.
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"user":"admin","password":"secret"}`))
	rec = httptest.NewRecorder()
	m.LoginHandler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, rec.Result().Cookies()[0].Secure)
}

func TestNewServerSessionCookieNameOption(t *testing.T) {
	loginBody := `{"user":"admin","password":"secret"}`

	// Legacy callers without options keep compiling and get the default
	// cookie name.
	s, err := NewServer(v1.WebServerConfig{Addr: "127.0.0.1", Port: 0, User: "admin", Password: "secret"})
	require.NoError(t, err)
	defer s.Close()

	ts := httptest.NewServer(s.router)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/auth/login", "application/json", strings.NewReader(loginBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	cookies := resp.Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, defaultSessionCookieName, cookies[0].Name)

	// Callers that pass WithSessionCookieName get that name instead.
	s2, err := NewServerWithOptions(v1.WebServerConfig{Addr: "127.0.0.1", Port: 0, User: "admin", Password: "secret"},
		WithSessionCookieName("frpc_session"))
	require.NoError(t, err)
	defer s2.Close()

	ts2 := httptest.NewServer(s2.router)
	defer ts2.Close()

	resp2, err := http.Post(ts2.URL+"/api/auth/login", "application/json", strings.NewReader(loginBody))
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	cookies2 := resp2.Cookies()
	require.Len(t, cookies2, 1)
	assert.Equal(t, "frpc_session", cookies2[0].Name)
}

func TestDistinctCookieNamesDoNotCrossAuthenticate(t *testing.T) {
	// Simulate frps and frpc dashboards on the same host: cookies are scoped
	// by host only, so isolation relies on distinct cookie names.
	svr := newTestAuthMiddleware(t, "admin", "secret")
	sessions, err := newSessionManager()
	require.NoError(t, err)
	clt := NewSessionAuthMiddleware("admin", "secret", "frpc_session", sessions)

	// Log in to the "server" and use its cookie on the "client": must fail.
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"user":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	svr.LoginHandler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	apiReq := httptest.NewRequest(http.MethodGet, "/api/serverinfo", nil)
	apiReq.AddCookie(rec.Result().Cookies()[0])
	apiRec := httptest.NewRecorder()
	clt.Middleware(okHandler()).ServeHTTP(apiRec, apiReq)
	assert.Equal(t, http.StatusUnauthorized, apiRec.Code)

	// The server still accepts its own cookie.
	apiRec = httptest.NewRecorder()
	svr.Middleware(okHandler()).ServeHTTP(apiRec, apiReq)
	assert.Equal(t, http.StatusOK, apiRec.Code)
}

func TestCheckHandler(t *testing.T) {
	m := newTestAuthMiddleware(t, "admin", "secret").SetAuthFailDelay(0)
	check := m.CheckHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/check", nil)
	rec := httptest.NewRecorder()
	check.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Login first, then check with the issued cookie.
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"user":"admin","password":"secret"}`))
	loginRec := httptest.NewRecorder()
	m.LoginHandler().ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/auth/check", nil)
	req.AddCookie(loginRec.Result().Cookies()[0])
	rec = httptest.NewRecorder()
	check.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Basic auth must NOT pass the check endpoint: browsers silently attach
	// cached basic credentials, which would resurrect the dashboard after a
	// logout. Basic auth still works for data APIs through the middleware.
	req = httptest.NewRequest(http.MethodGet, "/api/auth/check", nil)
	req.SetBasicAuth("admin", "secret")
	rec = httptest.NewRecorder()
	check.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLoginErrorResponseBody(t *testing.T) {
	m := newTestAuthMiddleware(t, "admin", "secret").SetAuthFailDelay(0)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"user":"admin","password":"wrong"}`))
	rec := httptest.NewRecorder()
	m.LoginHandler().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp GeneralResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.NotEmpty(t, resp.Msg)
}

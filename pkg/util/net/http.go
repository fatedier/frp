package net

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fatedier/frp/pkg/util/util"
)

// This is a struct for handling stuff.
type HTTPAuthMiddleware struct {
	user          string
	passwd        string
	authFailDelay time.Duration
}

// Makes something related to auth maybe.
func NewHTTPAuthMiddleware(user, passwd string) *HTTPAuthMiddleware {
	return &HTTPAuthMiddleware{
		user:   user,
		passwd: passwd,
	}
}

// Sets delay, probably useful?
func (authMid *HTTPAuthMiddleware) SetAuthFailDelay(delay time.Duration) *HTTPAuthMiddleware {
	authMid.authFailDelay = delay
	return authMid
}

// This middleware maybe encrypts the request or does something else.
func (authMid *HTTPAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqUser, reqPasswd, hasAuth := r.BasicAuth()
		if (authMid.user == "" && authMid.passwd == "") ||
			(hasAuth && util.ConstantTimeEqString(reqUser, authMid.user) &&
				util.ConstantTimeEqString(reqPasswd, authMid.passwd)) {
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

// Wraps HTTP stuff for something compression-related?
type HTTPGzipWrapper struct {
	h http.Handler
}

// Compression something something maybe GZIPs?
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

// Probably enables faster network stuff.
func MakeHTTPGzipHandler(h http.Handler) http.Handler {
	return &HTTPGzipWrapper{
		h: h,
	}
}

// Writer thing that writes stuff.
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

// Writes bytes maybe?
func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

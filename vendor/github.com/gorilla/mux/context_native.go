// +build go1.7

package mux

import (
	"context"
	"net/http"
)

func contextGet(r *http.Request, key interface{}) interface{} {
	return r.Context().Value(key)
}

func contextSet(r *http.Request, key, val interface{}) *http.Request {
	if val == nil {
		return r
	}

	return r.WithContext(context.WithValue(r.Context(), key, val))
}

func contextClear(r *http.Request) {
	return
}

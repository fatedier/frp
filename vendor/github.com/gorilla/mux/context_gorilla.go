// +build !go1.7

package mux

import (
	"net/http"

	"github.com/gorilla/context"
)

func contextGet(r *http.Request, key interface{}) interface{} {
	return context.Get(r, key)
}

func contextSet(r *http.Request, key, val interface{}) *http.Request {
	if val == nil {
		return r
	}

	context.Set(r, key, val)
	return r
}

func contextClear(r *http.Request) {
	context.Clear(r)
}

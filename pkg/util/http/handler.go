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

	"github.com/fatedier/frp/pkg/util/log"
)

type GeneralResponse struct {
	Code int
	Msg  string
}

// APIHandler is a handler function that returns a response object or an error.
type APIHandler func(ctx *Context) (any, error)

// MakeHTTPHandlerFunc turns a normal APIHandler into a http.HandlerFunc.
func MakeHTTPHandlerFunc(handler APIHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(w, r)
		res, err := handler(ctx)
		if err != nil {
			log.Warnf("http response [%s]: error: %v", r.URL.Path, err)
			code := http.StatusInternalServerError
			if e, ok := err.(*Error); ok {
				code = e.Code
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			_ = json.NewEncoder(w).Encode(GeneralResponse{Code: code, Msg: err.Error()})
			return
		}

		if res == nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		switch v := res.(type) {
		case []byte:
			_, _ = w.Write(v)
		case string:
			_, _ = w.Write([]byte(v))
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(v)
		}
	}
}

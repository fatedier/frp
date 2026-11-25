// Copyright 2026 The frp Authors
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

package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestHTTPPluginReadError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, `{"unchange":true}`)
	}))
	defer s.Close()
	p := &httpPlugin{url: s.URL, client: s.Client()}
	var res Response
	err := p.do(context.Background(), &Request{Op: "Login"}, &res)
	if !errors.Is(err, io.ErrUnexpectedEOF) || res.Unchange {
		t.Fatalf("expected unexpected EOF before decoding, got unchange=%v err=%v", res.Unchange, err)
	}
}

func TestHTTPPluginResponseLimit(t *testing.T) {
	const limit = 8 << 20
	for _, chunked := range []bool{false, true} {
		for _, size := range []int{17, limit - 1, limit, limit + 1} {
			t.Run(strconv.Itoa(size)+"/chunked="+strconv.FormatBool(chunked), func(t *testing.T) {
				body := `{"unchange":true}` + strings.Repeat(" ", size-17)
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if chunked {
						w.(http.Flusher).Flush()
					} else {
						w.Header().Set("Content-Length", strconv.Itoa(len(body)))
					}
					_, _ = io.WriteString(w, body)
				}))
				defer s.Close()
				p := &httpPlugin{url: s.URL, client: s.Client()}
				var res Response
				err := p.do(context.Background(), &Request{Op: "Login"}, &res)
				if size > limit {
					if err == nil || !strings.Contains(err.Error(), "response body exceeds") {
						t.Fatalf("expected size limit error, got %v", err)
					}
				} else if err != nil || !res.Unchange {
					t.Fatalf("response mismatch: unchange=%v, err=%v", res.Unchange, err)
				}
			})
		}
	}
}

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

package client

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

func TestGetConfigReadError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, "partial")
	}))
	defer s.Close()
	c := &Client{address: strings.TrimPrefix(s.URL, "http://")}
	body, err := c.GetConfig(context.Background())
	if !errors.Is(err, io.ErrUnexpectedEOF) || body != "" {
		t.Fatalf("expected no body and unexpected EOF, got body=%q err=%v", body, err)
	}
}

func TestGetConfigResponseLimit(t *testing.T) {
	const limit = 8 << 20
	for _, chunked := range []bool{false, true} {
		for _, size := range []int{0, 32, limit - 1, limit, limit + 1} {
			t.Run(strconv.Itoa(size)+"/chunked="+strconv.FormatBool(chunked), func(t *testing.T) {
				body := strings.Repeat("x", size)
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if chunked {
						w.(http.Flusher).Flush()
					} else {
						w.Header().Set("Content-Length", strconv.Itoa(size))
					}
					_, _ = io.WriteString(w, body)
				}))
				defer s.Close()
				c := &Client{address: strings.TrimPrefix(s.URL, "http://")}
				got, err := c.GetConfig(context.Background())
				if size > limit {
					if err == nil || !strings.Contains(err.Error(), "response body exceeds") {
						t.Fatalf("expected size limit error, got %v", err)
					}
					if got != "" {
						t.Fatal("oversized response returned partial content")
					}
				} else if err != nil || got != body {
					t.Fatalf("response mismatch: length=%d, err=%v", len(got), err)
				}
			})
		}
	}
}

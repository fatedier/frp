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

package http

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type countingResponseReader struct {
	read int
}

func (r *countingResponseReader) Read(p []byte) (int, error) {
	clear(p)
	r.read += len(p)
	return len(p), nil
}

func TestReadResponseBodyStopsReading(t *testing.T) {
	r := &countingResponseReader{}
	body, err := ReadResponseBody(r)
	if err == nil || !strings.Contains(err.Error(), "response body exceeds") {
		t.Fatalf("expected size limit error, got %v", err)
	}
	if body != nil {
		t.Fatal("oversized response returned partial content")
	}
	if r.read != MaxResponseBodySize+1 {
		t.Fatalf("read %d bytes, want %d", r.read, MaxResponseBodySize+1)
	}
}

type failingResponseReader struct {
	err error
}

func (r failingResponseReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestReadResponseBodyReadError(t *testing.T) {
	want := errors.New("response read failed")
	r := io.MultiReader(strings.NewReader("partial"), failingResponseReader{err: want})
	body, err := ReadResponseBody(r)
	if !errors.Is(err, want) || body != nil {
		t.Fatalf("expected no body and read error, got body=%q err=%v", body, err)
	}
}

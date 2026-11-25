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
	"fmt"
	"io"
)

// MaxResponseBodySize is the maximum buffered response size for the admin SDK
// and HTTP server plugins. Responses larger than 8 MiB are rejected, including
// chunked responses. This limit does not apply to streaming proxy traffic.
const MaxResponseBodySize = 8 << 20

// ReadResponseBody reads a response without buffering an unbounded body.
// The caller remains responsible for closing body.
func ReadResponseBody(body io.Reader) ([]byte, error) {
	buf, err := io.ReadAll(io.LimitReader(body, MaxResponseBodySize+1))
	if err != nil {
		return nil, err
	}
	if len(buf) > MaxResponseBodySize {
		return nil, fmt.Errorf("response body exceeds %d bytes", MaxResponseBodySize)
	}
	return buf, nil
}

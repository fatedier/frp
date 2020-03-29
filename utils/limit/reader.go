// Copyright 2019 fatedier, fatedier@gmail.com
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

package limit

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

type Reader struct {
	r       io.Reader
	limiter *rate.Limiter
}

func NewReader(r io.Reader, limiter *rate.Limiter) *Reader {
	return &Reader{
		r:       r,
		limiter: limiter,
	}
}

func (r *Reader) Read(p []byte) (n int, err error) {
	b := r.limiter.Burst()
	if b < len(p) {
		p = p[:b]
	}
	n, err = r.r.Read(p)
	if err != nil {
		return
	}

	err = r.limiter.WaitN(context.Background(), n)
	if err != nil {
		return
	}
	return
}

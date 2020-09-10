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

type Writer struct {
	w       io.Writer
	limiter *rate.Limiter
}

func NewWriter(w io.Writer, limiter *rate.Limiter) *Writer {
	return &Writer{
		w:       w,
		limiter: limiter,
	}
}

func (w *Writer) Write(p []byte) (n int, err error) {
	var nn int
	b := w.limiter.Burst()
	for {
		end := len(p)
		if end == 0 {
			break
		}
		if b < len(p) {
			end = b
		}
		err = w.limiter.WaitN(context.Background(), end)
		if err != nil {
			return
		}

		nn, err = w.w.Write(p[:end])
		n += nn
		if err != nil {
			return
		}
		p = p[end:]
	}
	return
}

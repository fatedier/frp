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

package net

import (
	"io"
)

type prependRWC struct {
	under io.ReadWriteCloser
	buf   []byte
}

func WrapPrependReadWriteCloser(under io.ReadWriteCloser, pre []byte) io.ReadWriteCloser {
	if len(pre) == 0 {
		return under
	}
	return &prependRWC{under: under, buf: append([]byte(nil), pre...)}
}

func (p *prependRWC) Read(b []byte) (int, error) {
	if len(p.buf) > 0 {
		n := copy(b, p.buf)
		p.buf = p.buf[n:]
		return n, nil
	}
	return p.under.Read(b)
}

func (p *prependRWC) Write(b []byte) (int, error) {

	return p.under.Write(b)
}

func (p *prependRWC) Close() error {
	return p.under.Close()
}

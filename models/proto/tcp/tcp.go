// Copyright 2017 fatedier, fatedier@gmail.com
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

package tcp

import (
	"io"

	"github.com/golang/snappy"

	"github.com/fatedier/frp/utils/crypto"
)

func WithEncryption(rwc io.ReadWriteCloser, key []byte) (io.ReadWriteCloser, error) {
	w, err := crypto.NewWriter(rwc, key)
	if err != nil {
		return nil, err
	}
	return WrapReadWriteCloser(crypto.NewReader(rwc, key), w), nil
}

func WithCompression(rwc io.ReadWriteCloser) io.ReadWriteCloser {
	return WrapReadWriteCloser(snappy.NewReader(rwc), snappy.NewWriter(rwc))
}

func WrapReadWriteCloser(r io.Reader, w io.Writer) io.ReadWriteCloser {
	return &ReadWriteCloser{
		r: r,
		w: w,
	}
}

type ReadWriteCloser struct {
	r io.Reader
	w io.Writer
}

func (rwc *ReadWriteCloser) Read(p []byte) (n int, err error) {
	return rwc.r.Read(p)
}

func (rwc *ReadWriteCloser) Write(p []byte) (n int, err error) {
	return rwc.w.Write(p)
}

func (rwc *ReadWriteCloser) Close() (errRet error) {
	var err error
	if rc, ok := rwc.r.(io.Closer); ok {
		err = rc.Close()
		if err != nil {
			errRet = err
		}
	}

	if wc, ok := rwc.w.(io.Closer); ok {
		err = wc.Close()
		if err != nil {
			errRet = err
		}
	}
	return
}

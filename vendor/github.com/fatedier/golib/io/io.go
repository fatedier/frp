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

package io

import (
	"io"
	"sync"

	"github.com/fatedier/golib/crypto"
	"github.com/fatedier/golib/pool"
)

// Join two io.ReadWriteCloser and do some operations.
func Join(c1 io.ReadWriteCloser, c2 io.ReadWriteCloser) (inCount int64, outCount int64) {
	var wait sync.WaitGroup
	pipe := func(to io.ReadWriteCloser, from io.ReadWriteCloser, count *int64) {
		defer to.Close()
		defer from.Close()
		defer wait.Done()

		buf := pool.GetBuf(16 * 1024)
		defer pool.PutBuf(buf)
		*count, _ = io.CopyBuffer(to, from, buf)
	}

	wait.Add(2)
	go pipe(c1, c2, &inCount)
	go pipe(c2, c1, &outCount)
	wait.Wait()
	return
}

func WithEncryption(rwc io.ReadWriteCloser, key []byte) (io.ReadWriteCloser, error) {
	w, err := crypto.NewWriter(rwc, key)
	if err != nil {
		return nil, err
	}
	return WrapReadWriteCloser(crypto.NewReader(rwc, key), w, func() error {
		return rwc.Close()
	}), nil
}

func WithCompression(rwc io.ReadWriteCloser) io.ReadWriteCloser {
	sr := pool.GetSnappyReader(rwc)
	sw := pool.GetSnappyWriter(rwc)
	return WrapReadWriteCloser(sr, sw, func() error {
		err := rwc.Close()
		pool.PutSnappyReader(sr)
		pool.PutSnappyWriter(sw)
		return err
	})
}

type ReadWriteCloser struct {
	r       io.Reader
	w       io.Writer
	closeFn func() error

	closed bool
	mu     sync.Mutex
}

// closeFn will be called only once
func WrapReadWriteCloser(r io.Reader, w io.Writer, closeFn func() error) io.ReadWriteCloser {
	return &ReadWriteCloser{
		r:       r,
		w:       w,
		closeFn: closeFn,
		closed:  false,
	}
}

func (rwc *ReadWriteCloser) Read(p []byte) (n int, err error) {
	return rwc.r.Read(p)
}

func (rwc *ReadWriteCloser) Write(p []byte) (n int, err error) {
	return rwc.w.Write(p)
}

func (rwc *ReadWriteCloser) Close() (errRet error) {
	rwc.mu.Lock()
	if rwc.closed {
		rwc.mu.Unlock()
		return
	}
	rwc.closed = true
	rwc.mu.Unlock()

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

	if rwc.closeFn != nil {
		err = rwc.closeFn()
		if err != nil {
			errRet = err
		}
	}
	return
}

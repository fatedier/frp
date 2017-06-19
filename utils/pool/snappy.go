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

package pool

import (
	"io"
	"sync"

	"github.com/golang/snappy"
)

var (
	snappyReaderPool sync.Pool
	snappyWriterPool sync.Pool
)

func GetSnappyReader(r io.Reader) *snappy.Reader {
	var x interface{}
	x = snappyReaderPool.Get()
	if x == nil {
		return snappy.NewReader(r)
	}
	sr := x.(*snappy.Reader)
	sr.Reset(r)
	return sr
}

func PutSnappyReader(sr *snappy.Reader) {
	snappyReaderPool.Put(sr)
}

func GetSnappyWriter(w io.Writer) *snappy.Writer {
	var x interface{}
	x = snappyWriterPool.Get()
	if x == nil {
		return snappy.NewWriter(w)
	}
	sw := x.(*snappy.Writer)
	sw.Reset(w)
	return sw
}

func PutSnappyWriter(sw *snappy.Writer) {
	snappyWriterPool.Put(sw)
}

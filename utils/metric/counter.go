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

package metric

import (
	"sync/atomic"
)

type Counter interface {
	Count() int64
	Inc(int64)
	Dec(int64)
	Snapshot() Counter
	Clear()
}

func NewCounter() Counter {
	return &StandardCounter{
		count: 0,
	}
}

type StandardCounter struct {
	count int64
}

func (c *StandardCounter) Count() int64 {
	return atomic.LoadInt64(&c.count)
}

func (c *StandardCounter) Inc(count int64) {
	atomic.AddInt64(&c.count, count)
}

func (c *StandardCounter) Dec(count int64) {
	atomic.AddInt64(&c.count, -count)
}

func (c *StandardCounter) Snapshot() Counter {
	tmp := &StandardCounter{
		count: atomic.LoadInt64(&c.count),
	}
	return tmp
}

func (c *StandardCounter) Clear() {
	atomic.StoreInt64(&c.count, 0)
}

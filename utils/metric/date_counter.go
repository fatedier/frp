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
	"sync"
	"time"
)

type DateCounter interface {
	TodayCount() int64
	GetLastDaysCount(lastdays int64) []int64
	Inc(int64)
	Dec(int64)
	Snapshot() DateCounter
	Clear()
}

func NewDateCounter(reserveDays int64) DateCounter {
	if reserveDays <= 0 {
		reserveDays = 1
	}
	return newStandardDateCounter(reserveDays)
}

type StandardDateCounter struct {
	reserveDays int64
	counts      []int64

	lastUpdateDate time.Time
	mu             sync.Mutex
}

func newStandardDateCounter(reserveDays int64) *StandardDateCounter {
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	s := &StandardDateCounter{
		reserveDays:    reserveDays,
		counts:         make([]int64, reserveDays),
		lastUpdateDate: now,
	}
	return s
}

func (c *StandardDateCounter) TodayCount() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rotate(time.Now())
	return c.counts[0]
}

func (c *StandardDateCounter) GetLastDaysCount(lastdays int64) []int64 {
	if lastdays > c.reserveDays {
		lastdays = c.reserveDays
	}
	counts := make([]int64, lastdays)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.rotate(time.Now())
	for i := 0; i < int(lastdays); i++ {
		counts[i] = c.counts[i]
	}
	return counts
}

func (c *StandardDateCounter) Inc(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rotate(time.Now())
	c.counts[0] += count
}

func (c *StandardDateCounter) Dec(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rotate(time.Now())
	c.counts[0] -= count
}

func (c *StandardDateCounter) Snapshot() DateCounter {
	c.mu.Lock()
	defer c.mu.Unlock()
	tmp := newStandardDateCounter(c.reserveDays)
	for i := 0; i < int(c.reserveDays); i++ {
		tmp.counts[i] = c.counts[i]
	}
	return tmp
}

func (c *StandardDateCounter) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := 0; i < int(c.reserveDays); i++ {
		c.counts[i] = 0
	}
}

// rotate
// Must hold the lock before calling this function.
func (c *StandardDateCounter) rotate(now time.Time) {
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(now.Sub(c.lastUpdateDate).Hours() / 24)

	defer func() {
		c.lastUpdateDate = now
	}()

	if days <= 0 {
		return
	} else if days >= int(c.reserveDays) {
		c.counts = make([]int64, c.reserveDays)
		return
	}
	newCounts := make([]int64, c.reserveDays)

	for i := days; i < int(c.reserveDays); i++ {
		newCounts[i] = c.counts[i-days]
	}
	c.counts = newCounts
}

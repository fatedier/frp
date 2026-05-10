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

	"k8s.io/utils/clock"
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
	clock       clock.PassiveClock

	lastUpdateDate time.Time
	mu             sync.Mutex
}

func newStandardDateCounter(reserveDays int64) *StandardDateCounter {
	return newStandardDateCounterWithClock(reserveDays, clock.RealClock{})
}

func newStandardDateCounterWithClock(reserveDays int64, clk clock.PassiveClock) *StandardDateCounter {
	if clk == nil {
		clk = clock.RealClock{}
	}
	return &StandardDateCounter{
		reserveDays:    reserveDays,
		counts:         make([]int64, reserveDays),
		clock:          clk,
		lastUpdateDate: startOfDay(clk.Now()),
	}
}

func (c *StandardDateCounter) TodayCount() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rotate(c.clock.Now())
	return c.counts[0]
}

func (c *StandardDateCounter) GetLastDaysCount(lastdays int64) []int64 {
	if lastdays > c.reserveDays {
		lastdays = c.reserveDays
	}
	counts := make([]int64, lastdays)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.rotate(c.clock.Now())
	copy(counts, c.counts)
	return counts
}

func (c *StandardDateCounter) Inc(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rotate(c.clock.Now())
	c.counts[0] += count
}

func (c *StandardDateCounter) Dec(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rotate(c.clock.Now())
	c.counts[0] -= count
}

func (c *StandardDateCounter) Snapshot() DateCounter {
	c.mu.Lock()
	defer c.mu.Unlock()
	tmp := newStandardDateCounterWithClock(c.reserveDays, c.clock)
	copy(tmp.counts, c.counts)
	return tmp
}

func (c *StandardDateCounter) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.counts)
}

// rotate
// Must hold the lock before calling this function.
func (c *StandardDateCounter) rotate(now time.Time) {
	now = startOfDay(now)
	days := int(now.Sub(c.lastUpdateDate).Hours() / 24)
	reserveDays := int(c.reserveDays)

	if days <= 0 {
		return
	} else if days >= reserveDays {
		c.counts = make([]int64, c.reserveDays)
		c.lastUpdateDate = now
		return
	}
	newCounts := make([]int64, c.reserveDays)

	copy(newCounts[days:], c.counts[:reserveDays-days])
	c.counts = newCounts
	c.lastUpdateDate = now
}

// startOfDay returns midnight in t's location.
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

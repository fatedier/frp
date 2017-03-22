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
	Close()
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

	closeCh chan struct{}
	closed  bool
	mu      sync.Mutex
}

func newStandardDateCounter(reserveDays int64) *StandardDateCounter {
	s := &StandardDateCounter{
		reserveDays: reserveDays,
		counts:      make([]int64, reserveDays),
		closeCh:     make(chan struct{}),
	}
	s.startRotateWorker()
	return s
}

func (c *StandardDateCounter) TodayCount() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[0]
}

func (c *StandardDateCounter) GetLastDaysCount(lastdays int64) []int64 {
	if lastdays > c.reserveDays {
		lastdays = c.reserveDays
	}
	counts := make([]int64, lastdays)

	c.mu.Lock()
	defer c.mu.Unlock()
	for i := 0; i < int(lastdays); i++ {
		counts[i] = c.counts[i]
	}
	return counts
}

func (c *StandardDateCounter) Inc(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[0] += count
}

func (c *StandardDateCounter) Dec(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[0] -= count
}

func (c *StandardDateCounter) Snapshot() DateCounter {
	c.mu.Lock()
	defer c.mu.Unlock()
	tmp := &StandardDateCounter{
		reserveDays: c.reserveDays,
		counts:      make([]int64, c.reserveDays),
	}
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

func (c *StandardDateCounter) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		close(c.closeCh)
		c.closed = true
	}
}

func (c *StandardDateCounter) rotate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	newCounts := make([]int64, c.reserveDays)

	for i := 1; i < int(c.reserveDays-1); i++ {
		newCounts[i] = c.counts[i+1]
	}
	c.counts = newCounts
}

func (c *StandardDateCounter) startRotateWorker() {
	now := time.Now()
	nextDayTimeStr := now.Add(24 * time.Hour).Format("20060102")
	nextDay, _ := time.Parse("20060102", nextDayTimeStr)
	d := nextDay.Sub(now)

	firstTimer := time.NewTimer(d)
	rotateTicker := time.NewTicker(24 * time.Hour)

	go func() {
		for {
			select {
			case <-firstTimer.C:
				firstTimer.Stop()
				rotateTicker.Stop()
				rotateTicker = time.NewTicker(24 * time.Hour)
				c.rotate()
			case <-rotateTicker.C:
				c.rotate()
			case <-c.closeCh:
				break
			}
		}
		firstTimer.Stop()
		rotateTicker.Stop()
	}()
}

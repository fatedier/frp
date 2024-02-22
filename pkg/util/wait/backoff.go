// Copyright 2023 The frp Authors
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

package wait

import (
	"math/rand"
	"time"

	"github.com/fatedier/frp/pkg/util/util"
)

type BackoffFunc func(previousDuration time.Duration, previousConditionError bool) time.Duration

func (f BackoffFunc) Backoff(previousDuration time.Duration, previousConditionError bool) time.Duration {
	return f(previousDuration, previousConditionError)
}

type BackoffManager interface {
	Backoff(previousDuration time.Duration, previousConditionError bool) time.Duration
}

type FastBackoffOptions struct {
	Duration           time.Duration
	Factor             float64
	Jitter             float64
	MaxDuration        time.Duration
	InitDurationIfFail time.Duration

	// If FastRetryCount > 0, then within the FastRetryWindow time window,
	// the retry will be performed with a delay of FastRetryDelay for the first FastRetryCount calls.
	FastRetryCount  int
	FastRetryDelay  time.Duration
	FastRetryJitter float64
	FastRetryWindow time.Duration
}

type fastBackoffImpl struct {
	options FastBackoffOptions

	lastCalledTime      time.Time
	consecutiveErrCount int

	fastRetryCutoffTime     time.Time
	countsInFastRetryWindow int
}

func NewFastBackoffManager(options FastBackoffOptions) BackoffManager {
	return &fastBackoffImpl{
		options:                 options,
		countsInFastRetryWindow: 1,
	}
}

func (f *fastBackoffImpl) Backoff(previousDuration time.Duration, previousConditionError bool) time.Duration {
	if f.lastCalledTime.IsZero() {
		f.lastCalledTime = time.Now()
		return f.options.Duration
	}
	now := time.Now()
	f.lastCalledTime = now

	if previousConditionError {
		f.consecutiveErrCount++
	} else {
		f.consecutiveErrCount = 0
	}

	if f.options.FastRetryCount > 0 && previousConditionError {
		f.countsInFastRetryWindow++
		if f.countsInFastRetryWindow <= f.options.FastRetryCount {
			return Jitter(f.options.FastRetryDelay, f.options.FastRetryJitter)
		}
		if now.After(f.fastRetryCutoffTime) {
			// reset
			f.fastRetryCutoffTime = now.Add(f.options.FastRetryWindow)
			f.countsInFastRetryWindow = 0
		}
	}

	if previousConditionError {
		var duration time.Duration
		if f.consecutiveErrCount == 1 {
			duration = util.EmptyOr(f.options.InitDurationIfFail, previousDuration)
		} else {
			duration = previousDuration
		}

		duration = util.EmptyOr(duration, time.Second)
		if f.options.Factor != 0 {
			duration = time.Duration(float64(duration) * f.options.Factor)
		}
		if f.options.Jitter > 0 {
			duration = Jitter(duration, f.options.Jitter)
		}
		if f.options.MaxDuration > 0 && duration > f.options.MaxDuration {
			duration = f.options.MaxDuration
		}
		return duration
	}
	return f.options.Duration
}

func BackoffUntil(f func() (bool, error), backoff BackoffManager, sliding bool, stopCh <-chan struct{}) {
	var delay time.Duration
	previousError := false

	ticker := time.NewTicker(backoff.Backoff(delay, previousError))
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		if !sliding {
			delay = backoff.Backoff(delay, previousError)
		}

		if done, err := f(); done {
			return
		} else if err != nil {
			previousError = true
		} else {
			previousError = false
		}

		if sliding {
			delay = backoff.Backoff(delay, previousError)
		}

		ticker.Reset(delay)
		select {
		case <-stopCh:
			return
		case <-ticker.C:
		}
	}
}

// Jitter returns a time.Duration between duration and duration + maxFactor *
// duration.
//
// This allows clients to avoid converging on periodic behavior. If maxFactor
// is 0.0, a suggested default value will be chosen.
func Jitter(duration time.Duration, maxFactor float64) time.Duration {
	if maxFactor <= 0.0 {
		maxFactor = 1.0
	}
	wait := duration + time.Duration(rand.Float64()*maxFactor*float64(duration))
	return wait
}

func Until(f func(), period time.Duration, stopCh <-chan struct{}) {
	ff := func() (bool, error) {
		f()
		return false, nil
	}
	BackoffUntil(ff, BackoffFunc(func(time.Duration, bool) time.Duration {
		return period
	}), true, stopCh)
}

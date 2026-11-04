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

package xlog

import (
	"sync"
	"testing"
)

func TestLoggerConcurrentPrefixAccess(t *testing.T) {
	captureLogs(t)
	xl := New().AddPrefix(LogPrefix{Name: "runID", Value: "old-run"})
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		xl.AddPrefix(LogPrefix{Name: "runID", Value: "new-run"})
		xl.AppendPrefix("proxy")
		xl.ResetPrefixes()
	}()
	go func() {
		defer wg.Done()
		<-start
		xl.Spawn().Tracef("spawn")
		xl.Errorf("error")
		xl.Warnf("warn")
		xl.Infof("info")
		xl.Debugf("debug")
		xl.Tracef("trace")
	}()
	close(start)
	wg.Wait()
}

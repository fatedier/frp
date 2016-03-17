// Copyright 2016 fatedier, fatedier@gmail.com
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

package broadcast

import (
	"sync"
	"testing"
	"time"
)

var (
	totalNum int = 5
	succNum  int = 0
	mutex    sync.Mutex
)

func TestBroadcast(t *testing.T) {
	b := NewBroadcast()
	if b == nil {
		t.Fatalf("New Broadcast error, nil return")
	}
	defer b.Close()

	var wait sync.WaitGroup
	wait.Add(totalNum)
	for i := 0; i < totalNum; i++ {
		go worker(b, &wait)
	}

	time.Sleep(1e6 * 20)
	msg := "test"
	b.In() <- msg

	wait.Wait()
	if succNum != totalNum {
		t.Fatalf("TotalNum %d, FailNum(timeout) %d", totalNum, totalNum-succNum)
	}
}

func worker(b *Broadcast, wait *sync.WaitGroup) {
	defer wait.Done()
	msgChan := b.Reg()

	// exit if nothing got in 2 seconds
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Duration(2) * time.Second)
		timeout <- true
	}()

	select {
	case item := <-msgChan:
		msg := item.(string)
		if msg == "test" {
			mutex.Lock()
			succNum++
			mutex.Unlock()
		} else {
			break
		}

	case <-timeout:
		break
	}
}

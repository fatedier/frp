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
		t.Errorf("New Broadcast error, nil return")
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
		t.Errorf("TotalNum %d, FailNum(timeout) %d", totalNum, totalNum-succNum)
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

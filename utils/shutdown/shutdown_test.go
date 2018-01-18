package shutdown

import (
	"testing"
	"time"
)

func TestShutdown(t *testing.T) {
	s := New()
	go func() {
		time.Sleep(time.Millisecond)
		s.Start()
	}()
	s.WaitStart()

	go func() {
		time.Sleep(time.Millisecond)
		s.Done()
	}()
	s.WaitDone()
}

package group

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLn is a controllable net.Listener for tests.
type fakeLn struct {
	connCh chan net.Conn
	closed chan struct{}
	once   sync.Once
}

func newFakeLn() *fakeLn {
	return &fakeLn{
		connCh: make(chan net.Conn, 8),
		closed: make(chan struct{}),
	}
}

func (f *fakeLn) Accept() (net.Conn, error) {
	select {
	case c := <-f.connCh:
		return c, nil
	case <-f.closed:
		return nil, net.ErrClosed
	}
}

func (f *fakeLn) Close() error {
	f.once.Do(func() { close(f.closed) })
	return nil
}

func (f *fakeLn) Addr() net.Addr { return fakeAddr("127.0.0.1:9999") }

func (f *fakeLn) inject(c net.Conn) {
	select {
	case f.connCh <- c:
	case <-f.closed:
	}
}

func TestBaseGroup_WorkerFanOut(t *testing.T) {
	fl := newFakeLn()
	var bg baseGroup
	bg.initBase("g", "key", fl, func() {})

	go bg.worker(fl, bg.acceptCh)

	c1, c2 := net.Pipe()
	defer c2.Close()
	fl.inject(c1)

	select {
	case got := <-bg.acceptCh:
		assert.Equal(t, c1, got)
		got.Close()
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for connection on acceptCh")
	}

	fl.Close()
}

func TestBaseGroup_WorkerStopsOnListenerClose(t *testing.T) {
	fl := newFakeLn()
	var bg baseGroup
	bg.initBase("g", "key", fl, func() {})

	done := make(chan struct{})
	go func() {
		bg.worker(fl, bg.acceptCh)
		close(done)
	}()

	fl.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after listener close")
	}
}

func TestBaseGroup_WorkerClosesConnOnClosedChannel(t *testing.T) {
	fl := newFakeLn()
	var bg baseGroup
	bg.initBase("g", "key", fl, func() {})

	// Close acceptCh before worker sends.
	close(bg.acceptCh)

	done := make(chan struct{})
	go func() {
		bg.worker(fl, bg.acceptCh)
		close(done)
	}()

	c1, c2 := net.Pipe()
	defer c2.Close()
	fl.inject(c1)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after panic recovery")
	}

	// c1 should have been closed by worker's panic recovery path.
	buf := make([]byte, 1)
	_, err := c1.Read(buf)
	assert.Error(t, err, "connection should be closed by worker")
}

func TestBaseGroup_CloseLastListenerTriggersCleanup(t *testing.T) {
	fl := newFakeLn()
	var bg baseGroup
	cleanupCalled := 0
	bg.initBase("g", "key", fl, func() { cleanupCalled++ })

	bg.mu.Lock()
	ln1 := bg.newListener(fl.Addr())
	ln2 := bg.newListener(fl.Addr())
	bg.mu.Unlock()

	go bg.worker(fl, bg.acceptCh)

	ln1.Close()
	assert.Equal(t, 0, cleanupCalled, "cleanup should not run while listeners remain")

	ln2.Close()
	assert.Equal(t, 1, cleanupCalled, "cleanup should run after last listener closed")
}

func TestBaseGroup_CloseOneOfTwoListeners(t *testing.T) {
	fl := newFakeLn()
	var bg baseGroup
	cleanupCalled := 0
	bg.initBase("g", "key", fl, func() { cleanupCalled++ })

	bg.mu.Lock()
	ln1 := bg.newListener(fl.Addr())
	ln2 := bg.newListener(fl.Addr())
	bg.mu.Unlock()

	go bg.worker(fl, bg.acceptCh)

	ln1.Close()
	assert.Equal(t, 0, cleanupCalled)

	// ln2 should still receive connections.
	c1, c2 := net.Pipe()
	defer c2.Close()
	fl.inject(c1)

	got, err := ln2.Accept()
	require.NoError(t, err)
	assert.Equal(t, c1, got)
	got.Close()

	ln2.Close()
	assert.Equal(t, 1, cleanupCalled)
}

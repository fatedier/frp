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

package msg

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	libcrypto "github.com/fatedier/golib/crypto"
)

var (
	errRegressionWriteTimeout = errors.New("injected recoverable connection write timeout")
	regressionKey             = []byte("isolated-public-test-key")
)

// regressionTransport fails exactly one Write, then accepts later writes. This
// models an individual mux stream write timing out while its session survives.
type regressionTransport struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	calls    int
	failures int
	failNext bool
	closed   chan struct{}
	once     sync.Once
}

func (w *regressionTransport) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls++
	if w.failNext {
		w.failNext = false
		w.failures++
		return 0, errRegressionWriteTimeout
	}
	select {
	case <-w.closed:
		return 0, io.ErrClosedPipe
	default:
	}
	return w.buffer.Write(p)
}

func (w *regressionTransport) arm() {
	w.mu.Lock()
	w.failNext = true
	w.mu.Unlock()
}

func (w *regressionTransport) snapshot() (calls, failures, size int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.calls, w.failures, w.buffer.Len()
}

func (w *regressionTransport) close() {
	w.once.Do(func() { close(w.closed) })
}

// regressionControl uses the production FRP message encoder and production
// golib crypto.Writer. Only the final transport and incoming messages are fake.
type regressionControl struct {
	w           *libcrypto.Writer
	transport   *regressionTransport
	writes      chan error
	incoming    chan Message
	beforeRead  func() error
	beforeWrite func()
	closeCalls  atomic.Int32
}

func newRegressionControl(t *testing.T) *regressionControl {
	t.Helper()
	transport := &regressionTransport{closed: make(chan struct{})}
	w, err := libcrypto.NewWriter(transport, regressionKey)
	if err != nil {
		t.Fatal(err)
	}
	return &regressionControl{
		w: w, transport: transport, writes: make(chan error, 100), incoming: make(chan Message, 1),
	}
}

func (c *regressionControl) ReadMsg() (Message, error) {
	if c.beforeRead != nil {
		if err := c.beforeRead(); err != nil {
			return nil, err
		}
	}
	select {
	case m := <-c.incoming:
		return m, nil
	case <-c.transport.closed:
		return nil, io.EOF
	}
}

func (c *regressionControl) ReadMsgInto(Message) error {
	_, err := c.ReadMsg()
	return err
}

func (c *regressionControl) WriteMsg(m Message) error {
	if c.beforeWrite != nil {
		c.beforeWrite()
	}
	err := WriteMsg(c.w, m)
	c.writes <- err
	return err
}

func (c *regressionControl) Close() error {
	c.closeCalls.Add(1)
	c.transport.close()
	return nil
}

func regressionReceiveWrite(t *testing.T, c *regressionControl) error {
	t.Helper()
	select {
	case err := <-c.writes:
		return err
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not attempt WriteMsg")
		return nil
	}
}

func regressionAwait(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func regressionWarm(t *testing.T, c *regressionControl) {
	t.Helper()
	if err := c.WriteMsg(&ReqWorkConn{}); err != nil {
		t.Fatal(err)
	}
	<-c.writes
	c.transport.mu.Lock()
	raw := append([]byte(nil), c.transport.buffer.Bytes()...)
	c.transport.mu.Unlock()
	m, err := ReadMsg(libcrypto.NewReader(bytes.NewReader(raw), regressionKey))
	if err != nil {
		t.Fatalf("encrypted control roundtrip: %v", err)
	}
	if _, ok := m.(*ReqWorkConn); !ok {
		t.Fatalf("decoded %T", m)
	}
}

func TestControlCryptoWriterKeepsErrorAfterTransportRecovery(t *testing.T) {
	c := newRegressionControl(t)
	regressionWarm(t, c)
	defer c.transport.close()
	c.transport.arm()
	if err := c.WriteMsg(&ReqWorkConn{}); !errors.Is(err, errRegressionWriteTimeout) {
		t.Fatalf("want timeout, got %v", err)
	}
	<-c.writes
	beforeCalls, failures, beforeBytes := c.transport.snapshot()
	for range 5 {
		if err := c.WriteMsg(&ReqWorkConn{}); !errors.Is(err, errRegressionWriteTimeout) {
			t.Fatalf("sticky error missing: %v", err)
		}
		<-c.writes
	}
	afterCalls, _, afterBytes := c.transport.snapshot()
	if beforeCalls != afterCalls || beforeBytes != afterBytes || failures != 1 {
		t.Fatalf("unexpected retry: calls %d -> %d, failures %d, bytes %d -> %d",
			beforeCalls, afterCalls, failures, beforeBytes, afterBytes)
	}
	if n, err := c.transport.Write([]byte("healthy underlying transport")); err != nil || n == 0 {
		t.Fatalf("underlying transport did not recover: %v", err)
	}
	t.Log("one recoverable write error permanently invalidates this crypto writer; recovered underlying transport is never retried")
}

func TestDispatcherWriteErrorClosesControlAndFreshSessionWorks(t *testing.T) {
	c := newRegressionControl(t)
	regressionWarm(t, c)
	defer c.transport.close()
	d := NewDispatcher(c)
	d.Run()
	c.transport.arm()
	if err := d.Send(&ReqWorkConn{}); err != nil {
		t.Fatal(err)
	}
	if err := regressionReceiveWrite(t, c); !errors.Is(err, errRegressionWriteTimeout) {
		t.Fatal(err)
	}
	regressionAwait(t, c.transport.closed, "failed control connection close")
	regressionAwait(t, d.Done(), "failed dispatcher completion")
	for range 100 {
		if err := d.Send(&ReqWorkConn{}); !errors.Is(err, io.EOF) {
			t.Fatalf("Send after completed shutdown: %v", err)
		}
	}
	if c.closeCalls.Load() != 1 {
		t.Fatalf("connection closed %d times", c.closeCalls.Load())
	}

	fresh := newRegressionControl(t)
	defer fresh.transport.close()
	d2 := NewDispatcher(fresh)
	d2.Run()
	for range 5 {
		if err := d2.Send(&ReqWorkConn{}); err != nil {
			t.Fatal(err)
		}
		if err := regressionReceiveWrite(t, fresh); err != nil {
			t.Fatalf("fresh session write failed: %v", err)
		}
	}
	fresh.transport.mu.Lock()
	raw := append([]byte(nil), fresh.transport.buffer.Bytes()...)
	fresh.transport.mu.Unlock()
	r := libcrypto.NewReader(bytes.NewReader(raw), regressionKey)
	for i := range 5 {
		m, err := ReadMsg(r)
		if err != nil {
			t.Fatalf("new stream decode %d: %v", i, err)
		}
		if _, ok := m.(*ReqWorkConn); !ok {
			t.Fatalf("decoded %T", m)
		}
	}
	fresh.transport.close()
	regressionAwait(t, d2.Done(), "new-session cleanup")
	t.Log("write error closes failed control; new session sends and decrypts five valid ReqWorkConn messages")
}

func TestDispatcherWriteErrorWaitsForSynchronousHandler(t *testing.T) {
	c := newRegressionControl(t)
	regressionWarm(t, c)
	defer c.transport.close()
	d := NewDispatcher(c)
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	d.RegisterHandler(&ReqWorkConn{}, func(Message) {
		close(entered)
		<-release
	})
	d.Run()
	c.incoming <- &ReqWorkConn{}
	regressionAwait(t, entered, "synchronous handler entry")
	c.transport.arm()
	if err := d.Send(&ReqWorkConn{}); err != nil {
		t.Fatal(err)
	}
	if err := regressionReceiveWrite(t, c); !errors.Is(err, errRegressionWriteTimeout) {
		t.Fatal(err)
	}
	regressionAwait(t, c.transport.closed, "write failure closes control during handler")
	for range 100 {
		if err := d.Send(&ReqWorkConn{}); !errors.Is(err, io.EOF) {
			t.Fatalf("Send after stop while handler is active: %v", err)
		}
	}
	select {
	case <-d.Done():
		t.Fatal("Done closed before synchronous handler returned; owner cleanup could race proxy registration")
	case <-time.After(100 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(release) })
	regressionAwait(t, d.Done(), "completion after handler returned")
}

func TestDispatcherWriteErrorUnblocksSenderWithFullQueue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newRegressionControl(t)
		regressionWarm(t, c)
		defer c.transport.close()
		entered, release := make(chan struct{}), make(chan struct{})
		var releaseOnce sync.Once
		defer releaseOnce.Do(func() { close(release) })
		c.beforeWrite = func() { close(entered); <-release }
		c.transport.arm()
		d := NewDispatcher(c)
		if err := d.Send(&ReqWorkConn{}); err != nil {
			t.Fatal(err)
		}
		d.Run()
		regressionAwait(t, entered, "blocked transport write")
		for i := 0; i < cap(d.sendCh); i++ {
			if err := d.Send(&ReqWorkConn{}); err != nil {
				t.Fatal(err)
			}
		}
		sent := make(chan error, 1)
		go func() { sent <- d.Send(&ReqWorkConn{}) }()
		// Wait until the sender has reached its full-queue channel wait.
		synctest.Wait()
		select {
		case err := <-sent:
			t.Fatalf("Send did not block on full queue: %v", err)
		default:
		}
		releaseOnce.Do(func() { close(release) })
		select {
		case err := <-sent:
			if !errors.Is(err, io.EOF) {
				t.Fatalf("blocked Send after write failure: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("write failure did not unblock full-queue sender")
		}
		regressionAwait(t, d.Done(), "full-queue dispatcher cleanup")
	})
}

func TestDispatcherConcurrentReadWriteFailureClosesOnce(t *testing.T) {
	for range 100 {
		func() {
			c := newRegressionControl(t)
			regressionWarm(t, c)
			defer c.transport.close()
			readEntered, writeEntered := make(chan struct{}), make(chan struct{})
			release := make(chan struct{})
			var releaseOnce sync.Once
			defer releaseOnce.Do(func() { close(release) })
			c.beforeRead = func() error {
				close(readEntered)
				<-release
				return io.EOF
			}
			c.beforeWrite = func() {
				close(writeEntered)
				<-release
			}
			d := NewDispatcher(c)
			c.transport.arm()
			if err := d.Send(&ReqWorkConn{}); err != nil {
				t.Fatal(err)
			}
			readExited, writeExited := make(chan struct{}), make(chan struct{})
			go func() {
				defer close(readExited)
				d.readLoop()
			}()
			go func() {
				defer close(writeExited)
				d.sendLoop()
			}()
			regressionAwait(t, readEntered, "read path before injected failure")
			regressionAwait(t, writeEntered, "write path before injected failure")
			releaseOnce.Do(func() { close(release) })
			regressionAwait(t, readExited, "failed read loop exit")
			regressionAwait(t, writeExited, "failed write loop exit")
			regressionAwait(t, d.Done(), "concurrent read/write shutdown")
			if err := regressionReceiveWrite(t, c); !errors.Is(err, errRegressionWriteTimeout) {
				t.Fatalf("concurrent WriteMsg failure: %v", err)
			}
			if _, failures, _ := c.transport.snapshot(); failures != 1 {
				t.Fatalf("injected transport failed %d times", failures)
			}
			if c.closeCalls.Load() != 1 {
				t.Fatalf("connection closed %d times", c.closeCalls.Load())
			}
		}()
	}
}

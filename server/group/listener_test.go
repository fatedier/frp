package group

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListener_Accept(t *testing.T) {
	acceptCh := make(chan net.Conn, 1)
	ln := newListener(acceptCh, fakeAddr("127.0.0.1:1234"), func(*Listener) {})

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	acceptCh <- c1
	got, err := ln.Accept()
	require.NoError(t, err)
	assert.Equal(t, c1, got)
}

func TestListener_AcceptAfterChannelClose(t *testing.T) {
	acceptCh := make(chan net.Conn)
	ln := newListener(acceptCh, fakeAddr("127.0.0.1:1234"), func(*Listener) {})

	close(acceptCh)
	_, err := ln.Accept()
	assert.ErrorIs(t, err, ErrListenerClosed)
}

func TestListener_AcceptAfterListenerClose(t *testing.T) {
	acceptCh := make(chan net.Conn) // open, not closed
	ln := newListener(acceptCh, fakeAddr("127.0.0.1:1234"), func(*Listener) {})

	ln.Close()
	_, err := ln.Accept()
	assert.ErrorIs(t, err, ErrListenerClosed)
}

func TestListener_DoubleClose(t *testing.T) {
	closeCalls := 0
	ln := newListener(
		make(chan net.Conn),
		fakeAddr("127.0.0.1:1234"),
		func(*Listener) { closeCalls++ },
	)

	assert.NotPanics(t, func() {
		ln.Close()
		ln.Close()
	})
	assert.Equal(t, 1, closeCalls, "onClose should be called exactly once")
}

func TestListener_Addr(t *testing.T) {
	addr := fakeAddr("10.0.0.1:5555")
	ln := newListener(make(chan net.Conn), addr, func(*Listener) {})
	assert.Equal(t, addr, ln.Addr())
}

// fakeAddr implements net.Addr for testing.
type fakeAddr string

func (a fakeAddr) Network() string { return "tcp" }
func (a fakeAddr) String() string  { return string(a) }

package yamux

import (
	"testing"
)

func TestAsyncSendErr(t *testing.T) {
	ch := make(chan error)
	asyncSendErr(ch, ErrTimeout)
	select {
	case <-ch:
		t.Fatalf("should not get")
	default:
	}

	ch = make(chan error, 1)
	asyncSendErr(ch, ErrTimeout)
	select {
	case <-ch:
	default:
		t.Fatalf("should get")
	}
}

func TestAsyncNotify(t *testing.T) {
	ch := make(chan struct{})
	asyncNotify(ch)
	select {
	case <-ch:
		t.Fatalf("should not get")
	default:
	}

	ch = make(chan struct{}, 1)
	asyncNotify(ch)
	select {
	case <-ch:
	default:
		t.Fatalf("should get")
	}
}

func TestMin(t *testing.T) {
	if min(1, 2) != 1 {
		t.Fatalf("bad")
	}
	if min(2, 1) != 1 {
		t.Fatalf("bad")
	}
}

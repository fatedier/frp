package group

import (
	"net"
	"sync"
)

// Listener is a per-proxy virtual listener that receives connections
// from a shared group. It implements net.Listener.
type Listener struct {
	acceptCh <-chan net.Conn
	addr     net.Addr
	closeCh  chan struct{}
	onClose  func(*Listener)
	once     sync.Once
}

func newListener(acceptCh <-chan net.Conn, addr net.Addr, onClose func(*Listener)) *Listener {
	return &Listener{
		acceptCh: acceptCh,
		addr:     addr,
		closeCh:  make(chan struct{}),
		onClose:  onClose,
	}
}

func (ln *Listener) Accept() (net.Conn, error) {
	select {
	case <-ln.closeCh:
		return nil, ErrListenerClosed
	case c, ok := <-ln.acceptCh:
		if !ok {
			return nil, ErrListenerClosed
		}
		return c, nil
	}
}

func (ln *Listener) Addr() net.Addr {
	return ln.addr
}

func (ln *Listener) Close() error {
	ln.once.Do(func() {
		close(ln.closeCh)
		ln.onClose(ln)
	})
	return nil
}

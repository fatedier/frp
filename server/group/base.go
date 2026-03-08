package group

import (
	"net"
	"sync"

	gerr "github.com/fatedier/golib/errors"
)

// baseGroup contains the shared plumbing for listener-based groups
// (TCP, HTTPS, TCPMux). Each concrete group embeds this and provides
// its own Listen method with protocol-specific validation.
type baseGroup struct {
	group    string
	groupKey string

	acceptCh  chan net.Conn
	realLn    net.Listener
	lns       []*Listener
	mu        sync.Mutex
	cleanupFn func()
}

// initBase resets the baseGroup for a fresh listen cycle.
// Must be called under mu when len(lns) == 0.
func (bg *baseGroup) initBase(group, groupKey string, realLn net.Listener, cleanupFn func()) {
	bg.group = group
	bg.groupKey = groupKey
	bg.realLn = realLn
	bg.acceptCh = make(chan net.Conn)
	bg.cleanupFn = cleanupFn
}

// worker reads from the real listener and fans out to acceptCh.
// The parameters are captured at creation time so that the worker is
// bound to a specific listen cycle and cannot observe a later initBase.
func (bg *baseGroup) worker(realLn net.Listener, acceptCh chan<- net.Conn) {
	for {
		c, err := realLn.Accept()
		if err != nil {
			return
		}
		err = gerr.PanicToError(func() {
			acceptCh <- c
		})
		if err != nil {
			c.Close()
			return
		}
	}
}

// newListener creates a new Listener wired to this baseGroup.
// Must be called under mu.
func (bg *baseGroup) newListener(addr net.Addr) *Listener {
	ln := newListener(bg.acceptCh, addr, bg.closeListener)
	bg.lns = append(bg.lns, ln)
	return ln
}

// closeListener removes ln from the list. When the last listener is removed,
// it closes acceptCh, closes the real listener, and calls cleanupFn.
func (bg *baseGroup) closeListener(ln *Listener) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	for i, l := range bg.lns {
		if l == ln {
			bg.lns = append(bg.lns[:i], bg.lns[i+1:]...)
			break
		}
	}
	if len(bg.lns) == 0 {
		close(bg.acceptCh)
		bg.realLn.Close()
		bg.cleanupFn()
	}
}

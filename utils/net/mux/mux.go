package mux

import (
	"fmt"
	"io"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/fatedier/frp/utils/errors"
	frpNet "github.com/fatedier/frp/utils/net"
)

const (
	// DefaultTimeout is the default length of time to wait for bytes we need.
	DefaultTimeout = 10 * time.Second
)

type Mux struct {
	ln net.Listener

	defaultLn       *listener
	lns             []*listener
	maxNeedBytesNum uint32
	mu              sync.RWMutex
}

func NewMux() (mux *Mux) {
	mux = &Mux{
		lns: make([]*listener, 0),
	}
	return
}

func (mux *Mux) Listen(priority int, needBytesNum uint32, fn MatchFunc) net.Listener {
	ln := &listener{
		c:            make(chan net.Conn),
		mux:          mux,
		needBytesNum: needBytesNum,
		matchFn:      fn,
	}

	mux.mu.Lock()
	defer mux.mu.Unlock()
	if needBytesNum > mux.maxNeedBytesNum {
		mux.maxNeedBytesNum = needBytesNum
	}

	newlns := append(mux.copyLns(), ln)
	sort.Slice(newlns, func(i, j int) bool {
		return newlns[i].needBytesNum < newlns[j].needBytesNum
	})
	mux.lns = newlns
	return ln
}

func (mux *Mux) ListenHttp(priority int) net.Listener {
	return mux.Listen(priority, HttpNeedBytesNum, HttpMatchFunc)
}

func (mux *Mux) ListenHttps(priority int) net.Listener {
	return mux.Listen(priority, HttpsNeedBytesNum, HttpsMatchFunc)
}

func (mux *Mux) DefaultListener() net.Listener {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	if mux.defaultLn == nil {
		mux.defaultLn = &listener{
			c:   make(chan net.Conn),
			mux: mux,
		}
	}
	return mux.defaultLn
}

func (mux *Mux) release(ln *listener) bool {
	result := false
	mux.mu.Lock()
	defer mux.mu.Unlock()
	lns := mux.copyLns()

	for i, l := range lns {
		if l == ln {
			lns = append(lns[:i], lns[i+1:]...)
			result = true
		}
	}
	mux.lns = lns
	return result
}

func (mux *Mux) copyLns() []*listener {
	lns := make([]*listener, 0, len(mux.lns))
	for _, l := range mux.lns {
		lns = append(lns, l)
	}
	return lns
}

// Serve handles connections from ln and multiplexes then across registered listeners.
func (mux *Mux) Serve(ln net.Listener) error {
	mux.mu.Lock()
	mux.ln = ln
	mux.mu.Unlock()
	for {
		// Wait for the next connection.
		// If it returns a temporary error then simply retry.
		// If it returns any other error then exit immediately.
		conn, err := ln.Accept()
		if err, ok := err.(interface {
			Temporary() bool
		}); ok && err.Temporary() {
			continue
		}

		if err != nil {
			return err
		}

		go mux.handleConn(conn)
	}
}

func (mux *Mux) handleConn(conn net.Conn) {
	mux.mu.RLock()
	maxNeedBytesNum := mux.maxNeedBytesNum
	lns := mux.lns
	defaultLn := mux.defaultLn
	mux.mu.RUnlock()

	shareConn, rd := frpNet.NewShareConnSize(frpNet.WrapConn(conn), int(maxNeedBytesNum))
	data := make([]byte, maxNeedBytesNum)

	conn.SetReadDeadline(time.Now().Add(DefaultTimeout))
	_, err := io.ReadFull(rd, data)
	if err != nil {
		conn.Close()
		return
	}
	conn.SetReadDeadline(time.Time{})

	for _, ln := range lns {
		if match := ln.matchFn(data); match {
			err = errors.PanicToError(func() {
				ln.c <- shareConn
			})
			if err != nil {
				conn.Close()
			}
			return
		}
	}

	// No match listeners
	if defaultLn != nil {
		err = errors.PanicToError(func() {
			defaultLn.c <- shareConn
		})
		if err != nil {
			conn.Close()
		}
		return
	}

	// No listeners for this connection, close it.
	conn.Close()
	return
}

type listener struct {
	mux *Mux

	needBytesNum uint32
	matchFn      MatchFunc

	c  chan net.Conn
	mu sync.RWMutex
}

// Accept waits for and returns the next connection to the listener.
func (ln *listener) Accept() (net.Conn, error) {
	conn, ok := <-ln.c
	if !ok {
		return nil, fmt.Errorf("network connection closed")
	}
	return conn, nil
}

// Close removes this listener from the parent mux and closes the channel.
func (ln *listener) Close() error {
	if ok := ln.mux.release(ln); ok {
		// Close done to signal to any RLock holders to release their lock.
		close(ln.c)
	}
	return nil
}

func (ln *listener) Addr() net.Addr {
	if ln.mux == nil {
		return nil
	}
	ln.mux.mu.RLock()
	defer ln.mux.mu.RUnlock()
	if ln.mux.ln == nil {
		return nil
	}
	return ln.mux.ln.Addr()
}

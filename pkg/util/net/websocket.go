package net

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	gerr "github.com/fatedier/golib/errors"
	"golang.org/x/net/websocket"
)

var ErrWebsocketListenerClosed = errors.New("websocket listener closed")

const (
	FrpWebsocketPath = "/~!frp"
)

type WebsocketListener struct {
	ln       net.Listener
	acceptCh chan net.Conn

	server *http.Server

	mu     sync.Mutex
	closed bool
}

// NewWebsocketListener to handle websocket connections
// ln: tcp listener for websocket connections
func NewWebsocketListener(ln net.Listener) (wl *WebsocketListener) {
	wl = &WebsocketListener{
		ln:       ln,
		acceptCh: make(chan net.Conn),
	}

	muxer := http.NewServeMux()
	muxer.Handle(FrpWebsocketPath, websocket.Handler(func(c *websocket.Conn) {
		// The tunnel payload is a raw byte stream (yamux), not UTF-8 text.
		// Send it as binary frames; otherwise RFC 6455-compliant intermediaries
		// (e.g. API gateways/reverse proxies) UTF-8-validate the default text
		// frames and close the connection on invalid bytes.
		c.PayloadType = websocket.BinaryFrame
		notifyCh := make(chan struct{})
		conn := WrapCloseNotifyConn(c, func(_ error) {
			close(notifyCh)
		})
		// The listener may be closed while this connection is being handed
		// over, so the send has to tolerate a closed channel. A nil error
		// means the connection was accepted and is owned by the caller.
		if err := gerr.PanicToError(func() {
			wl.acceptCh <- conn
		}); err != nil {
			conn.Close()
			return
		}
		<-notifyCh
	}))

	wl.server = &http.Server{
		Addr:              ln.Addr().String(),
		Handler:           muxer,
		ReadHeaderTimeout: 60 * time.Second,
	}

	go func() {
		_ = wl.server.Serve(ln)
	}()
	return
}

func (p *WebsocketListener) Accept() (net.Conn, error) {
	c, ok := <-p.acceptCh
	if !ok {
		return nil, ErrWebsocketListenerClosed
	}
	return c, nil
}

func (p *WebsocketListener) Close() error {
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		// Closing acceptCh releases a pending Accept, matching the other
		// listeners in this package. Otherwise it blocks forever and the
		// caller never learns that the listener is closed.
		close(p.acceptCh)
	}
	p.mu.Unlock()
	return p.server.Close()
}

func (p *WebsocketListener) Addr() net.Addr {
	return p.ln.Addr()
}

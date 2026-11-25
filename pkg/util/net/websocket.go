package net

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

var ErrWebsocketListenerClosed = errors.New("websocket listener closed")

const (
	FrpWebsocketPath = "/~!frp"
)

type WebsocketListener struct {
	ln       net.Listener
	acceptCh chan net.Conn
	done     chan struct{}

	server *http.Server

	closeOnce sync.Once
}

// NewWebsocketListener to handle websocket connections
// ln: tcp listener for websocket connections
func NewWebsocketListener(ln net.Listener) (wl *WebsocketListener) {
	wl = &WebsocketListener{
		ln:       ln,
		acceptCh: make(chan net.Conn),
		done:     make(chan struct{}),
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
		select {
		case <-wl.done:
			conn.Close()
			return
		case wl.acceptCh <- conn:
			// A ready handoff may win when Close races with it. Once delivered,
			// the connection is owned by the caller of Accept.
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
	select {
	case <-p.done:
		return nil, ErrWebsocketListenerClosed
	default:
	}
	select {
	case <-p.done:
		return nil, ErrWebsocketListenerClosed
	case c := <-p.acceptCh:
		return c, nil
	}
}

func (p *WebsocketListener) Close() error {
	p.closeOnce.Do(func() { close(p.done) })
	return p.server.Close()
}

func (p *WebsocketListener) Addr() net.Addr {
	return p.ln.Addr()
}

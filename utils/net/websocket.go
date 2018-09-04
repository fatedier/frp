package net

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/fatedier/frp/utils/log"

	"golang.org/x/net/websocket"
)

var (
	ErrWebsocketListenerClosed = errors.New("websocket listener closed")
)

const (
	FrpWebsocketPath = "/~!frp"
)

type WebsocketListener struct {
	net.Addr
	ln     net.Listener
	accept chan Conn
	log.Logger

	server    *http.Server
	httpMutex *http.ServeMux
}

// ln: tcp listener for websocket connections
func NewWebsocketListener(ln net.Listener) (wl *WebsocketListener) {
	wl = &WebsocketListener{
		Addr:   ln.Addr(),
		accept: make(chan Conn),
		Logger: log.NewPrefixLogger(""),
	}

	muxer := http.NewServeMux()
	muxer.Handle(FrpWebsocketPath, websocket.Handler(func(c *websocket.Conn) {
		notifyCh := make(chan struct{})
		conn := WrapCloseNotifyConn(c, func() {
			close(notifyCh)
		})
		wl.accept <- conn
		<-notifyCh
	}))

	wl.server = &http.Server{
		Addr:    ln.Addr().String(),
		Handler: muxer,
	}

	go wl.server.Serve(ln)
	return
}

func ListenWebsocket(bindAddr string, bindPort int) (*WebsocketListener, error) {
	tcpLn, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bindAddr, bindPort))
	if err != nil {
		return nil, err
	}
	l := NewWebsocketListener(tcpLn)
	return l, nil
}

func (p *WebsocketListener) Accept() (Conn, error) {
	c, ok := <-p.accept
	if !ok {
		return nil, ErrWebsocketListenerClosed
	}
	return c, nil
}

func (p *WebsocketListener) Close() error {
	return p.server.Close()
}

// addr: domain:port
func ConnectWebsocketServer(addr string) (Conn, error) {
	addr = "ws://" + addr + FrpWebsocketPath
	uri, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	origin := "http://" + uri.Host
	cfg, err := websocket.NewConfig(addr, origin)
	if err != nil {
		return nil, err
	}
	cfg.Dialer = &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := websocket.DialConfig(cfg)
	if err != nil {
		return nil, err
	}
	c := WrapConn(conn)
	return c, nil
}

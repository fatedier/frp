package net

import (
	// "crypto/tls"
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
	ErrWssListenerClosed = errors.New("wss listener closed")
)

type WssListener struct {
	net.Addr
	ln     net.Listener
	accept chan Conn
	log.Logger

	server    *http.Server
	httpMutex *http.ServeMux
}

// NewWssListener to handle wss connections
// ln: tcp listener for wss connections
func NewWssListener(ln net.Listener) (wl *WssListener) {
	wl = &WssListener{
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
	certFile := "a_cert.pem"
	keyFile := "a_key.pem"

	go wl.server.ServeTLS(ln, certFile, keyFile)
	return
}

func ListenWss(bindAddr string, bindPort int) (*WssListener, error) {
	tcpLn, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bindAddr, bindPort))
	if err != nil {
		return nil, err
	}
	l := NewWssListener(tcpLn)
	return l, nil
}

func (p *WssListener) Accept() (Conn, error) {
	c, ok := <-p.accept
	if !ok {
		return nil, ErrWssListenerClosed
	}
	return c, nil
}

func (p *WssListener) Close() error {
	return p.server.Close()
}

// addr: domain:port
func ConnectWssServer(addr string, httpProtocol string, wsProtocol string) (Conn, error) {
	addr = wsProtocol + "://" + addr + FrpWebsocketPath
	uri, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	origin := httpProtocol + "://" + uri.Host
	cfg, err := websocket.NewConfig(addr, origin)
	if err != nil {
		return nil, err
	}
	cfg.Dialer = &net.Dialer{
		Timeout: 10 * time.Second,
	}

	// cfg.TlsConfig = &tls.Config{
	// 	InsecureSkipVerify: true,
	// }

	conn, err := websocket.DialConfig(cfg)
	if err != nil {
		return nil, err
	}
	c := WrapConn(conn)
	return c, nil
}

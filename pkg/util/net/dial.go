package net

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"time"

	gnet "github.com/fatedier/golib/net"
	kcp "github.com/fatedier/kcp-go"
	"golang.org/x/net/websocket"
)

type dialOptions struct {
	proxyURL                 string
	protocol                 string
	laddr                    string
	addr                     string
	tlsConfig                *tls.Config
	disableCustomTLSHeadByte bool
}

// DialOption configures how we set up the connection.
type DialOption interface {
	apply(*dialOptions)
}

type EmptyDialOption struct{}

func (EmptyDialOption) apply(*dialOptions) {}

type funcDialOption struct {
	f func(*dialOptions)
}

func (fdo *funcDialOption) apply(do *dialOptions) {
	fdo.f(do)
}

func newFuncDialOption(f func(*dialOptions)) *funcDialOption {
	return &funcDialOption{
		f: f,
	}
}

func DefaultDialOptions() dialOptions {
	return dialOptions{
		protocol: "tcp",
	}
}

func WithProxy(proxyURL string) DialOption {
	return newFuncDialOption(func(do *dialOptions) {
		do.proxyURL = proxyURL
	})
}

func WithBindAddress(laddr string) DialOption {
	return newFuncDialOption(func(do *dialOptions) {
		do.laddr = laddr
	})
}

func WithTLSConfig(tlsConfig *tls.Config) DialOption {
	return newFuncDialOption(func(do *dialOptions) {
		do.tlsConfig = tlsConfig
	})
}

func WithRemoteAddress(addr string) DialOption {
	return newFuncDialOption(func(do *dialOptions) {
		do.addr = addr
	})
}

func WithDisableCustomTLSHeadByte(disableCustomTLSHeadByte bool) DialOption {
	return newFuncDialOption(func(do *dialOptions) {
		do.disableCustomTLSHeadByte = disableCustomTLSHeadByte
	})
}

func WithProtocol(protocol string) DialOption {
	return newFuncDialOption(func(do *dialOptions) {
		do.protocol = protocol
	})
}

func Dial(opts ...DialOption) (c net.Conn, err error) {
	op := DefaultDialOptions()

	for _, opt := range opts {
		opt.apply(&op)
	}

	c, err = dialServer(op.proxyURL, op.protocol, op.addr)
	if err != nil {
		return
	}

	if op.tlsConfig == nil {
		return
	}

	c = WrapTLSClientConn(c, op.tlsConfig, op.disableCustomTLSHeadByte)
	return
}

func dialServer(proxyURL string, protocol string, addr string) (c net.Conn, err error) {
	var d = &net.Dialer{}

	switch protocol {
	case "tcp":
		return gnet.DialTcpByProxy(d, proxyURL, addr)
	case "kcp":
		return DialKCPServer(addr)
	case "websocket":
		return DialWebsocketServer(d, addr)
	default:
		return nil, fmt.Errorf("unsupport protocol: %s", protocol)
	}
}

func DialKCPServer(addr string) (net.Conn, error) {
	// http proxy is not supported for kcp
	kcpConn, errRet := kcp.DialWithOptions(addr, nil, 10, 3)
	if errRet != nil {
		return nil, errRet
	}
	kcpConn.SetStreamMode(true)
	kcpConn.SetWriteDelay(true)
	kcpConn.SetNoDelay(1, 20, 2, 1)
	kcpConn.SetWindowSize(128, 512)
	kcpConn.SetMtu(1350)
	kcpConn.SetACKNoDelay(false)
	kcpConn.SetReadBuffer(4194304)
	kcpConn.SetWriteBuffer(4194304)

	return kcpConn, nil
}

// addr: domain:port
func DialWebsocketServer(d *net.Dialer, addr string) (net.Conn, error) {
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

	cfg.Dialer = d
	cfg.Dialer.Timeout = 10 * time.Second

	conn, err := websocket.DialConfig(cfg)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

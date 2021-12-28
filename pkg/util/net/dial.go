package net

import (
	"crypto/tls"
	"net"
)

type dialOptions struct {
	proxyURL                 string
	protocol                 string
	tlsConfig                *tls.Config
	disableCustomTLSHeadByte bool
}

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

func WithProxyURL(proxyURL string) DialOption {
	return newFuncDialOption(func(do *dialOptions) {
		do.proxyURL = proxyURL
	})
}

func WithTLSConfig(tlsConfig *tls.Config) DialOption {
	return newFuncDialOption(func(do *dialOptions) {
		do.tlsConfig = tlsConfig
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

func DialWithOptions(addr string, opts ...DialOption) (c net.Conn, err error) {
	op := DefaultDialOptions()

	for _, opt := range opts {
		opt.apply(&op)
	}

	if op.proxyURL == "" {
		c, err = ConnectServer(op.protocol, addr)
	} else {
		c, err = ConnectServerByProxy(op.proxyURL, op.protocol, addr)
	}
	if err != nil {
		return nil, err
	}

	if op.tlsConfig == nil {
		return
	}

	c = WrapTLSClientConn(c, op.tlsConfig, op.disableCustomTLSHeadByte)
	return
}

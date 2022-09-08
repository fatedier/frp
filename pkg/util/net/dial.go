package net

import (
	"context"
	"net"
	"net/url"

	libdial "github.com/fatedier/golib/net/dial"
	"golang.org/x/net/websocket"
)

func DialHookCustomTLSHeadByte(enableTLS bool, disableCustomTLSHeadByte bool) libdial.AfterHookFunc {
	return func(ctx context.Context, c net.Conn, addr string) (context.Context, net.Conn, error) {
		if enableTLS && !disableCustomTLSHeadByte {
			_, err := c.Write([]byte{byte(FRPTLSHeadByte)})
			if err != nil {
				return nil, nil, err
			}
		}
		return ctx, c, nil
	}
}

func DialHookWebsocket(isSecure bool) libdial.AfterHookFunc {
	return func(ctx context.Context, c net.Conn, addr string) (context.Context, net.Conn, error) {
		addrScheme := "ws"
		originScheme := "http"
		if isSecure {
			addrScheme = "wss"
			originScheme = "https"
		}
		addr = addrScheme + "://" + addr + FrpWebsocketPath
		uri, err := url.Parse(addr)
		if err != nil {
			return nil, nil, err
		}

		origin := originScheme + "://" + uri.Host
		cfg, err := websocket.NewConfig(addr, origin)
		if err != nil {
			return nil, nil, err
		}

		conn, err := websocket.NewClient(cfg, c)
		if err != nil {
			return nil, nil, err
		}
		return ctx, conn, nil
	}
}

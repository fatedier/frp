package vhost

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	httppkg "github.com/fatedier/frp/pkg/util/http"
)

func TestHTTPServerProtocols(t *testing.T) {
	rp := NewHTTPReverseProxy(HTTPReverseProxyOptions{}, NewRouters())
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	server := &http.Server{
		Handler:           rp,
		ReadHeaderTimeout: time.Second,
		Protocols:         protocols,
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()
	defer func() {
		require.NoError(t, server.Close())
		require.ErrorIs(t, <-serveErr, http.ErrServerClosed)
	}()

	require.True(t, server.Protocols.HTTP1())
	require.True(t, server.Protocols.UnencryptedHTTP2())

	t.Run("HTTP/1.1", func(t *testing.T) {
		transport := &http.Transport{Protocols: httpProtocols(true, false)}
		defer transport.CloseIdleConnections()
		client := &http.Client{Transport: transport}
		response, err := client.Get("http://" + listener.Addr().String() + "/")
		require.NoError(t, err)
		defer response.Body.Close()

		require.Equal(t, "HTTP/1.1", response.Proto)
		require.Equal(t, http.StatusNotFound, response.StatusCode)
	})

	t.Run("HTTP/2 prior knowledge", func(t *testing.T) {
		transport := &http.Transport{Protocols: httpProtocols(false, true)}
		defer transport.CloseIdleConnections()
		client := &http.Client{Transport: transport}
		response, err := client.Get("http://" + listener.Addr().String() + "/")
		require.NoError(t, err)
		defer response.Body.Close()

		require.Equal(t, "HTTP/2.0", response.Proto)
		require.Equal(t, http.StatusNotFound, response.StatusCode)
	})

	t.Run("HTTP/1.1 Upgrade h2c", func(t *testing.T) {
		conn, err := net.Dial("tcp", listener.Addr().String())
		require.NoError(t, err)
		defer conn.Close()

		_, err = fmt.Fprintf(conn,
			"GET / HTTP/1.1\r\nHost: %s\r\n"+
				"Connection: Upgrade, HTTP2-Settings\r\nUpgrade: h2c\r\n"+
				"HTTP2-Settings: AAMAAABkAAQCAAAAAAIAAAAA\r\n\r\n",
			listener.Addr())
		require.NoError(t, err)
		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		require.NoError(t, err)
		defer response.Body.Close()

		require.NotEqual(t, http.StatusSwitchingProtocols, response.StatusCode)
	})
}

func httpProtocols(http1, unencryptedHTTP2 bool) *http.Protocols {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(http1)
	protocols.SetUnencryptedHTTP2(unencryptedHTTP2)
	return protocols
}

func TestCheckRouteAuthByRequest(t *testing.T) {
	rc := &RouteConfig{
		Username: "alice",
		Password: "secret",
	}

	t.Run("accepts nil route config", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		require.True(t, checkRouteAuthByRequest(req, nil))
	})

	t.Run("accepts route without credentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		require.True(t, checkRouteAuthByRequest(req, &RouteConfig{}))
	})

	t.Run("accepts authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.SetBasicAuth("alice", "secret")
		require.True(t, checkRouteAuthByRequest(req, rc))
	})

	t.Run("accepts proxy authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://target.example.com/", nil)
		req.Header.Set("Proxy-Authorization", httppkg.BasicAuth("alice", "secret"))
		require.True(t, checkRouteAuthByRequest(req, rc))
	})

	t.Run("rejects authorization fallback for proxy request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://target.example.com/", nil)
		req.SetBasicAuth("alice", "secret")
		require.False(t, checkRouteAuthByRequest(req, rc))
	})

	t.Run("rejects wrong proxy authorization even when authorization matches", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://target.example.com/", nil)
		req.SetBasicAuth("alice", "secret")
		req.Header.Set("Proxy-Authorization", httppkg.BasicAuth("alice", "wrong"))
		require.False(t, checkRouteAuthByRequest(req, rc))
	})

	t.Run("rejects when neither header matches", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://target.example.com/", nil)
		req.SetBasicAuth("alice", "wrong")
		req.Header.Set("Proxy-Authorization", httppkg.BasicAuth("alice", "wrong"))
		require.False(t, checkRouteAuthByRequest(req, rc))
	})

	t.Run("rejects proxy authorization on direct request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Proxy-Authorization", httppkg.BasicAuth("alice", "secret"))
		require.False(t, checkRouteAuthByRequest(req, rc))
	})
}

func TestGetRequestRouteUser(t *testing.T) {
	t.Run("proxy request uses proxy authorization username", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://target.example.com/", nil)
		req.Host = "target.example.com"
		req.Header.Set("Proxy-Authorization", httppkg.BasicAuth("proxy-user", "proxy-pass"))
		req.SetBasicAuth("direct-user", "direct-pass")

		require.Equal(t, "proxy-user", getRequestRouteUser(req))
	})

	t.Run("connect request keeps proxy authorization routing", func(t *testing.T) {
		req := httptest.NewRequest("CONNECT", "http://target.example.com:443", nil)
		req.Host = "target.example.com:443"
		req.Header.Set("Proxy-Authorization", httppkg.BasicAuth("proxy-user", "proxy-pass"))
		req.SetBasicAuth("direct-user", "direct-pass")

		require.Equal(t, "proxy-user", getRequestRouteUser(req))
	})

	t.Run("direct request uses authorization username", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Host = "example.com"
		req.SetBasicAuth("direct-user", "direct-pass")

		require.Equal(t, "direct-user", getRequestRouteUser(req))
	})

	t.Run("proxy request does not fall back when proxy authorization is invalid", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://target.example.com/", nil)
		req.Host = "target.example.com"
		req.Header.Set("Proxy-Authorization", "Basic !!!")
		req.SetBasicAuth("direct-user", "direct-pass")

		require.Empty(t, getRequestRouteUser(req))
	})
}

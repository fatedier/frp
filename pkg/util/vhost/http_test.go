package vhost

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	httppkg "github.com/fatedier/frp/pkg/util/http"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

func TestHTTPConnectTunnel(t *testing.T) {
	for _, tc := range []struct {
		name      string
		framing   string
		pipelined bool
	}{
		{name: "plain"},
		{name: "plain-early", pipelined: true},
		{name: "cl0", framing: "Content-Length: 0\r\n"},
		{name: "cl0-early", framing: "Content-Length: 0\r\n", pipelined: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			remote, backend := net.Pipe()
			defer remote.Close()
			defer backend.Close()
			require.NoError(t, backend.SetDeadline(time.Now().Add(5*time.Second)))

			rp := NewHTTPReverseProxy(HTTPReverseProxyOptions{}, NewRouters())
			require.NoError(t, rp.Register(RouteConfig{
				Domain:   "target.example.com",
				Username: "alice",
				Password: "secret",
				CreateConnFn: func(string) (net.Conn, error) {
					return remote, nil
				},
			}))
			listener := netpkg.NewInternalListener()
			server := &http.Server{Handler: rp, ReadHeaderTimeout: time.Second}
			serveErr := make(chan error, 1)
			go func() {
				serveErr <- server.Serve(listener)
			}()
			defer func() {
				require.NoError(t, server.Close())
				require.ErrorIs(t, <-serveErr, http.ErrServerClosed)
			}()

			client, incoming := net.Pipe()
			defer client.Close()
			defer incoming.Close()
			require.NoError(t, client.SetDeadline(time.Now().Add(5*time.Second)))
			require.NoError(t, listener.PutConn(incoming))

			proxyAuth := httppkg.BasicAuth("alice", "secret")
			request := "CONNECT target.example.com:443 HTTP/1.1\r\nHost: target.example.com:443\r\n" +
				"X-Connect-Test: tunnel\r\nProxy-Authorization: " + proxyAuth + "\r\n" + tc.framing + "\r\n"
			if tc.pipelined {
				// One pipe write puts the tunnel data in net/http's request buffer.
				request += "early data"
			}
			_, err := io.WriteString(client, request)
			require.NoError(t, err)
			backendReader := bufio.NewReader(backend)
			req, err := http.ReadRequest(backendReader)
			require.NoError(t, err)
			defer req.Body.Close()
			require.Equal(t, http.MethodConnect, req.Method)
			require.Equal(t, "target.example.com:443", req.Host)
			require.Equal(t, "target.example.com:443", req.URL.Host)
			require.Equal(t, "tunnel", req.Header.Get("X-Connect-Test"))
			require.Equal(t, proxyAuth, req.Header.Get("Proxy-Authorization"))
			require.Zero(t, req.ContentLength)
			require.Empty(t, req.TransferEncoding)
			if tc.pipelined {
				data := make([]byte, len("early data"))
				_, err = io.ReadFull(backendReader, data)
				require.NoError(t, err)
				require.Equal(t, "early data", string(data))
			}

			writeErr := make(chan error, 1)
			go func() {
				_, err := io.WriteString(backend, "HTTP/1.1 200 Connection Established\r\n\r\nreply")
				writeErr <- err
			}()
			clientReader := bufio.NewReader(client)
			response, err := http.ReadResponse(clientReader, req)
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, http.StatusOK, response.StatusCode)
			reply := make([]byte, len("reply"))
			_, err = io.ReadFull(clientReader, reply)
			require.NoError(t, err)
			require.Equal(t, "reply", string(reply))
			require.NoError(t, <-writeErr)

			_, err = io.WriteString(client, "later data")
			require.NoError(t, err)
			data := make([]byte, len("later data"))
			_, err = io.ReadFull(backendReader, data)
			require.NoError(t, err)
			require.Equal(t, "later data", string(data))
		})
	}
}

func TestHTTPConnectRejectsContent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		framing string
		body    string
	}{
		{name: "content-length", framing: "Content-Length: 1\r\n", body: "X"},
		{name: "chunked", framing: "Transfer-Encoding: chunked\r\n", body: "1\r\nX\r\n0\r\n\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backendCalls := 0
			rp := NewHTTPReverseProxy(HTTPReverseProxyOptions{}, NewRouters())
			require.NoError(t, rp.Register(RouteConfig{
				Domain: "target.example.com",
				CreateConnFn: func(string) (net.Conn, error) {
					backendCalls++
					return nil, io.ErrClosedPipe
				},
			}))
			req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(
				"CONNECT target.example.com:443 HTTP/1.1\r\nHost: target.example.com:443\r\n" + tc.framing + "\r\n" + tc.body)))
			require.NoError(t, err)
			defer req.Body.Close()
			rw := &connectTestResponseWriter{ResponseRecorder: httptest.NewRecorder()}

			rp.ServeHTTP(rw, req)

			require.Equal(t, http.StatusBadRequest, rw.Code)
			require.Zero(t, rw.hijackCalls, "reject content before Hijack")
			require.Zero(t, backendCalls, "reject content before creating a backend")
		})
	}
}

func TestHTTPConnectHeaderWriteFailure(t *testing.T) {
	for _, originalBody := range []bool{false, true} {
		t.Run(fmt.Sprintf("originalBody=%v", originalBody), func(t *testing.T) {
			client, incoming := net.Pipe()
			defer client.Close()
			defer incoming.Close()
			remote, backend := net.Pipe()
			defer remote.Close()
			defer backend.Close()
			clientConn := &connectTestConn{Conn: incoming}
			remoteConn := &connectTestConn{Conn: remote, failWrites: true}
			rp := NewHTTPReverseProxy(HTTPReverseProxyOptions{}, NewRouters())
			require.NoError(t, rp.Register(RouteConfig{
				Domain: "target.example.com",
				CreateConnFn: func(string) (net.Conn, error) {
					return remoteConn, nil
				},
			}))
			req := rp.injectRequestInfoToCtx(httptest.NewRequest(http.MethodConnect, "target.example.com:443", nil))
			require.Equal(t, "target.example.com:443", req.Host)
			body := &connectTestBody{}
			if originalBody {
				req.Body = body
			}
			requestBody := req.Body
			getBodyCalls := 0
			req.GetBody = func() (io.ReadCloser, error) {
				getBodyCalls++
				return body, nil
			}
			req.Header.Set("Content-Length", "0")
			req.Header.Set("Trailer", "X-Connect-Trailer")
			req.Trailer = http.Header{"X-Connect-Trailer": {"value"}}
			originalHeader, originalTrailer := req.Header.Clone(), req.Trailer.Clone()
			rw := &connectTestResponseWriter{
				ResponseRecorder: httptest.NewRecorder(),
				conn:             clientConn,
				buffered:         bufio.NewReadWriter(bufio.NewReader(clientConn), bufio.NewWriter(clientConn)),
			}

			rp.connectHandler(rw, req)

			require.Equal(t, 1, rw.hijackCalls)
			require.Zero(t, body.reads, "do not read the original Body after Hijack")
			require.Zero(t, body.closes, "do not close the original Body after Hijack")
			require.Positive(t, remoteConn.writes.Load(), "exercise CONNECT header write failure")
			require.True(t, remoteConn.closed.Load(), "close backend on CONNECT header write failure")
			require.True(t, clientConn.closed.Load(), "close client on CONNECT header write failure")
			require.Zero(t, remoteConn.reads.Load(), "do not start Join after header write failure")
			require.Zero(t, clientConn.reads.Load(), "do not start Join after header write failure")
			require.Zero(t, getBodyCalls)
			require.Equal(t, requestBody, req.Body)
			require.NotNil(t, req.GetBody)
			require.Equal(t, originalHeader, req.Header)
			require.Equal(t, originalTrailer, req.Trailer)
		})
	}
}

type connectTestResponseWriter struct {
	*httptest.ResponseRecorder
	conn        net.Conn
	buffered    *bufio.ReadWriter
	hijackCalls int
}

func (rw *connectTestResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	rw.hijackCalls++
	if rw.conn == nil {
		return nil, nil, io.ErrClosedPipe
	}
	return rw.conn, rw.buffered, nil
}

type connectTestConn struct {
	net.Conn
	failWrites bool
	closed     atomic.Bool
	reads      atomic.Int32
	writes     atomic.Int32
}

func (c *connectTestConn) Read(p []byte) (int, error) {
	c.reads.Add(1)
	return c.Conn.Read(p)
}

func (c *connectTestConn) Write(p []byte) (int, error) {
	c.writes.Add(1)
	if c.failWrites {
		return 0, io.ErrClosedPipe
	}
	return c.Conn.Write(p)
}

func (c *connectTestConn) Close() error {
	c.closed.Store(true)
	return c.Conn.Close()
}

type connectTestBody struct {
	reads  int
	closes int
}

func (b *connectTestBody) Read([]byte) (int, error) {
	b.reads++
	return 0, io.ErrUnexpectedEOF
}

func (b *connectTestBody) Close() error {
	b.closes++
	return nil
}

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

package vhost

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPRouteReplacementUsesNewBackend(t *testing.T) {
	oldBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "old backend")
	}))
	t.Cleanup(oldBackend.Close)
	newBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "new backend")
	}))
	t.Cleanup(newBackend.Close)

	var oldDials, newDials atomic.Int32
	oldRoute := RouteConfig{
		Domain: "example.com",
		CreateConnFn: func(string) (net.Conn, error) {
			oldDials.Add(1)
			return net.Dial("tcp", oldBackend.Listener.Addr().String())
		},
	}
	newRoute := RouteConfig{
		Domain: oldRoute.Domain,
		CreateConnFn: func(string) (net.Conn, error) {
			newDials.Add(1)
			return net.Dial("tcp", newBackend.Listener.Addr().String())
		},
	}
	rp := NewHTTPReverseProxy(HTTPReverseProxyOptions{}, NewRouters())
	require.NoError(t, rp.Register(oldRoute))
	frontend := httptest.NewServer(rp)
	t.Cleanup(frontend.Close)

	client := &http.Client{
		// Each request opens a fresh frontend connection, so only the proxy's
		// backend connection pool can retain the old destination.
		Transport: &http.Transport{DisableKeepAlives: true},
		Timeout:   5 * time.Second,
	}
	t.Cleanup(client.CloseIdleConnections)
	request := func() string {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, frontend.URL, nil)
		require.NoError(t, err)
		req.Host = oldRoute.Domain
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return string(body)
	}

	require.Equal(t, "old backend", request())
	require.Equal(t, "old backend", request())
	require.EqualValues(t, 1, oldDials.Load(), "the unchanged route should reuse its backend connection")

	rp.UnRegister(oldRoute)
	require.NoError(t, rp.Register(newRoute))

	require.Equal(t, "new backend", request(), "a new request must use the replacement route")
	require.EqualValues(t, 1, newDials.Load())
}

func TestHTTPRouteReplacementKeepsMatchedConfiguration(t *testing.T) {
	oldBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "old backend")
	}))
	t.Cleanup(oldBackend.Close)
	newBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "new backend")
	}))
	t.Cleanup(newBackend.Close)

	rp := NewHTTPReverseProxy(HTTPReverseProxyOptions{}, NewRouters())
	oldRoute := RouteConfig{
		Domain:   "example.com",
		Username: "alice",
		Password: "old password",
		CreateConnFn: func(string) (net.Conn, error) {
			return net.Dial("tcp", oldBackend.Listener.Addr().String())
		},
	}
	newRoute := RouteConfig{
		Domain:   oldRoute.Domain,
		Username: "alice",
		Password: "new password",
		CreateConnFn: func(string) (net.Conn, error) {
			return net.Dial("tcp", newBackend.Listener.Addr().String())
		},
	}
	replaced := make(chan error, 1)
	oldRoute.ChooseEndpointFn = func() (string, error) {
		// Rewrite runs after route matching and authentication, but before
		// Transport dials. Replace the route at this exact boundary.
		rp.UnRegister(oldRoute)
		replaced <- rp.Register(newRoute)
		return "", nil
	}
	require.NoError(t, rp.Register(oldRoute))
	frontend := httptest.NewServer(rp)
	t.Cleanup(frontend.Close)
	client := &http.Client{Timeout: 5 * time.Second}

	request := func(password string) (int, string) {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, frontend.URL, nil)
		require.NoError(t, err)
		req.Host = oldRoute.Domain
		req.SetBasicAuth("alice", password)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return resp.StatusCode, string(body)
	}

	status, body := request("old password")
	require.NoError(t, <-replaced)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "old backend", body, "dial the route whose policy authorized this request")

	status, body = request("new password")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "new backend", body)

	status, _ = request("old password")
	require.Equal(t, http.StatusUnauthorized, status, "later requests must use the replacement policy")
}

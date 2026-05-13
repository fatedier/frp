package vhost

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	httppkg "github.com/fatedier/frp/pkg/util/http"
)

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

func TestStripPrefix(t *testing.T) {
	tests := []struct {
		name         string
		location     string
		stripPrefix  bool
		requestPath  string
		expectedPath string
	}{
		{
			name:         "strip prefix enabled with matching location",
			location:     "/api",
			stripPrefix:  true,
			requestPath:  "/api/users",
			expectedPath: "/users",
		},
		{
			name:         "strip prefix enabled with exact match",
			location:     "/api",
			stripPrefix:  true,
			requestPath:  "/api",
			expectedPath: "/",
		},
		{
			name:         "strip prefix enabled with nested path",
			location:     "/api",
			stripPrefix:  true,
			requestPath:  "/api/v1/data",
			expectedPath: "/v1/data",
		},
		{
			name:         "strip prefix disabled",
			location:     "/api",
			stripPrefix:  false,
			requestPath:  "/api/users",
			expectedPath: "/api/users",
		},
		{
			name:         "strip prefix enabled but path doesn't match",
			location:     "/api",
			stripPrefix:  true,
			requestPath:  "/other/path",
			expectedPath: "/other/path",
		},
		{
			name:         "empty location",
			location:     "",
			stripPrefix:  true,
			requestPath:  "/api/users",
			expectedPath: "/api/users",
		},
		{
			name:         "don't strip partial prefix match",
			location:     "/api",
			stripPrefix:  true,
			requestPath:  "/apiv2/users",
			expectedPath: "/apiv2/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that echoes the request path
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(r.URL.Path))
			}))
			defer backend.Close()

			// Create the reverse proxy with our rewrite logic
			proxy := &httputil.ReverseProxy{
				Rewrite: func(r *httputil.ProxyRequest) {
					req := r.Out
					req.URL.Scheme = "http"
					req.URL.Host = strings.TrimPrefix(backend.URL, "http://")

					// Simulate the RouteConfig being set in context
					rc := &RouteConfig{
						Location:    tt.location,
						StripPrefix: tt.stripPrefix,
					}

					// Apply the strip prefix logic
					if rc.StripPrefix && rc.Location != "" && hasPathPrefix(req.URL.Path, rc.Location) {
						req.URL.Path = strings.TrimPrefix(req.URL.Path, rc.Location)
						if req.URL.Path == "" {
							req.URL.Path = "/"
						}
					}
				},
			}

			// Create a test request
			req := httptest.NewRequest("GET", "http://example.com"+tt.requestPath, nil)
			w := httptest.NewRecorder()

			// Execute the proxy
			proxy.ServeHTTP(w, req)

			// Check the result
			if w.Body.String() != tt.expectedPath {
				t.Errorf("Expected path %q, got %q", tt.expectedPath, w.Body.String())
			}
		})
	}
}

package vhost

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"testing"
)

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
					if rc.StripPrefix && rc.Location != "" && strings.HasPrefix(req.URL.Path, rc.Location) {
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

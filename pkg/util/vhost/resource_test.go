package vhost

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func resetErrorPageState(t *testing.T) {
	t.Helper()
	oldPath, oldPages := NotFoundPagePath, customErrorPages
	NotFoundPagePath = ""
	customErrorPages = make(map[int]CustomErrorPage)
	t.Cleanup(func() {
		NotFoundPagePath = oldPath
		customErrorPages = oldPages
	})
}

func TestResolveNotFoundPage(t *testing.T) {
	t.Run("default page", func(t *testing.T) {
		resetErrorPageState(t)
		p := resolveNotFoundPage()
		require.Equal(t, NotFound, p.Body)
		require.Equal(t, "text/html", p.contentType())
	})

	t.Run("custom error page for 404", func(t *testing.T) {
		resetErrorPageState(t)
		customErrorPages[http.StatusNotFound] = CustomErrorPage{
			ContentType: "text/plain",
			Body:        "custom not found",
			Headers:     map[string]string{"X-Custom": "yes"},
		}
		p := resolveNotFoundPage()
		require.Equal(t, "custom not found", p.Body)
		require.Equal(t, "text/plain", p.contentType())
		require.Equal(t, "yes", p.Headers["X-Custom"])
	})

	t.Run("custom404Page file takes precedence", func(t *testing.T) {
		resetErrorPageState(t)
		customErrorPages[http.StatusNotFound] = CustomErrorPage{Body: "inline"}
		file := filepath.Join(t.TempDir(), "404.html")
		require.NoError(t, os.WriteFile(file, []byte("from file"), 0o644))
		NotFoundPagePath = file

		p := resolveNotFoundPage()
		require.Equal(t, "from file", p.Body)
	})

	t.Run("missing custom404Page file falls back to inline page", func(t *testing.T) {
		resetErrorPageState(t)
		customErrorPages[http.StatusNotFound] = CustomErrorPage{Body: "inline"}
		NotFoundPagePath = filepath.Join(t.TempDir(), "missing.html")

		p := resolveNotFoundPage()
		require.Equal(t, "inline", p.Body)
	})
}

func TestWriteCustomErrorPage(t *testing.T) {
	t.Run("no custom page", func(t *testing.T) {
		resetErrorPageState(t)
		rw := httptest.NewRecorder()
		require.False(t, writeCustomErrorPage(rw, http.StatusGatewayTimeout))
		require.Equal(t, http.StatusOK, rw.Code)
	})

	t.Run("writes custom page", func(t *testing.T) {
		resetErrorPageState(t)
		customErrorPages[http.StatusGatewayTimeout] = CustomErrorPage{
			Body:        "timeout page",
			ContentType: "text/plain",
			Headers:     map[string]string{"Retry-After": "30"},
		}
		rw := httptest.NewRecorder()
		require.True(t, writeCustomErrorPage(rw, http.StatusGatewayTimeout))
		require.Equal(t, http.StatusGatewayTimeout, rw.Code)
		require.Equal(t, "text/plain", rw.Header().Get("Content-Type"))
		require.Equal(t, "30", rw.Header().Get("Retry-After"))
		require.Equal(t, "timeout page", rw.Body.String())
	})
}

func TestNotFoundResponseCustomPage(t *testing.T) {
	resetErrorPageState(t)
	customErrorPages[http.StatusNotFound] = CustomErrorPage{
		Body:        "inline not found",
		ContentType: "application/json",
	}

	res := NotFoundResponse()
	defer res.Body.Close()
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	require.Equal(t, "application/json", res.Header.Get("Content-Type"))
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, "inline not found", string(body))
	require.Equal(t, int64(len("inline not found")), res.ContentLength)
}

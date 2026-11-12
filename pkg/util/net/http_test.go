// Copyright 2026 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package net

import (
	"bytes"
	"compress/gzip"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPGzipHandlerFileServer(t *testing.T) {
	const content = "0123456789abcdef"
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte(content), 0o600))

	server := httptest.NewServer(MakeHTTPGzipHandler(http.FileServer(http.Dir(dir))))
	t.Cleanup(server.Close)
	client := &http.Client{
		// Inspect the response encoding ourselves, without automatic decompression.
		Transport: &http.Transport{DisableCompression: true},
		Timeout:   5 * time.Second,
	}
	t.Cleanup(client.CloseIdleConnections)

	tests := []struct {
		name             string
		acceptEncoding   string
		rangeHeader      string
		wantStatus       int
		wantEncoding     string
		wantContentRange string
		wantContent      string
	}{
		{
			name:        "uncompressed full response",
			wantStatus:  http.StatusOK,
			wantContent: content,
		},
		{
			name:           "compressed full response",
			acceptEncoding: "gzip",
			wantStatus:     http.StatusOK,
			wantEncoding:   "gzip",
			wantContent:    content,
		},
		{
			name:             "single range with gzip accepted",
			acceptEncoding:   "gzip",
			rangeHeader:      "bytes=0-9",
			wantStatus:       http.StatusPartialContent,
			wantContentRange: "bytes 0-9/16",
			wantContent:      "0123456789",
		},
		{
			name:           "multiple ranges with gzip accepted",
			acceptEncoding: "gzip",
			rangeHeader:    "bytes=0-3,8-11",
			wantStatus:     http.StatusPartialContent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, server.URL+"/file.txt", nil)
			require.NoError(t, err)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			if tt.rangeHeader != "" {
				req.Header.Set("Range", tt.rangeHeader)
			}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tt.wantStatus, resp.StatusCode)
			require.Equal(t, tt.wantContentRange, resp.Header.Get("Content-Range"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if tt.wantEncoding == "gzip" {
				reader, err := gzip.NewReader(bytes.NewReader(body))
				require.NoError(t, err)
				defer reader.Close()
				body, err = io.ReadAll(reader)
				require.NoError(t, err)
			}

			if tt.wantContent != "" {
				require.Equal(t, tt.wantContent, string(body))
			} else {
				mediaType, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
				require.NoError(t, err)
				require.Equal(t, "multipart/byteranges", mediaType)
				reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
				for _, want := range []struct{ contentRange, content string }{
					{"bytes 0-3/16", "0123"},
					{"bytes 8-11/16", "89ab"},
				} {
					part, err := reader.NextPart()
					require.NoError(t, err)
					require.Equal(t, want.contentRange, part.Header.Get("Content-Range"))
					partBody, err := io.ReadAll(part)
					require.NoError(t, err)
					require.Equal(t, want.content, string(partBody))
				}
				_, err = reader.NextPart()
				require.ErrorIs(t, err, io.EOF)
			}
			require.Equal(t, tt.wantEncoding, resp.Header.Get("Content-Encoding"))
		})
	}
}

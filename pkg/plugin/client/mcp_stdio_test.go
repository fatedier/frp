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

//go:build !frps

package client

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestMCPStdioClassifyJSONRPC(t *testing.T) {
	cases := []struct {
		name                      string
		body                      string
		hasID, isInit, isInitNote bool
	}{
		{"request", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, true, false, false},
		{"initialize", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, true, true, false},
		{"initialized note", `{"jsonrpc":"2.0","method":"notifications/initialized"}`, false, false, true},
		{"plain notification", `{"jsonrpc":"2.0","method":"notifications/cancelled"}`, false, false, false},
		{"explicit null id", `{"jsonrpc":"2.0","id":null,"method":"x"}`, false, false, false},
		{"malformed JSON", `not json`, true, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotInit, gotInitNote := classifyJSONRPC([]byte(tc.body))
			require.Equal(t, tc.hasID, gotID, "hasID")
			require.Equal(t, tc.isInit, gotInit, "isInit")
			require.Equal(t, tc.isInitNote, gotInitNote, "isInitNote")
		})
	}
}

// fakeMCPScript is a tiny stdio program loaded via `/bin/sh script`. It
// counts every input line and echoes only those that look like JSON-RPC
// requests (have an "id" field), prefixing the response with the call
// count so a test can detect when a new instance has been spawned.
const fakeMCPScript = `n=0
while IFS= read -r line; do
  n=$((n+1))
  case "$line" in
    *'"id"'*) printf '%s\n' "{\"jsonrpc\":\"2.0\",\"result\":\"call=${n}\",\"id\":1,\"echo\":${line}}" ;;
  esac
done
`

func TestMCPStdioDispatchAndReplay(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh stub")
	}

	dir := t.TempDir()
	scriptPath := dir + "/fake.sh"
	require.NoError(t, os.WriteFile(scriptPath, []byte(fakeMCPScript), 0o600))

	plugin, err := NewMCPStdioPlugin(PluginContext{}, &v1.MCPStdioPluginOptions{
		Type:    v1.PluginMCPStdio,
		Command: []string{"/bin/sh", scriptPath},
	})
	require.NoError(t, err)
	defer plugin.Close()

	p := plugin.(*MCPStdioPlugin)

	// Initialize handshake.
	resp, err := p.dispatch([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	require.NoError(t, err)
	require.Contains(t, string(resp), `"call=1"`)
	require.NotNil(t, p.cachedInitReq)

	// Notification: no response.
	resp, err = p.dispatch([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	require.NoError(t, err)
	require.Nil(t, resp)
	require.NotNil(t, p.cachedInitNote)

	// Regular request goes to the same child (call=3 because init was 1, note was 2).
	resp, err = p.dispatch([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	require.NoError(t, err)
	require.Contains(t, string(resp), `"call=3"`)

	// Simulate idle reap.
	p.mu.Lock()
	p.killChildLocked()
	p.mu.Unlock()

	// Next request must respawn and replay initialize+initialized note before
	// processing, so this request will be call=3 in the new child (init=1, note=2).
	resp, err = p.dispatch([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	require.NoError(t, err)
	require.Contains(t, string(resp), `"call=3"`)
}

func TestMCPStdioHTTPHandler(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh stub")
	}

	dir := t.TempDir()
	scriptPath := dir + "/fake.sh"
	require.NoError(t, os.WriteFile(scriptPath, []byte(fakeMCPScript), 0o600))

	plugin, err := NewMCPStdioPlugin(PluginContext{}, &v1.MCPStdioPluginOptions{
		Type:    v1.PluginMCPStdio,
		Command: []string{"/bin/sh", scriptPath},
	})
	require.NoError(t, err)
	defer plugin.Close()

	p := plugin.(*MCPStdioPlugin)

	// Wrong method
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	p.handle(w, r)
	require.Equal(t, http.StatusMethodNotAllowed, w.Code)

	// Empty body
	r = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	w = httptest.NewRecorder()
	p.handle(w, r)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Notification → 202 with empty body
	r = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc":"2.0","method":"x"}`))
	w = httptest.NewRecorder()
	p.handle(w, r)
	require.Equal(t, http.StatusAccepted, w.Code)
	require.Empty(t, w.Body.String())

	// Request with id → 200 + body
	r = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"ping"}`))
	w = httptest.NewRecorder()
	p.handle(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, bytes.Contains(w.Body.Bytes(), []byte(`"call=`)))
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Give reapLoop goroutine a moment if any (idle=0 means it never started).
	time.Sleep(10 * time.Millisecond)
}

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

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// TestManagerNewHTTPRequestCancelsHTTPPluginTransport verifies that incoming
// request cancellation interrupts an in-flight HTTP plugin request.
func TestManagerNewHTTPRequestCancelsHTTPPluginTransport(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	pluginServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-releaseRequest
	}))
	t.Cleanup(pluginServer.Close)
	t.Cleanup(func() { close(releaseRequest) })

	m := NewManager()
	m.Register(NewHTTPPluginOptions(v1.HTTPPluginOptions{
		Name: "blocking request plugin",
		Addr: pluginServer.URL,
		Ops:  []string{OpNewHTTPRequest},
	}))

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- m.NewHTTPRequest(ctx, &NewHTTPRequestContent{})
	}()

	require.Eventually(t, func() bool {
		select {
		case <-requestStarted:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("HTTP plugin call did not return after request cancellation")
	}
}

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

package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestMonitorResetsFailedTimesAfterSuccess(t *testing.T) {
	var checkCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := checkCount.Add(1)
		if count == 1 || count == 2 || count == 4 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var failedCount atomic.Int32
	monitor := NewMonitor(
		context.Background(),
		v1.HealthCheckConfig{
			Type:            "http",
			Path:            "/health",
			TimeoutSeconds:  1,
			IntervalSeconds: 1,
			MaxFailed:       3,
		},
		strings.TrimPrefix(server.URL, "http://"),
		func() {},
		func() { failedCount.Add(1) },
	)
	monitor.interval = 10 * time.Millisecond
	monitor.Start()
	defer monitor.Stop()

	require.Eventually(t, func() bool {
		return checkCount.Load() >= 5
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, int32(0), failedCount.Load())
}

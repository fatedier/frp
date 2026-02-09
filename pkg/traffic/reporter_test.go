// Copyright 2024 The frp Authors
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

package traffic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReport_JSON(t *testing.T) {
	report := Report{
		Token:      "test-token-123",
		Region:     "us-east",
		ProxyName:  "web-proxy",
		TrafficIn:  1024 * 1024,     // 1MB
		TrafficOut: 10 * 1024 * 1024, // 10MB
		Timestamp:  1234567890,
	}

	data, err := json.Marshal(report)
	require.NoError(t, err)

	var decoded Report
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, report.Token, decoded.Token)
	assert.Equal(t, report.Region, decoded.Region)
	assert.Equal(t, report.ProxyName, decoded.ProxyName)
	assert.Equal(t, report.TrafficIn, decoded.TrafficIn)
	assert.Equal(t, report.TrafficOut, decoded.TrafficOut)
	assert.Equal(t, report.Timestamp, decoded.Timestamp)
}

func TestTokenTrafficCounter_AddTraffic(t *testing.T) {
	counter := NewTokenTrafficCounter("test-token", "us-east", "", 0)

	// Add traffic
	counter.AddTraffic("proxy1", 100, 200)
	counter.AddTraffic("proxy1", 150, 250)
	counter.AddTraffic("proxy2", 50, 100)

	in, out := counter.GetTraffic()
	assert.Equal(t, int64(300), in)  // 100 + 150 + 50
	assert.Equal(t, int64(550), out) // 200 + 250 + 100
}

func TestTokenTrafficCounter_ThresholdReport(t *testing.T) {
	var reportCount atomic.Int32
	var lastReport Report

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var report Report
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		lastReport = report
		reportCount.Add(1)
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create counter with 1MB threshold
	counter := NewTokenTrafficCounter("test-token-123", "us-east", server.URL, 1)

	// Add traffic below threshold (should not trigger report)
	counter.AddTraffic("proxy1", 100, 500*1024) // 500KB
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(0), reportCount.Load())

	// Add more traffic to exceed threshold (1MB out)
	counter.AddTraffic("proxy1", 100, 600*1024) // Now total out = 1.1MB
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), reportCount.Load())
	assert.Equal(t, "test-token-123", lastReport.Token)
	assert.Equal(t, "us-east", lastReport.Region)
	assert.Equal(t, "proxy1", lastReport.ProxyName)
}

func TestTokenTrafficCounter_NoReportWhenDisabled(t *testing.T) {
	var reportCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reportCount.Add(1)
		w.WriteHeader(200)
	}))
	defer server.Close()

	// Counter with empty URL (disabled)
	counter1 := NewTokenTrafficCounter("test-token", "us-east", "", 1)
	counter1.AddTraffic("proxy1", 10*MB, 10*MB)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(0), reportCount.Load())

	// Counter with 0 interval (disabled)
	counter2 := NewTokenTrafficCounter("test-token", "us-east", server.URL, 0)
	counter2.AddTraffic("proxy1", 10*MB, 10*MB)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(0), reportCount.Load())
}

func TestTokenTrafficCounter_Flush(t *testing.T) {
	var reportCount atomic.Int32
	var lastReport Report

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var report Report
		json.NewDecoder(r.Body).Decode(&report)
		lastReport = report
		reportCount.Add(1)
		w.WriteHeader(200)
	}))
	defer server.Close()

	counter := NewTokenTrafficCounter("test-token", "us-east", server.URL, 100) // 100MB threshold

	// Add traffic below threshold
	counter.AddTraffic("proxy1", 1*MB, 2*MB)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(0), reportCount.Load()) // No report yet

	// Flush should send remaining traffic
	counter.Flush("proxy1")
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), reportCount.Load())
	assert.Equal(t, int64(1*MB), lastReport.TrafficIn)
	assert.Equal(t, int64(2*MB), lastReport.TrafficOut)
}

func TestManager_GetOrCreateCounter(t *testing.T) {
	manager := NewManager("http://example.com/report", "us-east")

	// Get counter for new token
	counter1 := manager.GetOrCreateCounter("token-1", 50)
	require.NotNil(t, counter1)

	// Get same counter again
	counter2 := manager.GetOrCreateCounter("token-1", 50)
	assert.Equal(t, counter1, counter2) // Should be same instance

	// Get different counter
	counter3 := manager.GetOrCreateCounter("token-2", 100)
	assert.NotEqual(t, counter1, counter3)
}

func TestManager_GetCounter(t *testing.T) {
	manager := NewManager("http://example.com/report", "us-east")

	// Non-existent counter
	counter := manager.GetCounter("non-existent")
	assert.Nil(t, counter)

	// Create and get
	manager.GetOrCreateCounter("token-1", 50)
	counter = manager.GetCounter("token-1")
	assert.NotNil(t, counter)
}

func TestManager_RemoveCounter(t *testing.T) {
	var reportCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reportCount.Add(1)
		w.WriteHeader(200)
	}))
	defer server.Close()

	manager := NewManager(server.URL, "us-east")

	// Create counter and add traffic
	counter := manager.GetOrCreateCounter("token-1", 100)
	counter.AddTraffic("proxy1", 1*MB, 2*MB)

	// Remove should flush
	manager.RemoveCounter("token-1")
	time.Sleep(100 * time.Millisecond)

	// Counter should be removed
	assert.Nil(t, manager.GetCounter("token-1"))

	// Flush should have been called (report sent)
	assert.Equal(t, int32(1), reportCount.Load())
}

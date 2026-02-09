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
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatedier/frp/pkg/util/log"
)

const (
	MB = 1024 * 1024
)

// Report represents a traffic usage report.
type Report struct {
	Token     string `json:"token"`
	Region    string `json:"region"`
	ProxyName string `json:"proxyName"`
	TrafficIn int64  `json:"trafficIn"`  // bytes
	TrafficOut int64  `json:"trafficOut"` // bytes
	Timestamp int64  `json:"timestamp"`  // unix timestamp
}

// TokenTrafficCounter tracks traffic for a single token.
type TokenTrafficCounter struct {
	token              string
	region             string
	reportURL          string
	reportIntervalMB   int64

	// Total traffic since last report
	trafficIn  atomic.Int64
	trafficOut atomic.Int64

	// Last reported traffic threshold
	lastReportedIn  atomic.Int64
	lastReportedOut atomic.Int64

	// Per-proxy traffic (for detailed reporting)
	proxyTraffic sync.Map // map[proxyName]*proxyTrafficCounter
}

type proxyTrafficCounter struct {
	in  atomic.Int64
	out atomic.Int64
}

// NewTokenTrafficCounter creates a new traffic counter for a token.
func NewTokenTrafficCounter(token, region, reportURL string, reportIntervalMB int64) *TokenTrafficCounter {
	return &TokenTrafficCounter{
		token:            token,
		region:           region,
		reportURL:        reportURL,
		reportIntervalMB: reportIntervalMB,
	}
}

// AddTraffic adds traffic and triggers report if threshold is reached.
func (c *TokenTrafficCounter) AddTraffic(proxyName string, in, out int64) {
	c.trafficIn.Add(in)
	c.trafficOut.Add(out)

	// Track per-proxy traffic
	counter, _ := c.proxyTraffic.LoadOrStore(proxyName, &proxyTrafficCounter{})
	pc := counter.(*proxyTrafficCounter)
	pc.in.Add(in)
	pc.out.Add(out)

	// Check if we should report
	c.checkAndReport(proxyName)
}

// checkAndReport checks if traffic threshold is reached and sends report.
func (c *TokenTrafficCounter) checkAndReport(proxyName string) {
	if c.reportURL == "" || c.reportIntervalMB <= 0 {
		return
	}

	thresholdBytes := c.reportIntervalMB * MB

	totalIn := c.trafficIn.Load()
	totalOut := c.trafficOut.Load()
	lastIn := c.lastReportedIn.Load()
	lastOut := c.lastReportedOut.Load()

	// Check if either direction has reached the threshold
	inDiff := totalIn - lastIn
	outDiff := totalOut - lastOut

	log.Debugf("traffic check: totalIn=%d, totalOut=%d, lastIn=%d, lastOut=%d, inDiff=%d, outDiff=%d, threshold=%d",
		totalIn, totalOut, lastIn, lastOut, inDiff, outDiff, thresholdBytes)

	if inDiff >= thresholdBytes || outDiff >= thresholdBytes {
		// Update last reported values
		c.lastReportedIn.Store(totalIn)
		c.lastReportedOut.Store(totalOut)

		log.Infof("traffic threshold reached, sending report: inDiff=%d, outDiff=%d", inDiff, outDiff)
		// Send report asynchronously
		go c.sendReport(proxyName, inDiff, outDiff)
	}
}

// sendReport sends a traffic report to the configured URL.
func (c *TokenTrafficCounter) sendReport(proxyName string, trafficIn, trafficOut int64) {
	report := Report{
		Token:      c.token,
		Region:     c.region,
		ProxyName:  proxyName,
		TrafficIn:  trafficIn,
		TrafficOut: trafficOut,
		Timestamp:  time.Now().Unix(),
	}

	data, err := json.Marshal(report)
	if err != nil {
		log.Warnf("failed to marshal traffic report: %v", err)
		return
	}

	log.Infof("sending traffic report to %s: %s", c.reportURL, string(data))

	resp, err := http.Post(c.reportURL, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Warnf("failed to send traffic report for token [%s]: %v", c.token[:min(8, len(c.token))]+"...", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warnf("traffic report for token [%s] returned status %d", c.token[:min(8, len(c.token))]+"...", resp.StatusCode)
		return
	}

	log.Infof("traffic report sent successfully for token [%s]: in=%d, out=%d",
		c.token[:min(8, len(c.token))]+"...", trafficIn, trafficOut)
}

// GetTraffic returns current traffic statistics.
func (c *TokenTrafficCounter) GetTraffic() (in, out int64) {
	return c.trafficIn.Load(), c.trafficOut.Load()
}

// Flush sends any remaining unreported traffic.
func (c *TokenTrafficCounter) Flush(proxyName string) {
	if c.reportURL == "" || c.reportIntervalMB <= 0 {
		return
	}

	totalIn := c.trafficIn.Load()
	totalOut := c.trafficOut.Load()
	lastIn := c.lastReportedIn.Load()
	lastOut := c.lastReportedOut.Load()

	inDiff := totalIn - lastIn
	outDiff := totalOut - lastOut

	// Only report if there's unreported traffic
	if inDiff > 0 || outDiff > 0 {
		c.lastReportedIn.Store(totalIn)
		c.lastReportedOut.Store(totalOut)
		c.sendReport(proxyName, inDiff, outDiff)
	}
}

// Manager manages traffic counters for all tokens.
type Manager struct {
	reportURL string
	region    string
	counters  sync.Map // map[token]*TokenTrafficCounter
}

// NewManager creates a new traffic manager.
func NewManager(reportURL, region string) *Manager {
	return &Manager{
		reportURL: reportURL,
		region:    region,
	}
}

// GetOrCreateCounter gets or creates a traffic counter for a token.
func (m *Manager) GetOrCreateCounter(token string, reportIntervalMB int64) *TokenTrafficCounter {
	counter, loaded := m.counters.LoadOrStore(token, NewTokenTrafficCounter(token, m.region, m.reportURL, reportIntervalMB))
	if !loaded {
		log.Debugf("created traffic counter for token [%s] with interval %dMB", token[:min(8, len(token))]+"...", reportIntervalMB)
	}
	return counter.(*TokenTrafficCounter)
}

// RemoveCounter removes a traffic counter and flushes remaining traffic.
func (m *Manager) RemoveCounter(token string) {
	if counter, ok := m.counters.LoadAndDelete(token); ok {
		tc := counter.(*TokenTrafficCounter)
		tc.Flush("")
	}
}

// GetCounter returns a traffic counter for a token if it exists.
func (m *Manager) GetCounter(token string) *TokenTrafficCounter {
	if counter, ok := m.counters.Load(token); ok {
		return counter.(*TokenTrafficCounter)
	}
	return nil
}

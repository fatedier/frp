// Copyright 2019 fatedier, fatedier@gmail.com
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

package mem

import (
	"sync"
	"time"

	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/metric"
	server "github.com/fatedier/frp/server/metrics"
)

var (
	sm = newServerMetrics()

	ServerMetrics  server.ServerMetrics
	StatsCollector Collector
)

func init() {
	ServerMetrics = sm
	StatsCollector = sm
	sm.run()
}

type serverMetrics struct {
	info *ServerStatistics
	mu   sync.Mutex
}

func newServerMetrics() *serverMetrics {
	return &serverMetrics{
		info: &ServerStatistics{
			TotalTrafficIn:  metric.NewDateCounter(ReserveDays),
			TotalTrafficOut: metric.NewDateCounter(ReserveDays),
			CurConns:        metric.NewCounter(),

			ClientCounts:    metric.NewCounter(),
			ProxyTypeCounts: make(map[string]metric.Counter),

			AutoNegotiationSuccess:         metric.NewCounter(),
			AutoNegotiationFailure:         metric.NewCounter(),
			AutoTransportSelections:        make(map[string]metric.Counter),
			AutoTransportClientCounts:      make(map[string]metric.Counter),
			AutoTransportSwitchCounts:      make(map[string]metric.Counter),
			AutoTransportIllegalSelections: make(map[string]metric.Counter),

			ProxyStatistics: make(map[string]*ProxyStatistics),
		},
	}
}

func (m *serverMetrics) run() {
	go func() {
		for {
			time.Sleep(12 * time.Hour)
			start := time.Now()
			count, total := m.clearUselessInfo(time.Duration(7*24) * time.Hour)
			log.Debugf("clear useless proxy statistics data count %d/%d, cost %v", count, total, time.Since(start))
		}
	}()
}

func (m *serverMetrics) clearUselessInfo(continuousOfflineDuration time.Duration) (int, int) {
	count := 0
	total := 0
	// To check if there are any proxies that have been closed for more than continuousOfflineDuration and remove them.
	m.mu.Lock()
	defer m.mu.Unlock()
	total = len(m.info.ProxyStatistics)
	for name, data := range m.info.ProxyStatistics {
		if !data.LastCloseTime.IsZero() &&
			data.LastStartTime.Before(data.LastCloseTime) &&
			time.Since(data.LastCloseTime) > continuousOfflineDuration {
			delete(m.info.ProxyStatistics, name)
			count++
			log.Tracef("clear proxy [%s]'s statistics data, lastCloseTime: [%s]", name, data.LastCloseTime.String())
		}
	}
	return count, total
}

func (m *serverMetrics) ClearOfflineProxies() (int, int) {
	return m.clearUselessInfo(0)
}

func (m *serverMetrics) NewClient() {
	m.info.ClientCounts.Inc(1)
}

func (m *serverMetrics) CloseClient() {
	m.info.ClientCounts.Dec(1)
}

func (m *serverMetrics) NewProxy(name string, proxyType string, user string, clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	counter, ok := m.info.ProxyTypeCounts[proxyType]
	if !ok {
		counter = metric.NewCounter()
	}
	counter.Inc(1)
	m.info.ProxyTypeCounts[proxyType] = counter

	proxyStats, ok := m.info.ProxyStatistics[name]
	if !ok || proxyStats.ProxyType != proxyType {
		proxyStats = &ProxyStatistics{
			Name:       name,
			ProxyType:  proxyType,
			CurConns:   metric.NewCounter(),
			TrafficIn:  metric.NewDateCounter(ReserveDays),
			TrafficOut: metric.NewDateCounter(ReserveDays),
		}
		m.info.ProxyStatistics[name] = proxyStats
	}
	proxyStats.User = user
	proxyStats.ClientID = clientID
	proxyStats.LastStartTime = time.Now()
}

func (m *serverMetrics) CloseProxy(name string, proxyType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if counter, ok := m.info.ProxyTypeCounts[proxyType]; ok {
		counter.Dec(1)
	}
	if proxyStats, ok := m.info.ProxyStatistics[name]; ok {
		proxyStats.LastCloseTime = time.Now()
	}
}

func (m *serverMetrics) OpenConnection(name string, _ string) {
	m.info.CurConns.Inc(1)

	m.mu.Lock()
	defer m.mu.Unlock()
	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.CurConns.Inc(1)
	}
}

func (m *serverMetrics) CloseConnection(name string, _ string) {
	m.info.CurConns.Dec(1)

	m.mu.Lock()
	defer m.mu.Unlock()
	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.CurConns.Dec(1)
	}
}

func (m *serverMetrics) AddTrafficIn(name string, _ string, trafficBytes int64) {
	m.info.TotalTrafficIn.Inc(trafficBytes)

	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.TrafficIn.Inc(trafficBytes)
	}
}

func (m *serverMetrics) AddTrafficOut(name string, _ string, trafficBytes int64) {
	m.info.TotalTrafficOut.Inc(trafficBytes)

	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.TrafficOut.Inc(trafficBytes)
	}
}

func (m *serverMetrics) AutoNegotiation(success bool) {
	if success {
		m.info.AutoNegotiationSuccess.Inc(1)
		return
	}
	m.info.AutoNegotiationFailure.Inc(1)
}

func (m *serverMetrics) AutoTransportSelected(protocol string) {
	if protocol == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	incCounterMap(m.info.AutoTransportSelections, protocol, 1)
}

func (m *serverMetrics) AutoTransportClientOnline(protocol string) {
	if protocol == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	incCounterMap(m.info.AutoTransportClientCounts, protocol, 1)
}

func (m *serverMetrics) AutoTransportClientOffline(protocol string) {
	if protocol == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	incCounterMap(m.info.AutoTransportClientCounts, protocol, -1)
}

func (m *serverMetrics) AutoTransportSwitch(oldProtocol string, newProtocol string) {
	if oldProtocol == "" || newProtocol == "" || oldProtocol == newProtocol {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	incCounterMap(m.info.AutoTransportSwitchCounts, oldProtocol+"->"+newProtocol, 1)
}

func (m *serverMetrics) AutoTransportRejected(protocol string) {
	if protocol == "" {
		protocol = "unknown"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	incCounterMap(m.info.AutoTransportIllegalSelections, protocol, 1)
}

func incCounterMap(counters map[string]metric.Counter, key string, delta int32) {
	counter, ok := counters[key]
	if !ok {
		counter = metric.NewCounter()
		counters[key] = counter
	}
	if delta >= 0 {
		counter.Inc(delta)
		return
	}
	counter.Dec(-delta)
}

// Get stats data api.

func (m *serverMetrics) GetServer() *ServerStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := &ServerStats{
		TotalTrafficIn:                 m.info.TotalTrafficIn.TodayCount(),
		TotalTrafficOut:                m.info.TotalTrafficOut.TodayCount(),
		CurConns:                       int64(m.info.CurConns.Count()),
		ClientCounts:                   int64(m.info.ClientCounts.Count()),
		ProxyTypeCounts:                make(map[string]int64),
		AutoNegotiationSuccess:         int64(m.info.AutoNegotiationSuccess.Count()),
		AutoNegotiationFailure:         int64(m.info.AutoNegotiationFailure.Count()),
		AutoTransportSelections:        make(map[string]int64),
		AutoTransportClientCounts:      make(map[string]int64),
		AutoTransportSwitchCounts:      make(map[string]int64),
		AutoTransportIllegalSelections: make(map[string]int64),
	}
	for k, v := range m.info.ProxyTypeCounts {
		s.ProxyTypeCounts[k] = int64(v.Count())
	}
	copyCounterMap(s.AutoTransportSelections, m.info.AutoTransportSelections)
	copyCounterMap(s.AutoTransportClientCounts, m.info.AutoTransportClientCounts)
	copyCounterMap(s.AutoTransportSwitchCounts, m.info.AutoTransportSwitchCounts)
	copyCounterMap(s.AutoTransportIllegalSelections, m.info.AutoTransportIllegalSelections)
	return s
}

func copyCounterMap(dst map[string]int64, src map[string]metric.Counter) {
	for k, v := range src {
		dst[k] = int64(v.Count())
	}
}

func toProxyStats(name string, proxyStats *ProxyStatistics) *ProxyStats {
	ps := &ProxyStats{
		Name:            name,
		Type:            proxyStats.ProxyType,
		User:            proxyStats.User,
		ClientID:        proxyStats.ClientID,
		TodayTrafficIn:  proxyStats.TrafficIn.TodayCount(),
		TodayTrafficOut: proxyStats.TrafficOut.TodayCount(),
		CurConns:        int64(proxyStats.CurConns.Count()),
	}
	if !proxyStats.LastStartTime.IsZero() {
		ps.LastStartTime = proxyStats.LastStartTime.Format("01-02 15:04:05")
	}
	if !proxyStats.LastCloseTime.IsZero() {
		ps.LastCloseTime = proxyStats.LastCloseTime.Format("01-02 15:04:05")
	}
	return ps
}

func (m *serverMetrics) GetProxiesByType(proxyType string) []*ProxyStats {
	res := make([]*ProxyStats, 0)
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, proxyStats := range m.info.ProxyStatistics {
		if proxyStats.ProxyType != proxyType {
			continue
		}
		res = append(res, toProxyStats(name, proxyStats))
	}
	return res
}

func (m *serverMetrics) GetProxiesByTypeAndName(proxyType string, proxyName string) (res *ProxyStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[proxyName]
	if ok && proxyStats.ProxyType == proxyType {
		res = toProxyStats(proxyName, proxyStats)
	}
	return
}

func (m *serverMetrics) GetProxyByName(proxyName string) (res *ProxyStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[proxyName]
	if ok {
		res = toProxyStats(proxyName, proxyStats)
	}
	return
}

func (m *serverMetrics) GetProxyTraffic(name string) (res *ProxyTrafficInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		res = &ProxyTrafficInfo{
			Name: name,
		}
		res.TrafficIn = proxyStats.TrafficIn.GetLastDaysCount(ReserveDays)
		res.TrafficOut = proxyStats.TrafficOut.GetLastDaysCount(ReserveDays)
	}
	return
}

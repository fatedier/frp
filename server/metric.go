// Copyright 2017 fatedier, fatedier@gmail.com
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
	"sync"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/utils/metric"
)

const (
	ReserveDays = 7
)

var globalStats *ServerStatistics

type ServerStatistics struct {
	TotalFlowIn  metric.DateCounter
	TotalFlowOut metric.DateCounter
	CurConns     metric.Counter

	ClientCounts    metric.Counter
	ProxyTypeCounts map[string]metric.Counter

	ProxyStatistics map[string]*ProxyStatistics

	mu sync.Mutex
}

type ProxyStatistics struct {
	ProxyType string
	FlowIn    metric.DateCounter
	FlowOut   metric.DateCounter
	CurConns  metric.Counter
}

func init() {
	globalStats = &ServerStatistics{
		TotalFlowIn:  metric.NewDateCounter(ReserveDays),
		TotalFlowOut: metric.NewDateCounter(ReserveDays),
		CurConns:     metric.NewCounter(),

		ClientCounts:    metric.NewCounter(),
		ProxyTypeCounts: make(map[string]metric.Counter),

		ProxyStatistics: make(map[string]*ProxyStatistics),
	}
}

func StatsNewClient() {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.ClientCounts.Inc(1)
	}
}

func StatsCloseClient() {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.ClientCounts.Dec(1)
	}
}

func StatsNewProxy(name string, proxyType string) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()
		counter, ok := globalStats.ProxyTypeCounts[proxyType]
		if !ok {
			counter = metric.NewCounter()
		}
		counter.Inc(1)
		globalStats.ProxyTypeCounts[proxyType] = counter

		proxyStats, ok := globalStats.ProxyStatistics[name]
		if !ok {
			proxyStats = &ProxyStatistics{
				ProxyType: proxyType,
				CurConns:  metric.NewCounter(),
				FlowIn:    metric.NewDateCounter(ReserveDays),
				FlowOut:   metric.NewDateCounter(ReserveDays),
			}
			globalStats.ProxyStatistics[name] = proxyStats
		}
	}
}

func StatsCloseProxy(proxyType string) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()
		if counter, ok := globalStats.ProxyTypeCounts[proxyType]; ok {
			counter.Dec(1)
		}
	}
}

func StatsOpenConnection(name string) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.CurConns.Inc(1)

		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()
		proxyStats, ok := globalStats.ProxyStatistics[name]
		if ok {
			proxyStats.CurConns.Inc(1)
			globalStats.ProxyStatistics[name] = proxyStats
		}
	}
}

func StatsCloseConnection(name string) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.CurConns.Dec(1)

		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()
		proxyStats, ok := globalStats.ProxyStatistics[name]
		if ok {
			proxyStats.CurConns.Dec(1)
			globalStats.ProxyStatistics[name] = proxyStats
		}
	}
}

func StatsAddFlowIn(name string, flowIn int64) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.TotalFlowIn.Inc(flowIn)

		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()

		proxyStats, ok := globalStats.ProxyStatistics[name]
		if ok {
			proxyStats.FlowIn.Inc(flowIn)
			globalStats.ProxyStatistics[name] = proxyStats
		}
	}
}

func StatsAddFlowOut(name string, flowOut int64) {
	if config.ServerCommonCfg.DashboardPort != 0 {
		globalStats.TotalFlowOut.Inc(flowOut)

		globalStats.mu.Lock()
		defer globalStats.mu.Unlock()

		proxyStats, ok := globalStats.ProxyStatistics[name]
		if ok {
			proxyStats.FlowOut.Inc(flowOut)
			globalStats.ProxyStatistics[name] = proxyStats
		}
	}
}

// Functions for getting server stats.
type ServerStats struct {
	TotalFlowIn     int64
	TotalFlowOut    int64
	CurConns        int64
	ClientCounts    int64
	ProxyTypeCounts map[string]int64
}

func StatsGetServer() *ServerStats {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	s := &ServerStats{
		TotalFlowIn:     globalStats.TotalFlowIn.TodayCount(),
		TotalFlowOut:    globalStats.TotalFlowOut.TodayCount(),
		CurConns:        globalStats.CurConns.Count(),
		ClientCounts:    globalStats.ClientCounts.Count(),
		ProxyTypeCounts: make(map[string]int64),
	}
	for k, v := range globalStats.ProxyTypeCounts {
		s.ProxyTypeCounts[k] = v.Count()
	}
	return s
}

type ProxyStats struct {
	Name         string
	Type         string
	TodayFlowIn  int64
	TodayFlowOut int64
	CurConns     int64
}

func StatsGetProxiesByType(proxyType string) []*ProxyStats {
	res := make([]*ProxyStats, 0)
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()

	for name, proxyStats := range globalStats.ProxyStatistics {
		if proxyStats.ProxyType != proxyType {
			continue
		}

		ps := &ProxyStats{
			Name:         name,
			Type:         proxyStats.ProxyType,
			TodayFlowIn:  proxyStats.FlowIn.TodayCount(),
			TodayFlowOut: proxyStats.FlowOut.TodayCount(),
			CurConns:     proxyStats.CurConns.Count(),
		}
		res = append(res, ps)
	}
	return res
}

type ProxyFlowInfo struct {
	Name    string
	FlowIn  []int64
	FlowOut []int64
}

func StatsGetProxyFlow(name string) (res *ProxyFlowInfo) {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()

	proxyStats, ok := globalStats.ProxyStatistics[name]
	if ok {
		res = &ProxyFlowInfo{
			Name: name,
		}
		res.FlowIn = proxyStats.FlowIn.GetLastDaysCount(ReserveDays)
		res.FlowOut = proxyStats.FlowOut.GetLastDaysCount(ReserveDays)
	}
	return
}
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

package stats

import (
	"sync"
	"time"

	"github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/metric"
)

type internalCollector struct {
	enable bool
	info   *ServerStatistics
	mu     sync.Mutex
}

func NewInternalCollector(enable bool) Collector {
	return &internalCollector{
		enable: enable,
		info: &ServerStatistics{
			TotalTrafficIn:  metric.NewDateCounter(ReserveDays),
			TotalTrafficOut: metric.NewDateCounter(ReserveDays),
			CurConns:        metric.NewCounter(),

			ClientCounts:    metric.NewCounter(),
			ProxyTypeCounts: make(map[string]metric.Counter),

			ProxyStatistics: make(map[string]*ProxyStatistics),
		},
	}
}

func (collector *internalCollector) Run() error {
	go func() {
		for {
			time.Sleep(12 * time.Hour)
			log.Debug("start to clear useless proxy statistics data...")
			collector.ClearUselessInfo()
			log.Debug("finish to clear useless proxy statistics data")
		}
	}()
	return nil
}

func (collector *internalCollector) ClearUselessInfo() {
	// To check if there are proxies that closed than 7 days and drop them.
	collector.mu.Lock()
	defer collector.mu.Unlock()
	for name, data := range collector.info.ProxyStatistics {
		if !data.LastCloseTime.IsZero() && time.Since(data.LastCloseTime) > time.Duration(7*24)*time.Hour {
			delete(collector.info.ProxyStatistics, name)
			log.Trace("clear proxy [%s]'s statistics data, lastCloseTime: [%s]", name, data.LastCloseTime.String())
		}
	}
}

func (collector *internalCollector) Mark(statsType StatsType, payload interface{}) {
	if !collector.enable {
		return
	}

	switch v := payload.(type) {
	case *NewClientPayload:
		collector.newClient(v)
	case *CloseClientPayload:
		collector.closeClient(v)
	case *NewProxyPayload:
		collector.newProxy(v)
	case *CloseProxyPayload:
		collector.closeProxy(v)
	case *OpenConnectionPayload:
		collector.openConnection(v)
	case *CloseConnectionPayload:
		collector.closeConnection(v)
	case *AddTrafficInPayload:
		collector.addTrafficIn(v)
	case *AddTrafficOutPayload:
		collector.addTrafficOut(v)
	}
}

func (collector *internalCollector) newClient(payload *NewClientPayload) {
	collector.info.ClientCounts.Inc(1)
}

func (collector *internalCollector) closeClient(payload *CloseClientPayload) {
	collector.info.ClientCounts.Dec(1)
}

func (collector *internalCollector) newProxy(payload *NewProxyPayload) {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	counter, ok := collector.info.ProxyTypeCounts[payload.ProxyType]
	if !ok {
		counter = metric.NewCounter()
	}
	counter.Inc(1)
	collector.info.ProxyTypeCounts[payload.ProxyType] = counter

	proxyStats, ok := collector.info.ProxyStatistics[payload.Name]
	if !(ok && proxyStats.ProxyType == payload.ProxyType) {
		proxyStats = &ProxyStatistics{
			Name:       payload.Name,
			ProxyType:  payload.ProxyType,
			CurConns:   metric.NewCounter(),
			TrafficIn:  metric.NewDateCounter(ReserveDays),
			TrafficOut: metric.NewDateCounter(ReserveDays),
		}
		collector.info.ProxyStatistics[payload.Name] = proxyStats
	}
	proxyStats.LastStartTime = time.Now()
}

func (collector *internalCollector) closeProxy(payload *CloseProxyPayload) {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	if counter, ok := collector.info.ProxyTypeCounts[payload.ProxyType]; ok {
		counter.Dec(1)
	}
	if proxyStats, ok := collector.info.ProxyStatistics[payload.Name]; ok {
		proxyStats.LastCloseTime = time.Now()
	}
}

func (collector *internalCollector) openConnection(payload *OpenConnectionPayload) {
	collector.info.CurConns.Inc(1)

	collector.mu.Lock()
	defer collector.mu.Unlock()
	proxyStats, ok := collector.info.ProxyStatistics[payload.ProxyName]
	if ok {
		proxyStats.CurConns.Inc(1)
		collector.info.ProxyStatistics[payload.ProxyName] = proxyStats
	}
}

func (collector *internalCollector) closeConnection(payload *CloseConnectionPayload) {
	collector.info.CurConns.Dec(1)

	collector.mu.Lock()
	defer collector.mu.Unlock()
	proxyStats, ok := collector.info.ProxyStatistics[payload.ProxyName]
	if ok {
		proxyStats.CurConns.Dec(1)
		collector.info.ProxyStatistics[payload.ProxyName] = proxyStats
	}
}

func (collector *internalCollector) addTrafficIn(payload *AddTrafficInPayload) {
	collector.info.TotalTrafficIn.Inc(payload.TrafficBytes)

	collector.mu.Lock()
	defer collector.mu.Unlock()

	proxyStats, ok := collector.info.ProxyStatistics[payload.ProxyName]
	if ok {
		proxyStats.TrafficIn.Inc(payload.TrafficBytes)
		collector.info.ProxyStatistics[payload.ProxyName] = proxyStats
	}
}

func (collector *internalCollector) addTrafficOut(payload *AddTrafficOutPayload) {
	collector.info.TotalTrafficOut.Inc(payload.TrafficBytes)

	collector.mu.Lock()
	defer collector.mu.Unlock()

	proxyStats, ok := collector.info.ProxyStatistics[payload.ProxyName]
	if ok {
		proxyStats.TrafficOut.Inc(payload.TrafficBytes)
		collector.info.ProxyStatistics[payload.ProxyName] = proxyStats
	}
}

func (collector *internalCollector) GetServer() *ServerStats {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	s := &ServerStats{
		TotalTrafficIn:  collector.info.TotalTrafficIn.TodayCount(),
		TotalTrafficOut: collector.info.TotalTrafficOut.TodayCount(),
		CurConns:        collector.info.CurConns.Count(),
		ClientCounts:    collector.info.ClientCounts.Count(),
		ProxyTypeCounts: make(map[string]int64),
	}
	for k, v := range collector.info.ProxyTypeCounts {
		s.ProxyTypeCounts[k] = v.Count()
	}
	return s
}

func (collector *internalCollector) GetProxiesByType(proxyType string) []*ProxyStats {
	res := make([]*ProxyStats, 0)
	collector.mu.Lock()
	defer collector.mu.Unlock()

	for name, proxyStats := range collector.info.ProxyStatistics {
		if proxyStats.ProxyType != proxyType {
			continue
		}

		ps := &ProxyStats{
			Name:            name,
			Type:            proxyStats.ProxyType,
			TodayTrafficIn:  proxyStats.TrafficIn.TodayCount(),
			TodayTrafficOut: proxyStats.TrafficOut.TodayCount(),
			CurConns:        proxyStats.CurConns.Count(),
		}
		if !proxyStats.LastStartTime.IsZero() {
			ps.LastStartTime = proxyStats.LastStartTime.Format("01-02 15:04:05")
		}
		if !proxyStats.LastCloseTime.IsZero() {
			ps.LastCloseTime = proxyStats.LastCloseTime.Format("01-02 15:04:05")
		}
		res = append(res, ps)
	}
	return res
}

func (collector *internalCollector) GetProxiesByTypeAndName(proxyType string, proxyName string) (res *ProxyStats) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	for name, proxyStats := range collector.info.ProxyStatistics {
		if proxyStats.ProxyType != proxyType {
			continue
		}

		if name != proxyName {
			continue
		}

		res = &ProxyStats{
			Name:            name,
			Type:            proxyStats.ProxyType,
			TodayTrafficIn:  proxyStats.TrafficIn.TodayCount(),
			TodayTrafficOut: proxyStats.TrafficOut.TodayCount(),
			CurConns:        proxyStats.CurConns.Count(),
		}
		if !proxyStats.LastStartTime.IsZero() {
			res.LastStartTime = proxyStats.LastStartTime.Format("01-02 15:04:05")
		}
		if !proxyStats.LastCloseTime.IsZero() {
			res.LastCloseTime = proxyStats.LastCloseTime.Format("01-02 15:04:05")
		}
		break
	}
	return
}

func (collector *internalCollector) GetProxyTraffic(name string) (res *ProxyTrafficInfo) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	proxyStats, ok := collector.info.ProxyStatistics[name]
	if ok {
		res = &ProxyTrafficInfo{
			Name: name,
		}
		res.TrafficIn = proxyStats.TrafficIn.GetLastDaysCount(ReserveDays)
		res.TrafficOut = proxyStats.TrafficOut.GetLastDaysCount(ReserveDays)
	}
	return
}

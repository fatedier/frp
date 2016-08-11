// Copyright 2016 fatedier, fatedier@gmail.com
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

package metric

import (
	"sort"
	"sync"
	"time"

	"frp/models/consts"
)

var (
	DailyDataKeepDays   int = 7
	ServerMetricInfoMap map[string]*ServerMetric
	smMutex             sync.RWMutex
)

type ServerMetric struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	BindAddr      string   `json:"bind_addr"`
	ListenPort    int64    `json:"listen_port"`
	CustomDomains []string `json:"custom_domains"`
	Status        string   `json:"status"`
	UseEncryption bool     `json:"use_encryption"`
	UseGzip       bool     `json:"use_gzip"`
	PrivilegeMode bool     `json:"privilege_mode"`

	// statistics
	CurrentConns int64               `json:"current_conns"`
	Daily        []*DailyServerStats `json:"daily"`
	mutex        sync.RWMutex
}

type DailyServerStats struct {
	Time             string `json:"time"`
	FlowIn           int64  `json:"flow_in"`
	FlowOut          int64  `json:"flow_out"`
	TotalAcceptConns int64  `json:"total_accept_conns"`
}

// for sort
type ServerMetricList []*ServerMetric

func (l ServerMetricList) Len() int           { return len(l) }
func (l ServerMetricList) Less(i, j int) bool { return l[i].Name < l[j].Name }
func (l ServerMetricList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

func init() {
	ServerMetricInfoMap = make(map[string]*ServerMetric)
}

func (s *ServerMetric) clone() *ServerMetric {
	copy := *s
	copy.CustomDomains = make([]string, len(s.CustomDomains))
	var i int
	for i = range copy.CustomDomains {
		copy.CustomDomains[i] = s.CustomDomains[i]
	}

	copy.Daily = make([]*DailyServerStats, len(s.Daily))
	for i = range copy.Daily {
		tmpDaily := *s.Daily[i]
		copy.Daily[i] = &tmpDaily
	}
	return &copy
}

func GetAllProxyMetrics() []*ServerMetric {
	result := make(ServerMetricList, 0)
	smMutex.RLock()
	for _, metric := range ServerMetricInfoMap {
		metric.mutex.RLock()
		tmpMetric := metric.clone()
		metric.mutex.RUnlock()
		result = append(result, tmpMetric)
	}
	smMutex.RUnlock()

	// sort for result by proxy name
	sort.Sort(result)
	return result
}

// if proxyName isn't exist, return nil
func GetProxyMetrics(proxyName string) *ServerMetric {
	smMutex.RLock()
	defer smMutex.RUnlock()
	metric, ok := ServerMetricInfoMap[proxyName]
	if ok {
		metric.mutex.RLock()
		tmpMetric := metric.clone()
		metric.mutex.RUnlock()
		return tmpMetric
	} else {
		return nil
	}
}

func SetProxyInfo(proxyName string, proxyType, bindAddr string,
	useEncryption, useGzip, privilegeMode bool, customDomains []string,
	listenPort int64) {
	smMutex.Lock()
	info, ok := ServerMetricInfoMap[proxyName]
	if !ok {
		info = &ServerMetric{}
		info.Daily = make([]*DailyServerStats, 0)
	}
	info.Name = proxyName
	info.Type = proxyType
	info.UseEncryption = useEncryption
	info.UseGzip = useGzip
	info.PrivilegeMode = privilegeMode
	info.BindAddr = bindAddr
	info.ListenPort = listenPort
	info.CustomDomains = customDomains
	ServerMetricInfoMap[proxyName] = info
	smMutex.Unlock()
}

func SetStatus(proxyName string, status int64) {
	smMutex.RLock()
	metric, ok := ServerMetricInfoMap[proxyName]
	smMutex.RUnlock()
	if ok {
		metric.mutex.Lock()
		metric.Status = consts.StatusStr[status]
		metric.mutex.Unlock()
	}
}

type DealFuncType func(*DailyServerStats)

func DealDailyData(dailyData []*DailyServerStats, fn DealFuncType) (newDailyData []*DailyServerStats) {
	now := time.Now().Format("20060102")
	dailyLen := len(dailyData)
	if dailyLen == 0 {
		daily := &DailyServerStats{}
		daily.Time = now
		fn(daily)
		dailyData = append(dailyData, daily)
	} else {
		daily := dailyData[dailyLen-1]
		if daily.Time == now {
			fn(daily)
		} else {
			newDaily := &DailyServerStats{}
			newDaily.Time = now
			fn(newDaily)
			if dailyLen == DailyDataKeepDays {
				for i := 0; i < dailyLen-1; i++ {
					dailyData[i] = dailyData[i+1]
				}
				dailyData[dailyLen-1] = newDaily
			} else {
				dailyData = append(dailyData, newDaily)
			}
		}
	}
	return dailyData
}

func OpenConnection(proxyName string) {
	smMutex.RLock()
	metric, ok := ServerMetricInfoMap[proxyName]
	smMutex.RUnlock()
	if ok {
		metric.mutex.Lock()
		metric.CurrentConns++
		metric.Daily = DealDailyData(metric.Daily, func(stats *DailyServerStats) {
			stats.TotalAcceptConns++
		})
		metric.mutex.Unlock()
	}
}

func CloseConnection(proxyName string) {
	smMutex.RLock()
	metric, ok := ServerMetricInfoMap[proxyName]
	smMutex.RUnlock()
	if ok {
		metric.mutex.Lock()
		metric.CurrentConns--
		metric.mutex.Unlock()
	}
}

func AddFlowIn(proxyName string, value int64) {
	smMutex.RLock()
	metric, ok := ServerMetricInfoMap[proxyName]
	smMutex.RUnlock()
	if ok {
		metric.mutex.Lock()
		metric.Daily = DealDailyData(metric.Daily, func(stats *DailyServerStats) {
			stats.FlowIn += value
		})
		metric.mutex.Unlock()
	}
}

func AddFlowOut(proxyName string, value int64) {
	smMutex.RLock()
	metric, ok := ServerMetricInfoMap[proxyName]
	smMutex.RUnlock()
	if ok {
		metric.mutex.Lock()
		metric.Daily = DealDailyData(metric.Daily, func(stats *DailyServerStats) {
			stats.FlowOut += value
		})
		metric.mutex.Unlock()
	}
}

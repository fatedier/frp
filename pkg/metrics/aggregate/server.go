// Copyright 2020 fatedier, fatedier@gmail.com
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

package aggregate

import (
	"github.com/fatedier/frp/pkg/metrics/mem"
	"github.com/fatedier/frp/pkg/metrics/prometheus"
	"github.com/fatedier/frp/server/metrics"
)

// EnableMem start to mark metrics to memory monitor system.
func EnableMem() {
	sm.Add(mem.ServerMetrics)
}

// EnablePrometheus start to mark metrics to prometheus.
func EnablePrometheus() {
	sm.Add(prometheus.ServerMetrics)
}

var sm *serverMetrics = &serverMetrics{}

func init() {
	metrics.Register(sm)
}

type serverMetrics struct {
	ms []metrics.ServerMetrics
}

func (m *serverMetrics) Add(sm metrics.ServerMetrics) {
	m.ms = append(m.ms, sm)
}

func (m *serverMetrics) NewClient() {
	for _, v := range m.ms {
		v.NewClient()
	}
}

func (m *serverMetrics) CloseClient() {
	for _, v := range m.ms {
		v.CloseClient()
	}
}

func (m *serverMetrics) NewProxy(name string, proxyType string) {
	for _, v := range m.ms {
		v.NewProxy(name, proxyType)
	}
}

func (m *serverMetrics) CloseProxy(name string, proxyType string) {
	for _, v := range m.ms {
		v.CloseProxy(name, proxyType)
	}
}

func (m *serverMetrics) OpenConnection(name string, proxyType string) {
	for _, v := range m.ms {
		v.OpenConnection(name, proxyType)
	}
}

func (m *serverMetrics) CloseConnection(name string, proxyType string) {
	for _, v := range m.ms {
		v.CloseConnection(name, proxyType)
	}
}

func (m *serverMetrics) AddTrafficIn(name string, proxyType string, trafficBytes int64) {
	for _, v := range m.ms {
		v.AddTrafficIn(name, proxyType, trafficBytes)
	}
}

func (m *serverMetrics) AddTrafficOut(name string, proxyType string, trafficBytes int64) {
	for _, v := range m.ms {
		v.AddTrafficOut(name, proxyType, trafficBytes)
	}
}

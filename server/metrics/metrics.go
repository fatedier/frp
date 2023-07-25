package metrics

import (
	"sync"
)

type ServerMetrics interface {
	NewClient()
	CloseClient()
	NewProxy(name string, proxyType string)
	CloseProxy(name string, proxyType string)
	OpenConnection(name string, proxyType string)
	CloseConnection(name string, proxyType string)
	AddTrafficIn(name string, proxyType string, trafficBytes int64)
	AddTrafficOut(name string, proxyType string, trafficBytes int64)
}

var Server ServerMetrics = noopServerMetrics{}

var registerMetrics sync.Once

func Register(m ServerMetrics) {
	registerMetrics.Do(func() {
		Server = m
	})
}

type noopServerMetrics struct{}

func (noopServerMetrics) NewClient()                          {}
func (noopServerMetrics) CloseClient()                        {}
func (noopServerMetrics) NewProxy(string, string)             {}
func (noopServerMetrics) CloseProxy(string, string)           {}
func (noopServerMetrics) OpenConnection(string, string)       {}
func (noopServerMetrics) CloseConnection(string, string)      {}
func (noopServerMetrics) AddTrafficIn(string, string, int64)  {}
func (noopServerMetrics) AddTrafficOut(string, string, int64) {}

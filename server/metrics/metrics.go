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

func (noopServerMetrics) NewClient()                                                      {}
func (noopServerMetrics) CloseClient()                                                    {}
func (noopServerMetrics) NewProxy(name string, proxyType string)                          {}
func (noopServerMetrics) CloseProxy(name string, proxyType string)                        {}
func (noopServerMetrics) OpenConnection(name string, proxyType string)                    {}
func (noopServerMetrics) CloseConnection(name string, proxyType string)                   {}
func (noopServerMetrics) AddTrafficIn(name string, proxyType string, trafficBytes int64)  {}
func (noopServerMetrics) AddTrafficOut(name string, proxyType string, trafficBytes int64) {}

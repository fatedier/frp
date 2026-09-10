package metrics

import (
	"sync"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

type ServerMetrics interface {
	NewClient()
	CloseClient()
	NewProxy(name string, proxyType string, user string, clientID string)
	CloseProxy(name string, proxyType string)
	OpenConnection(name string, proxyType string)
	CloseConnection(name string, proxyType string)
	AddTrafficIn(name string, proxyType string, trafficBytes int64)
	AddTrafficOut(name string, proxyType string, trafficBytes int64)
}

// AutoTransportServerMetrics is an optional extension interface for auto transport telemetry.
type AutoTransportServerMetrics interface {
	AutoNegotiation(success bool)
	AutoTransportSelected(protocol string)
	AutoTransportClientOnline(protocol string)
	AutoTransportClientOffline(protocol string)
	AutoTransportSwitch(oldProtocol string, newProtocol string)
	AutoTransportRejected(protocol string)
}

func AutoNegotiation(m ServerMetrics, success bool) {
	if atm, ok := m.(AutoTransportServerMetrics); ok {
		atm.AutoNegotiation(success)
	}
}

func AutoTransportSelected(m ServerMetrics, protocol string) {
	if atm, ok := m.(AutoTransportServerMetrics); ok {
		atm.AutoTransportSelected(protocol)
	}
}

func AutoTransportClientOnline(m ServerMetrics, protocol string) {
	if atm, ok := m.(AutoTransportServerMetrics); ok {
		atm.AutoTransportClientOnline(protocol)
	}
}

func AutoTransportClientOffline(m ServerMetrics, protocol string) {
	if atm, ok := m.(AutoTransportServerMetrics); ok {
		atm.AutoTransportClientOffline(protocol)
	}
}

func AutoTransportSwitch(m ServerMetrics, oldProtocol, newProtocol string) {
	if atm, ok := m.(AutoTransportServerMetrics); ok {
		atm.AutoTransportSwitch(oldProtocol, newProtocol)
	}
}

func AutoTransportRejected(m ServerMetrics, protocol string) {
	if atm, ok := m.(AutoTransportServerMetrics); ok {
		atm.AutoTransportRejected(protocol)
	}
}

var Server ServerMetrics = noopServerMetrics{}

var registerMetrics sync.Once

func SanitizeAutoTransportProtocol(protocol string) string {
	switch protocol {
	case v1.TransportProtocolTCP,
		v1.TransportProtocolKCP,
		v1.TransportProtocolQUIC,
		v1.TransportProtocolWebsocket,
		v1.TransportProtocolWSS:
		return protocol
	default:
		return "unknown"
	}
}

func Register(m ServerMetrics) {
	registerMetrics.Do(func() {
		Server = m
	})
}

type noopServerMetrics struct{}

func (noopServerMetrics) NewClient()                              {}
func (noopServerMetrics) CloseClient()                            {}
func (noopServerMetrics) NewProxy(string, string, string, string) {}
func (noopServerMetrics) CloseProxy(string, string)               {}
func (noopServerMetrics) OpenConnection(string, string)           {}
func (noopServerMetrics) CloseConnection(string, string)          {}
func (noopServerMetrics) AddTrafficIn(string, string, int64)      {}
func (noopServerMetrics) AddTrafficOut(string, string, int64)     {}
func (noopServerMetrics) AutoNegotiation(bool)                    {}
func (noopServerMetrics) AutoTransportSelected(string)            {}
func (noopServerMetrics) AutoTransportClientOnline(string)        {}
func (noopServerMetrics) AutoTransportClientOffline(string)       {}
func (noopServerMetrics) AutoTransportSwitch(string, string)      {}
func (noopServerMetrics) AutoTransportRejected(string)            {}

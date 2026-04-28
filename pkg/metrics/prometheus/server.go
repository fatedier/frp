package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/fatedier/frp/server/metrics"
)

const (
	namespace       = "frp"
	serverSubsystem = "server"
)

var ServerMetrics metrics.ServerMetrics = newServerMetrics()

type serverMetrics struct {
	clientCount        prometheus.Gauge
	proxyCount         *prometheus.GaugeVec
	proxyCountDetailed *prometheus.GaugeVec
	connectionCount    *prometheus.GaugeVec
	trafficIn          *prometheus.CounterVec
	trafficOut         *prometheus.CounterVec
	autoNegotiations   *prometheus.CounterVec
	autoSelections     *prometheus.CounterVec
	autoClientCount    *prometheus.GaugeVec
	autoSwitches       *prometheus.CounterVec
	autoRejects        *prometheus.CounterVec
}

func (m *serverMetrics) NewClient() {
	m.clientCount.Inc()
}

func (m *serverMetrics) CloseClient() {
	m.clientCount.Dec()
}

func (m *serverMetrics) NewProxy(name string, proxyType string, _ string, _ string) {
	m.proxyCount.WithLabelValues(proxyType).Inc()
	m.proxyCountDetailed.WithLabelValues(proxyType, name).Inc()
}

func (m *serverMetrics) CloseProxy(name string, proxyType string) {
	m.proxyCount.WithLabelValues(proxyType).Dec()
	m.proxyCountDetailed.WithLabelValues(proxyType, name).Dec()
}

func (m *serverMetrics) OpenConnection(name string, proxyType string) {
	m.connectionCount.WithLabelValues(name, proxyType).Inc()
}

func (m *serverMetrics) CloseConnection(name string, proxyType string) {
	m.connectionCount.WithLabelValues(name, proxyType).Dec()
}

func (m *serverMetrics) AddTrafficIn(name string, proxyType string, trafficBytes int64) {
	m.trafficIn.WithLabelValues(name, proxyType).Add(float64(trafficBytes))
}

func (m *serverMetrics) AddTrafficOut(name string, proxyType string, trafficBytes int64) {
	m.trafficOut.WithLabelValues(name, proxyType).Add(float64(trafficBytes))
}

func (m *serverMetrics) AutoNegotiation(success bool) {
	result := "failure"
	if success {
		result = "success"
	}
	m.autoNegotiations.WithLabelValues(result).Inc()
}

func (m *serverMetrics) AutoTransportSelected(protocol string) {
	if protocol == "" {
		return
	}
	m.autoSelections.WithLabelValues(protocol).Inc()
}

func (m *serverMetrics) AutoTransportClientOnline(protocol string) {
	if protocol == "" {
		return
	}
	m.autoClientCount.WithLabelValues(protocol).Inc()
}

func (m *serverMetrics) AutoTransportClientOffline(protocol string) {
	if protocol == "" {
		return
	}
	m.autoClientCount.WithLabelValues(protocol).Dec()
}

func (m *serverMetrics) AutoTransportSwitch(oldProtocol string, newProtocol string) {
	if oldProtocol == "" || newProtocol == "" || oldProtocol == newProtocol {
		return
	}
	m.autoSwitches.WithLabelValues(oldProtocol, newProtocol).Inc()
}

func (m *serverMetrics) AutoTransportRejected(protocol string) {
	m.autoRejects.WithLabelValues(metrics.SanitizeAutoTransportProtocol(protocol)).Inc()
}

func newServerMetrics() *serverMetrics {
	m := &serverMetrics{
		clientCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "client_counts",
			Help:      "The current client counts of frps",
		}),
		proxyCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "proxy_counts",
			Help:      "The current proxy counts",
		}, []string{"type"}),
		proxyCountDetailed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "proxy_counts_detailed",
			Help:      "The current number of proxies grouped by type and name",
		}, []string{"type", "name"}),
		connectionCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "connection_counts",
			Help:      "The current connection counts",
		}, []string{"name", "type"}),
		trafficIn: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "traffic_in",
			Help:      "The total in traffic",
		}, []string{"name", "type"}),
		trafficOut: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "traffic_out",
			Help:      "The total out traffic",
		}, []string{"name", "type"}),
		autoNegotiations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "auto_transport_negotiations_total",
			Help:      "The total number of auto transport negotiations grouped by result",
		}, []string{"result"}),
		autoSelections: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "auto_transport_selections_total",
			Help:      "The total number of selected auto transports grouped by protocol",
		}, []string{"protocol"}),
		autoClientCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "auto_transport_client_counts",
			Help:      "The current number of auto transport clients grouped by protocol",
		}, []string{"protocol"}),
		autoSwitches: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "auto_transport_switches_total",
			Help:      "The total number of auto transport switches grouped by old and new protocol",
		}, []string{"old_protocol", "new_protocol"}),
		autoRejects: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: serverSubsystem,
			Name:      "auto_transport_illegal_selections_total",
			Help:      "The total number of rejected auto transport selections grouped by protocol",
		}, []string{"protocol"}),
	}
	prometheus.MustRegister(m.clientCount)
	prometheus.MustRegister(m.proxyCount)
	prometheus.MustRegister(m.proxyCountDetailed)
	prometheus.MustRegister(m.connectionCount)
	prometheus.MustRegister(m.trafficIn)
	prometheus.MustRegister(m.trafficOut)
	prometheus.MustRegister(m.autoNegotiations)
	prometheus.MustRegister(m.autoSelections)
	prometheus.MustRegister(m.autoClientCount)
	prometheus.MustRegister(m.autoSwitches)
	prometheus.MustRegister(m.autoRejects)
	return m
}

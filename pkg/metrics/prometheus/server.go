package prometheus

import (
	"github.com/fatedier/frp/server/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace       = "frp"
	serverSubsystem = "server"
)

var ServerMetrics metrics.ServerMetrics = newServerMetrics()

type serverMetrics struct {
	clientCount     prometheus.Gauge
	proxyCount      *prometheus.GaugeVec
	connectionCount *prometheus.GaugeVec
	trafficIn       *prometheus.CounterVec
	trafficOut      *prometheus.CounterVec
}

func (m *serverMetrics) NewClient() {
	m.clientCount.Inc()
}

func (m *serverMetrics) CloseClient() {
	m.clientCount.Dec()
}

func (m *serverMetrics) NewProxy(name string, proxyType string) {
	m.proxyCount.WithLabelValues(proxyType).Inc()
}

func (m *serverMetrics) CloseProxy(name string, proxyType string) {
	m.proxyCount.WithLabelValues(proxyType).Dec()
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
	}
	prometheus.MustRegister(m.clientCount)
	prometheus.MustRegister(m.proxyCount)
	prometheus.MustRegister(m.connectionCount)
	prometheus.MustRegister(m.trafficIn)
	prometheus.MustRegister(m.trafficOut)
	return m
}

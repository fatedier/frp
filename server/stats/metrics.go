package stats

import (
	"net/http"

	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsServer is a HTTP server wrapper
type MetricsServer struct {
	collector Collector
}

// NewMetricsServer creates a MetricsServer
func NewMetricsServer(c Collector) *MetricsServer {
	return &MetricsServer{
		collector: c,
	}
}

// Serve exposes Prometheus metrics data
func (s *MetricsServer) Serve() {
	http.Handle("/metrics", promhttp.Handler())

	timestampCounter := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "frps_info",
			Name:      "timestamp",
			Help:      "unix nanosec timestamp the data is collected",
		})
	clientCounts := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "frps_info",
			Name:      "client_counts",
			Help:      "number of connected clients",
		})
	prometheus.MustRegister(timestampCounter)
	prometheus.MustRegister(clientCounts)

	go func() {
		for {
			stats := s.collector.GetServer()
			timestampCounter.Add(float64(time.Now().UnixNano()))
			clientCounts.Add(float64(stats.ClientCounts))
			time.Sleep(time.Second)
		}
	}()

	// FIXME load from conf
	http.ListenAndServe(":8080", nil)
}

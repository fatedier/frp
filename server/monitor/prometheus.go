package monitor

import (
	"net/http"
	"strconv"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/server/stats"
	"github.com/fatedier/frp/utils/version"
	"github.com/fatedier/frp/utils/xlog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Prometheus struct {
	statsCollector stats.Collector
	config         config.ServerCommonConf
}

func NewPrometheus(statsCollector stats.Collector, config config.ServerCommonConf) *Prometheus {
	return &Prometheus{
		statsCollector: statsCollector,
		config:         config,
	}
}

var (
	serverBindPort = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "frps_server_bind_port",
		Help: "The port of server frps.",
	}, []string{"address", "version"})
	serverBindUdpPort = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "frps_server_bind_udp_port",
		Help: "The udp port of server frps.",
	})
	serverHeartBeatTimeout = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "frps_server_heartbeat_timeout",
		Help: "The heartbeat timeout of server frps.",
	})
	serverKcpBindPort = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "frps_server_kcp_bind_port",
		Help: "The kcp port of server frps.",
	})
	serverMaxPoolCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "frps_server_max_poolcount",
		Help: "The max poolcount of server frps.",
	})
	serverVhostHttpPort = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "frps_server_http_port",
		Help: "The http port of server frps.",
	}, []string{"subdomain"})
	serverMaxPortsPerClient = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "frps_server_maxport_perclient",
		Help: "The maxport perclient of server frps.",
	})
	serverVhostHttpsPort = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "frps_server_https_port",
		Help: "The https port of server frps.",
	}, []string{"subdomain"})
	serverClientCounts = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "frps_server_client_counts",
		Help: "The client counts of server frps.",
	})
	serverCurConns = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "frps_server_current_connections",
		Help: "The current connections of server frps.",
	})
	serverTotalTrafficIn = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "frps_server_total_trafficin",
		Help: "The total trafficin of server frps.",
	})
	serverTotalTrafficOut = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "frps_server_total_trafficout",
		Help: "The total trafficout of server frps.",
	})
)

type ProxyStats struct {
	Proxys []*stats.ProxyStats
}

func NewProxyStats(Proxys []*stats.ProxyStats) *ProxyStats {
	return &ProxyStats{
		Proxys: Proxys,
	}
}

func (s *ProxyStats) Describe(ch chan<- *prometheus.Desc) {}
func (s *ProxyStats) Collect(ch chan<- prometheus.Metric) {
	dealWithProxyStats(s.Proxys, ch)
}

func setValue(cfg config.ServerCommonConf) {
	serverBindPort.With(prometheus.Labels{"address": cfg.BindAddr, "version": version.Full()}).Set(float64(cfg.BindPort))
	serverBindUdpPort.Set(float64(cfg.BindUdpPort))
	serverVhostHttpsPort.With(prometheus.Labels{"subdomain": cfg.SubDomainHost}).Set(float64(cfg.VhostHttpsPort))
	serverVhostHttpPort.With(prometheus.Labels{"subdomain": cfg.SubDomainHost}).Set(float64(cfg.VhostHttpPort))
	serverMaxPortsPerClient.Set(float64(cfg.MaxPortsPerClient))
	serverMaxPoolCount.Set(float64(cfg.MaxPoolCount))
	serverKcpBindPort.Set(float64(cfg.KcpBindPort))
	serverHeartBeatTimeout.Set(float64(cfg.HeartBeatTimeout))

}

func (p *Prometheus) dealWithServerStats(ServerStats *stats.ServerStats) {
	serverClientCounts.Set(float64(ServerStats.ClientCounts))
	serverCurConns.Set(float64(ServerStats.CurConns))
	serverTotalTrafficIn.Set(float64(ServerStats.TotalTrafficIn))
	serverTotalTrafficOut.Set(float64(ServerStats.TotalTrafficOut))
}

func dealWithProxyStats(ProxyStats []*stats.ProxyStats, ch chan<- prometheus.Metric) {
	for _, v := range ProxyStats {
		// Get the constant labels
		var Labels = []string{"type", "laststarttime", "lastclosetime"}
		var labelValue = []string{v.Type, v.LastStartTime, v.LastCloseTime}
		//set value type
		vType := prometheus.GaugeValue

		//get name
		name := "frps_" + v.Name + "_today_trafficin"
		//get description
		description := "The today trafficin of proxy."

		//registry
		desc := prometheus.NewDesc(name, description, Labels, nil)
		ch <- prometheus.MustNewConstMetric(desc, vType, float64(v.TodayTrafficIn), labelValue...)

		//get name
		name = "frps_" + v.Name + "_today_trafficout"
		description = "The today trafficout of proxy."

		//registry
		desc = prometheus.NewDesc(name, description, Labels, nil)
		ch <- prometheus.MustNewConstMetric(desc, vType, float64(v.TodayTrafficOut), labelValue...)

		//get name
		name = "frps_" + v.Name + "_currconns"
		description = "The today currconns of proxy."

		//registry
		desc = prometheus.NewDesc(name, description, Labels, nil)
		ch <- prometheus.MustNewConstMetric(desc, vType, float64(v.CurConns), labelValue...)
	}

}

func (p *Prometheus) handle(xl *xlog.Logger, w http.ResponseWriter, r *http.Request) {
	//init config metrics
	setValue(p.config)

	//deal with server stat.
	ServerStats := p.statsCollector.GetServer()
	p.dealWithServerStats(ServerStats)

	var Proxys []*stats.ProxyStats
	for k, _ := range ServerStats.ProxyTypeCounts {
		ProxyStats := p.statsCollector.GetProxiesByType(k)
		Proxys = append(Proxys, ProxyStats...)
	}
	//NewProxyStats
	proxyStats := NewProxyStats(Proxys)
	//new registry
	registry := prometheus.NewRegistry()
	//registry exporter
	registry.MustRegister(proxyStats)
	registry.MustRegister(serverBindPort)
	registry.MustRegister(serverBindUdpPort)
	registry.MustRegister(serverVhostHttpPort)
	registry.MustRegister(serverVhostHttpsPort)
	registry.MustRegister(serverKcpBindPort)
	registry.MustRegister(serverMaxPoolCount)
	registry.MustRegister(serverMaxPortsPerClient)
	registry.MustRegister(serverHeartBeatTimeout)
	registry.MustRegister(serverClientCounts)
	registry.MustRegister(serverCurConns)
	registry.MustRegister(serverTotalTrafficIn)
	registry.MustRegister(serverTotalTrafficOut)
	//http server
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func (p *Prometheus) Start() {
	//init log
	xl := xlog.New()
	xl.Debug("Start To Export Data For Prometheus Metrics")

	//script metrics
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		p.handle(xl, w, r)
	})

	//root
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>frps Exporter</title></head>
			<body>
			<h1>frps Exporter</h1>
			<p><a href="\metrics">Metrics</a></p>
			</body>
			</html>`))
	})

	listenAddress := p.config.PromesAddr + ":" + strconv.Itoa(p.config.PromesPort)
	xl.Info("prometheus exporter listen on %s:%d", p.config.PromesAddr, p.config.PromesPort)

	//listen
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		xl.Error("Error starting HTTP server: %s", err)
	}

}

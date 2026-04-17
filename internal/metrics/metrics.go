// Package metrics 注册供应链服务 Prometheus 指标。
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	AdsTxtFetchTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "adortb_adstxt_fetch_total",
		Help: "Total ads.txt fetch attempts.",
	}, []string{"status"})

	AdsTxtDeclaredGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "adortb_adstxt_declared_publishers",
		Help: "Number of publishers that have declared this ADX in ads.txt.",
	})

	SellersJSONGenerateSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "adortb_sellersjson_generate_seconds",
		Help:    "Time to generate sellers.json.",
		Buckets: prometheus.DefBuckets,
	})

	SchainBuildTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "adortb_schain_build_total",
		Help: "Total SupplyChain objects built.",
	})

	SPOAnalyzeTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "adortb_spo_analyze_total",
		Help: "Total SPO analysis requests.",
	})

	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "adortb_supply_chain_http_requests_total",
		Help: "Total HTTP requests by path and status.",
	}, []string{"path", "method", "status"})
)

// Register 将所有指标注册到默认 Prometheus 注册表。
func Register() {
	prometheus.MustRegister(
		AdsTxtFetchTotal,
		AdsTxtDeclaredGauge,
		SellersJSONGenerateSeconds,
		SchainBuildTotal,
		SPOAnalyzeTotal,
		HTTPRequestsTotal,
	)
}

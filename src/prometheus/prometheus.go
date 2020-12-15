package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var logsTotal prometheus.Counter
var logsValidTotal prometheus.Counter

func init() {
	logsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "quotascraper_processed_logs_total",
		Help: "The total number of processed log lines",
	})
	logsValidTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "quotascraper_processed_validlogs_total",
		Help: "The total number of processed valid log lines",
	})
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":2112", nil)
}

func LogCountInc() {
	logsTotal.Inc()
}

func LogValidInc() {
	logsValidTotal.Inc()
}

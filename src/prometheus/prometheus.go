package prometheus

import (
	"fmt"
	"net/http"

	"github.com/NODO-UH/quota-scraper/src/configuration"
	"github.com/NODO-UH/quota-scraper/src/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var logsTotal prometheus.Counter
var logsValidTotal prometheus.Counter

func Start() {
	conf := configuration.GetConfiguration()
	logsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("quotascraper_%s_processed_logs_total", *conf.Id),
		Help: "The total number of processed log lines",
	})
	logsValidTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("quotascraper_%s_processed_validlogs_total", *conf.Id),
		Help: "The total number of processed valid log lines",
	})
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Error.Println(err)
		}
	}()
}

func LogCountInc() {
	logsTotal.Inc()
}

func LogValidInc() {
	logsValidTotal.Inc()
}

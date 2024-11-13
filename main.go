// prommonitoring.go
package prommonitoring

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Define a counter metric for the number of requests to an endpoint
	RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint"},
	)

	// Define a histogram metric for tracking response time
	ResponseDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_duration_seconds",
			Help:    "Histogram of response durations for HTTP requests",
			Buckets: prometheus.DefBuckets, // default buckets
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(RequestCounter)
	prometheus.MustRegister(ResponseDuration)
}

// ExposePrometheusMetrics is a helper function to expose the metrics at a specific endpoint.
func ExposePrometheusMetrics(mux *http.ServeMux) {
	mux.Handle("/metrics", promhttp.Handler())
}

package prommonitoring

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the HTTP service
type Metrics struct {
	RequestCounter   *prometheus.CounterVec
	ResponseDuration *prometheus.HistogramVec
	RequestSize      *prometheus.HistogramVec
	ResponseSize     *prometheus.HistogramVec
	RequestsInFlight *prometheus.GaugeVec
	TotalErrors      *prometheus.CounterVec
	RequestsByStatus *prometheus.CounterVec
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		RequestCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		ResponseDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request latency in seconds",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path", "status"},
		),
		RequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),
		ResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),
		RequestsInFlight: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "Current number of HTTP requests being processed",
			},
			[]string{"method"},
		),
		TotalErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_errors_total",
				Help:      "Total number of HTTP errors",
			},
			[]string{"method", "path", "error_type"},
		),
		RequestsByStatus: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_by_status",
				Help:      "HTTP requests partitioned by status code",
			},
			[]string{"status_class", "status_code"},
		),
	}
}

// ResponseWriter wrapper that captures additional metrics
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (w *metricsResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.responseSize += int64(size)
	return size, err
}

// Middleware creates a new middleware handler with the provided metrics
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track in-flight requests
		m.RequestsInFlight.WithLabelValues(r.Method).Inc()
		defer m.RequestsInFlight.WithLabelValues(r.Method).Dec()

		// Track request size
		if r.ContentLength > 0 {
			m.RequestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
		}

		// Wrap response writer to capture metrics
		metricsWriter := newMetricsResponseWriter(w)

		// Call the next handler
		next.ServeHTTP(metricsWriter, r)

		// Record duration
		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(metricsWriter.statusCode)
		statusClass := strconv.Itoa(metricsWriter.statusCode/100) + "xx"

		// Update metrics
		m.RequestCounter.WithLabelValues(r.Method, r.URL.Path, statusCode).Inc()
		m.ResponseDuration.WithLabelValues(r.Method, r.URL.Path, statusCode).Observe(duration)
		m.RequestsByStatus.WithLabelValues(statusClass, statusCode).Inc()

		// Track response size
		if metricsWriter.responseSize > 0 {
			m.ResponseSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(metricsWriter.responseSize))
		}

		// Track errors (status code >= 400)
		if metricsWriter.statusCode >= 400 {
			errorType := "client_error"
			if metricsWriter.statusCode >= 500 {
				errorType = "server_error"
			}
			m.TotalErrors.WithLabelValues(r.Method, r.URL.Path, errorType).Inc()
		}
	})
}

// RecoverMiddleware adds panic recovery and metrics
func (m *Metrics) RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.TotalErrors.WithLabelValues(r.Method, r.URL.Path, "panic").Inc()
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// prommiddleware.go
package prommonitoring

import (
	"net/http"
	"time"
)

// hitMetric is a middleware to count the number of requests for each endpoint
func HitMetric(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Count the request (method and endpoint as labels)
		RequestCounter.WithLabelValues(r.Method, r.URL.Path).Inc()

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

// responseMetric is a middleware to track the response time for each endpoint
func ResponseMetric(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record the start time for measuring response duration
		start := time.Now()

		// Create a response writer to capture status code
		rec := &statusCodeResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the actual handler
		next.ServeHTTP(rec, r)

		// Measure the response duration
		duration := time.Since(start).Seconds()
		ResponseDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

// statusCodeResponseWriter wraps the http.ResponseWriter to capture the status code
type statusCodeResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCodeResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

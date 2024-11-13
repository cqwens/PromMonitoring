// prommiddleware.go
package prommonitoring

import (
	"net/http"
	"strconv"
	"time"
)

// HitMetric is a middleware to count the number of requests for each endpoint, method, and status code
func HitMetric(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap the ResponseWriter to capture the status code
		ww := &statusCapturingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler in the chain
		next.ServeHTTP(ww, r)

		// Count the request (endpoint, and status code as labels)
		statusCode := strconv.Itoa(ww.statusCode)
		RequestCounter.WithLabelValues(r.URL.Path, statusCode).Inc()
	})
}

// responseMetric is a middleware to track the response time for each endpoint
func ResponseMetric(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record the start time for measuring response duration
		start := time.Now()

		// Create a response writer to capture status code
		rec := &statusCapturingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the actual handler
		next.ServeHTTP(rec, r)

		// Measure the response duration
		duration := time.Since(start).Seconds()
		ResponseDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

// Custom ResponseWriter to capture the status code
type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *statusCapturingResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

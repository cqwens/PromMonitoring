package prommonitoring

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config holds the configuration for the Prometheus monitoring setup
type Config struct {
	Namespace   string
	MetricsPath string
	Registry    *prometheus.Registry
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Namespace:   "app",
		MetricsPath: "/metrics",
		Registry:    prometheus.NewRegistry(),
	}
}

var (
	metrics     *Metrics
	metricsOnce sync.Once
)

// InitMetrics initializes the Prometheus metrics with the given configuration
func InitMetrics(cfg *Config) *Metrics {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	metricsOnce.Do(func() {
		metrics = NewMetrics(cfg.Namespace)

		// Register metrics with the registry
		if cfg.Registry != nil {
			cfg.Registry.MustRegister(
				metrics.RequestCounter,
				metrics.ResponseDuration,
				metrics.RequestSize,
				metrics.ResponseSize,
				metrics.RequestsInFlight,
				metrics.TotalErrors,
				metrics.RequestsByStatus,
			)
		}
	})

	return metrics
}

// GetMetrics returns the initialized metrics instance
func GetMetrics() *Metrics {
	if metrics == nil {
		return InitMetrics(nil)
	}
	return metrics
}

// MetricsHandler returns a handler for exposing Prometheus metrics
func MetricsHandler(cfg *Config) http.Handler {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Create a custom registry if one isn't provided
	if cfg.Registry == nil {
		cfg.Registry = prometheus.NewRegistry()
	}

	// Create handler options
	handlerOpts := promhttp.HandlerOpts{
		Registry:          cfg.Registry,
		EnableOpenMetrics: true,
	}

	return promhttp.HandlerFor(cfg.Registry, handlerOpts)
}

// SetupMetricsServer creates and configures a complete metrics server
func SetupMetricsServer(cfg *Config, middlewares ...func(http.Handler) http.Handler) *http.ServeMux {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Initialize metrics
	InitMetrics(cfg)

	// Create a new mux for metrics
	mux := http.NewServeMux()

	// Create the metrics handler
	handler := MetricsHandler(cfg)

	// Apply any additional middlewares
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}

	// Register the metrics handler
	mux.Handle(cfg.MetricsPath, handler)

	return mux
}

// Example usage:
/*
func main() {
    // Create custom configuration
    cfg := &prommonitoring.Config{
        Namespace:   "myapp",
        MetricsPath: "/metrics",
        Registry:    prometheus.NewRegistry(),
    }

    // Initialize metrics
    metrics := prommonitoring.InitMetrics(cfg)

    // Create your main application router
    mainRouter := http.NewServeMux()

    // Add your application routes
    mainRouter.HandleFunc("/api/v1/users", handleUsers)

    // Wrap with metrics middleware
    handler := metrics.RecoverMiddleware(metrics.Middleware(mainRouter))

    // Create metrics server
    metricsServer := prommonitoring.SetupMetricsServer(cfg)

    // Start servers
    go func() {
        log.Fatal(http.ListenAndServe(":8080", handler))
    }()

    go func() {
        log.Fatal(http.ListenAndServe(":9090", metricsServer))
    }()

    // Wait for shutdown signal
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
    <-signalChan
}
*/

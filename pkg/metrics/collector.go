package metrics

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector manages all metrics for ProofBox
type Collector struct {
	jmtMetrics     JMTMetrics
	enabled        bool
	httpServerOnce sync.Once
}

// NewCollector creates a new metrics collector
func NewCollector(enabled bool) *Collector {
	var jmtMetrics JMTMetrics
	if enabled {
		jmtMetrics = NewPrometheusMetrics()
	} else {
		jmtMetrics = NoOpMetrics{}
	}

	return &Collector{
		jmtMetrics: jmtMetrics,
		enabled:    enabled,
	}
}

// JMT returns the JMT metrics interface
func (c *Collector) JMT() JMTMetrics {
	return c.jmtMetrics
}

// IsEnabled returns whether metrics collection is enabled
func (c *Collector) IsEnabled() bool {
	return c.enabled
}

// ServeHTTP serves the Prometheus metrics endpoint
func (c *Collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !c.enabled {
		http.Error(w, "Metrics not enabled", http.StatusNotFound)
		return
	}
	promhttp.Handler().ServeHTTP(w, r)
}

// StartHTTPServer starts an HTTP server for metrics on the specified address
// Deprecated: Use NewMetricsServer instead for proper lifecycle management
func (c *Collector) StartHTTPServer(addr string) error {
	if !c.enabled {
		return nil // No-op if metrics disabled
	}

	server := NewMetricsServer(addr, c)
	return server.Start()
}
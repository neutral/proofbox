package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsServer provides an HTTP server for exposing Prometheus metrics
type MetricsServer struct {
	server   *http.Server
	enabled  bool
	collector *Collector
}

// NewMetricsServer creates a new metrics HTTP server
func NewMetricsServer(addr string, collector *Collector) *MetricsServer {
	if collector == nil || !collector.IsEnabled() {
		return &MetricsServer{enabled: false}
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	
	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &MetricsServer{
		server:    server,
		enabled:   true,
		collector: collector,
	}
}

// Start starts the metrics server in a goroutine
func (s *MetricsServer) Start() error {
	if !s.enabled {
		return nil // No-op if metrics disabled
	}

	go func() {
		log.Printf("Starting metrics server on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully stops the metrics server
func (s *MetricsServer) Stop(ctx context.Context) error {
	if !s.enabled || s.server == nil {
		return nil
	}

	log.Printf("Stopping metrics server...")
	return s.server.Shutdown(ctx)
}

// GetAddr returns the server address
func (s *MetricsServer) GetAddr() string {
	if !s.enabled || s.server == nil {
		return ""
	}
	return s.server.Addr
}

// IsEnabled returns whether the metrics server is enabled
func (s *MetricsServer) IsEnabled() bool {
	return s.enabled
}

// ServeMetricsHTTP is a convenience function to serve metrics on a simple HTTP server
func ServeMetricsHTTP(addr string, metricsEnabled bool) error {
	if !metricsEnabled {
		return fmt.Errorf("metrics not enabled")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("Serving metrics on http://%s/metrics", addr)
	return http.ListenAndServe(addr, mux)
}
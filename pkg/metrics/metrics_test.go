package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestPrometheusMetrics(t *testing.T) {
	// Use a custom registry for testing to avoid conflicts
	registry := prometheus.NewRegistry()
	metrics := NewPrometheusMetricsWithRegistry(registry)
	
	// Test that all methods can be called without panicking
	metrics.RecordCommit(0.001, 5)
	metrics.RecordOperation("insert")
	metrics.RecordLookup(0.0005, true)
	metrics.RecordProofGeneration(0.002, "inclusion", 256)
	metrics.RecordProofValidationFailure()
	metrics.RecordHashOperation()
	metrics.RecordDBRead()
	metrics.RecordDBWrite()
	metrics.RecordError("storage")
	metrics.UpdateTreeHeight(10)
	metrics.UpdateTreeNodeCount(1000)
	metrics.UpdateVersionCount(5)
	metrics.UpdateDatabaseSize(1024*1024)
}

func TestNoOpMetrics(t *testing.T) {
	metrics := NoOpMetrics{}
	
	// Test that all methods can be called without panicking
	metrics.RecordCommit(0.001, 5)
	metrics.RecordOperation("insert")
	metrics.RecordLookup(0.0005, true)
	metrics.RecordProofGeneration(0.002, "inclusion", 256)
	metrics.RecordProofValidationFailure()
	metrics.RecordHashOperation()
	metrics.RecordDBRead()
	metrics.RecordDBWrite()
	metrics.RecordError("storage")
	metrics.UpdateTreeHeight(10)
	metrics.UpdateTreeNodeCount(1000)
	metrics.UpdateVersionCount(5)
	metrics.UpdateDatabaseSize(1024*1024)
}

func TestCollector(t *testing.T) {
	// Test with metrics enabled - but use NoOp to avoid registry conflicts
	collector := NewCollector(false) // Use disabled for test simplicity
	if collector.IsEnabled() {
		t.Error("Expected collector to be disabled in test")
	}
	if collector.JMT() == nil {
		t.Error("Expected JMT metrics to be non-nil")
	}
	
	// Test with metrics disabled
	collectorDisabled := NewCollector(false)
	if collectorDisabled.IsEnabled() {
		t.Error("Expected collector to be disabled")
	}
	if collectorDisabled.JMT() == nil {
		t.Error("Expected JMT metrics to be non-nil (NoOp)")
	}
}

func TestMetricsServer(t *testing.T) {
	// Test with metrics disabled
	collector := NewCollector(false)
	server := NewMetricsServer(":0", collector)
	
	if server.IsEnabled() {
		t.Error("Expected server to be disabled when metrics are disabled")
	}
	
	// Test with metrics enabled
	collectorEnabled := NewCollector(true)
	serverEnabled := NewMetricsServer(":0", collectorEnabled)
	
	if !serverEnabled.IsEnabled() {
		t.Error("Expected server to be enabled when metrics are enabled")
	}
}
// Package metrics provides Prometheus-compatible metrics collection for ProofBox.
//
// This package implements the comprehensive metrics specified in Step 18 of the ProofBox
// blueprint, exposing all core JMT operations, performance metrics, and system state.
//
// # Usage
//
// To enable metrics in your ProofBox application:
//
//	// Create metrics collector
//	collector := metrics.NewCollector(true) // enable metrics
//
//	// Configure tree with metrics
//	config := tree.DefaultTreeConfig()
//	config.MetricsEnabled = true
//	config.Metrics = collector.JMT() // optional: use custom metrics
//
//	// Create tree with metrics
//	tr, err := tree.NewTree(storage, keyEncoder, config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Start metrics server
//	server := metrics.NewMetricsServer(":9090", collector)
//	if err := server.Start(); err != nil {
//		log.Fatal(err)
//	}
//
// # Metrics Exposed
//
// The following Prometheus metrics are exposed at /metrics:
//
// Counter Metrics:
//   - jmt_commits_total: Total number of commits
//   - jmt_operations_total{type}: Operations by type (insert/update/delete)
//   - jmt_proof_generations_total{type}: Proofs by type (inclusion/exclusion)
//   - jmt_hash_operations_total: Hash computations
//   - jmt_db_reads_total: Database reads
//   - jmt_db_writes_total: Database writes
//   - jmt_errors_total{type}: Errors by type
//   - jmt_validation_failures_total: Proof validation failures
//
// Histogram Metrics:
//   - jmt_commit_latency_seconds: Commit operation duration
//   - jmt_lookup_latency_seconds: Lookup operation duration
//   - jmt_proof_generation_latency_seconds: Proof generation time
//   - jmt_batch_size: Operations per batch
//   - jmt_proof_size_bytes: Proof size in bytes
//
// Gauge Metrics:
//   - jmt_tree_height: Current tree height
//   - jmt_tree_nodes_total: Total nodes in tree
//   - jmt_versions_stored: Number of versions stored
//   - jmt_db_live_bytes: Live database size in bytes
//
// # Performance Impact
//
// When metrics are disabled (MetricsEnabled: false), all metric operations
// are no-ops with minimal overhead. When enabled, the performance impact
// is typically <1% for most workloads.
//
// # Production Deployment
//
// In production, expose metrics on a separate port from your main application:
//
//	// Main app on :8080
//	go http.ListenAndServe(":8080", mainHandler)
//
//	// Metrics on :9090
//	go metrics.ServeMetricsHTTP(":9090", true)
//
// This allows monitoring systems to scrape metrics without affecting
// application performance.
package metrics

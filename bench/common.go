package bench

import "github.com/neutral/proofbox/pkg/tree"

// BenchmarkTreeConfig returns a tree configuration suitable for benchmarks
func BenchmarkTreeConfig() tree.TreeConfig {
	config := tree.DefaultTreeConfig()
	config.MetricsEnabled = false // Disable metrics in benchmarks to avoid duplicate registration
	return config
}

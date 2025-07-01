package storage

import (
	"time"
)

// Config defines storage backend configuration.
type Config struct {
	// Backend selects the storage implementation
	Backend string `json:"backend"` // "pebble", "memory"

	// Path is the directory for persistent storage (ignored for memory backend)
	Path string `json:"path"`

	// PebbleDB specific configuration
	Pebble PebbleConfig `json:"pebble"`

	// Metrics configuration
	Metrics MetricsConfig `json:"metrics"`

	// Pruning configuration
	Pruning PruningConfig `json:"pruning"`

	// Key encoder selection (defaults to "default")
	KeyEncoder string `json:"key_encoder"`
}

// PebbleConfig contains PebbleDB-specific settings.
type PebbleConfig struct {
	// CacheSize is the size of the block cache in bytes
	CacheSize int64 `json:"cache_size"`

	// WriteBufferSize is the size of the memtable in bytes
	WriteBufferSize int `json:"write_buffer_size"`

	// MaxOpenFiles limits the number of open file descriptors
	MaxOpenFiles int `json:"max_open_files"`

	// Compression algorithm: "snappy", "zstd", "none"
	Compression string `json:"compression"`

	// DisableWAL disables write-ahead logging (not recommended)
	DisableWAL bool `json:"disable_wal"`

	// MaxConcurrentCompactions limits background compactions
	MaxConcurrentCompactions int `json:"max_concurrent_compactions"`
}

// MetricsConfig controls metrics collection.
type MetricsConfig struct {
	// Enable metrics collection
	Enabled bool `json:"enabled"`

	// Interval for periodic metrics reporting
	ReportInterval time.Duration `json:"report_interval"`

	// Maximum samples to keep for latency percentiles
	MaxLatencySamples int `json:"max_latency_samples"`
}

// PruningConfig defines version pruning behavior.
type PruningConfig struct {
	// Enable automatic pruning
	Enabled bool `json:"enabled"`

	// Interval between pruning runs
	Interval time.Duration `json:"interval"`

	// Policy determines what to prune
	Policy PruningPolicyConfig `json:"policy"`
}

// PruningPolicyConfig specifies the pruning strategy.
type PruningPolicyConfig struct {
	// Type of pruning policy
	Type string `json:"type"` // "keep_last_n", "keep_newer_than", "keep_milestones"

	// KeepLastN configuration
	KeepVersions int `json:"keep_versions"`

	// KeepNewerThan configuration
	MaxAge time.Duration `json:"max_age"`

	// KeepMilestones configuration
	MilestoneInterval int `json:"milestone_interval"`
}

// DefaultConfig returns a production-ready configuration.
func DefaultConfig(path string) Config {
	return Config{
		Backend: "pebble",
		Path:    path,
		Pebble: PebbleConfig{
			CacheSize:                64 << 20, // 64MB
			WriteBufferSize:          32 << 20, // 32MB
			MaxOpenFiles:             1000,
			Compression:              "snappy",
			DisableWAL:               false,
			MaxConcurrentCompactions: 3,
		},
		Metrics: MetricsConfig{
			Enabled:           true,
			ReportInterval:    time.Minute,
			MaxLatencySamples: 10000,
		},
		Pruning: PruningConfig{
			Enabled:  false, // Disabled by default
			Interval: time.Hour,
			Policy: PruningPolicyConfig{
				Type:         "keep_last_n",
				KeepVersions: 100,
			},
		},
		KeyEncoder: "default",
	}
}

// MemoryConfig returns configuration for in-memory storage.
func MemoryConfig() Config {
	return Config{
		Backend: "memory",
		Metrics: MetricsConfig{
			Enabled: false, // Usually not needed for tests
		},
		Pruning: PruningConfig{
			Enabled: false,
		},
		KeyEncoder: "default",
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	switch c.Backend {
	case "pebble", "memory":
		// Valid backends
	default:
		return ErrInvalidBackend
	}

	if c.Backend == "pebble" && c.Path == "" {
		return ErrPathRequired
	}

	if c.Pebble.CacheSize < 0 {
		return ErrInvalidCacheSize
	}

	if c.Pebble.WriteBufferSize < 0 {
		return ErrInvalidWriteBufferSize
	}

	switch c.Pebble.Compression {
	case "snappy", "zstd", "none", "":
		// Valid compression types (empty means default)
	default:
		return ErrInvalidCompression
	}

	switch c.Pruning.Policy.Type {
	case "keep_last_n", "keep_newer_than", "keep_milestones", "":
		// Valid policy types (empty means disabled)
	default:
		return ErrInvalidPruningPolicy
	}

	return nil
}
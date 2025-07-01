package corpus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/neutral/proofbox/pkg/fuzz/generators"
	"github.com/neutral/proofbox/pkg/types"
)

// CorpusEntry represents a single test case in the corpus
type CorpusEntry struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Operations  []generators.Operation  `json:"operations"`
	Tags        []string                `json:"tags"`
}

// CorpusBuilder helps create interesting test cases
type CorpusBuilder struct {
	entries []CorpusEntry
	dir     string
}

// NewCorpusBuilder creates a new corpus builder
func NewCorpusBuilder(dir string) *CorpusBuilder {
	return &CorpusBuilder{
		entries: make([]CorpusEntry, 0),
		dir:     dir,
	}
}

// AddEntry adds a test case to the corpus
func (cb *CorpusBuilder) AddEntry(entry CorpusEntry) {
	cb.entries = append(cb.entries, entry)
}

// BuildEdgeCases creates corpus entries for edge cases
func (cb *CorpusBuilder) BuildEdgeCases() {
	// Empty operations
	cb.AddEntry(CorpusEntry{
		Name:        "empty_ops",
		Description: "No operations",
		Operations:  []generators.Operation{},
		Tags:        []string{"edge", "empty"},
	})
	
	// Single operation types
	cb.AddEntry(CorpusEntry{
		Name:        "single_put",
		Description: "Single put operation",
		Operations: []generators.Operation{
			{Type: generators.OpPut, Key: types.KeyHash([]byte("key1")), Value: []byte("value1")},
		},
		Tags: []string{"single", "put"},
	})
	
	// Put then delete same key
	key := types.KeyHash([]byte("test-key"))
	cb.AddEntry(CorpusEntry{
		Name:        "put_delete_same",
		Description: "Put then delete same key",
		Operations: []generators.Operation{
			{Type: generators.OpPut, Key: key, Value: []byte("value")},
			{Type: generators.OpDelete, Key: key},
		},
		Tags: []string{"edge", "delete"},
	})
	
	// Multiple puts to same key
	cb.AddEntry(CorpusEntry{
		Name:        "overwrite_key",
		Description: "Multiple puts to same key",
		Operations: []generators.Operation{
			{Type: generators.OpPut, Key: key, Value: []byte("value1")},
			{Type: generators.OpPut, Key: key, Value: []byte("value2")},
			{Type: generators.OpPut, Key: key, Value: []byte("value3")},
		},
		Tags: []string{"overwrite", "put"},
	})
	
	// Keys with common prefixes (tests internal node sharing)
	cb.AddEntry(CorpusEntry{
		Name:        "common_prefix_keys",
		Description: "Keys sharing common prefix",
		Operations: []generators.Operation{
			{Type: generators.OpPut, Key: types.Key{0xAB, 0xCD, 0xEF, 0x00}, Value: []byte("v1")},
			{Type: generators.OpPut, Key: types.Key{0xAB, 0xCD, 0xEF, 0x01}, Value: []byte("v2")},
			{Type: generators.OpPut, Key: types.Key{0xAB, 0xCD, 0xEF, 0x02}, Value: []byte("v3")},
			{Type: generators.OpPut, Key: types.Key{0xAB, 0xCD, 0xEF, 0x03}, Value: []byte("v4")},
		},
		Tags: []string{"prefix", "structural-sharing"},
	})
}

// BuildVersionIsolationCases creates test cases for version isolation
func (cb *CorpusBuilder) BuildVersionIsolationCases() {
	// Version isolation regression test
	key1 := types.KeyHash([]byte("key1"))
	key2 := types.KeyHash([]byte("key2"))
	
	cb.AddEntry(CorpusEntry{
		Name:        "version_isolation_basic",
		Description: "Basic version isolation test",
		Operations: []generators.Operation{
			{Type: generators.OpPut, Key: key1, Value: []byte("v1")},
			{Type: generators.OpGet, Key: key2, Version: 0}, // Should not see key1
			{Type: generators.OpPut, Key: key2, Value: []byte("v2")},
			{Type: generators.OpGet, Key: key1, Version: 1}, // Should see key1
			{Type: generators.OpGet, Key: key2, Version: 1}, // Should not see key2
		},
		Tags: []string{"version-isolation", "regression"},
	})
}

// BuildPerformanceCases creates performance-testing corpus entries
func (cb *CorpusBuilder) BuildPerformanceCases() {
	// Large batch operations
	ops := make([]generators.Operation, 1000)
	for i := 0; i < 1000; i++ {
		ops[i] = generators.Operation{
			Type:  generators.OpPut,
			Key:   types.KeyHash([]byte(fmt.Sprintf("batch-key-%d", i))),
			Value: []byte(fmt.Sprintf("batch-value-%d", i)),
		}
	}
	
	cb.AddEntry(CorpusEntry{
		Name:        "large_batch",
		Description: "Large batch of put operations",
		Operations:  ops,
		Tags:        []string{"performance", "batch"},
	})
	
	// Deep tree structure
	deepOps := make([]generators.Operation, 100)
	baseKey := make([]byte, 32)
	for i := range baseKey {
		baseKey[i] = 0xFF
	}
	
	for i := 0; i < 100; i++ {
		key := make([]byte, 32)
		copy(key, baseKey)
		key[31] = byte(i)
		
		deepOps[i] = generators.Operation{
			Type:  generators.OpPut,
			Key:   types.Key(key),
			Value: []byte{byte(i)},
		}
	}
	
	cb.AddEntry(CorpusEntry{
		Name:        "deep_tree",
		Description: "Creates deep tree structure",
		Operations:  deepOps,
		Tags:        []string{"performance", "deep-tree"},
	})
}

// BuildSecurityCases creates security-focused test cases
func (cb *CorpusBuilder) BuildSecurityCases() {
	// Maximum size values
	maxValue := make([]byte, 1024*100) // 100KB
	for i := range maxValue {
		maxValue[i] = byte(i % 256)
	}
	
	cb.AddEntry(CorpusEntry{
		Name:        "large_values",
		Description: "Operations with large values",
		Operations: []generators.Operation{
			{Type: generators.OpPut, Key: types.KeyHash([]byte("large1")), Value: maxValue},
			{Type: generators.OpPut, Key: types.KeyHash([]byte("large2")), Value: maxValue},
		},
		Tags: []string{"security", "dos", "large-value"},
	})
	
	// Adversarial key patterns
	cb.AddEntry(CorpusEntry{
		Name:        "adversarial_keys",
		Description: "Keys designed to cause collisions",
		Operations: []generators.Operation{
			{Type: generators.OpPut, Key: types.Key{}, Value: []byte("empty-key")},
			{Type: generators.OpPut, Key: types.Key{0xFF, 0xFF, 0xFF, 0xFF}, Value: []byte("max-prefix")},
			{Type: generators.OpPut, Key: types.Key{0x00, 0x00, 0x00, 0x00}, Value: []byte("min-prefix")},
		},
		Tags: []string{"security", "adversarial"},
	})
}

// Save writes the corpus to disk
// Note: The corpus directory should be gitignored as it contains generated test data
func (cb *CorpusBuilder) Save() error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(cb.dir, 0755); err != nil {
		return fmt.Errorf("failed to create corpus directory: %w", err)
	}
	
	// Save each entry as a separate file
	for i, entry := range cb.entries {
		filename := filepath.Join(cb.dir, fmt.Sprintf("%03d_%s.json", i, entry.Name))
		
		data, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal entry %s: %w", entry.Name, err)
		}
		
		if err := os.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("failed to write entry %s: %w", entry.Name, err)
		}
	}
	
	// Create index file
	indexFile := filepath.Join(cb.dir, "index.json")
	index := struct {
		TotalEntries int      `json:"total_entries"`
		Tags         []string `json:"tags"`
		Entries      []string `json:"entries"`
	}{
		TotalEntries: len(cb.entries),
		Tags:         cb.collectTags(),
		Entries:      cb.collectNames(),
	}
	
	indexData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}
	
	return os.WriteFile(indexFile, indexData, 0644)
}

// collectTags returns all unique tags
func (cb *CorpusBuilder) collectTags() []string {
	tagSet := make(map[string]bool)
	for _, entry := range cb.entries {
		for _, tag := range entry.Tags {
			tagSet[tag] = true
		}
	}
	
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	return tags
}

// collectNames returns all entry names
func (cb *CorpusBuilder) collectNames() []string {
	names := make([]string, len(cb.entries))
	for i, entry := range cb.entries {
		names[i] = entry.Name
	}
	return names
}

// LoadCorpus loads corpus entries from disk
func LoadCorpus(dir string) ([]CorpusEntry, error) {
	pattern := filepath.Join(dir, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list corpus files: %w", err)
	}
	
	var entries []CorpusEntry
	for _, file := range files {
		// Skip index file
		if filepath.Base(file) == "index.json" {
			continue
		}
		
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}
		
		var entry CorpusEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", file, err)
		}
		
		entries = append(entries, entry)
	}
	
	return entries, nil
}
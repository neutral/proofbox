package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/neutral/proofbox/pkg/storage"
	pebblestorage "github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

// Colors for terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"
)

// PrintSection prints a section header
func PrintSection(title string) {
	fmt.Printf("\n%s=== %s ===%s\n", ColorBlue, title, ColorReset)
}

// PrintSubSection prints a subsection header
func PrintSubSection(title string) {
	fmt.Printf("\n%s--- %s ---%s\n", ColorCyan, title, ColorReset)
}

// PrintSuccess prints a success message
func PrintSuccess(format string, args ...interface{}) {
	fmt.Printf("%s✓ %s%s\n", ColorGreen, fmt.Sprintf(format, args...), ColorReset)
}

// PrintError prints an error message
func PrintError(format string, args ...interface{}) {
	fmt.Printf("%s✗ %s%s\n", ColorRed, fmt.Sprintf(format, args...), ColorReset)
}

// PrintInfo prints an info message
func PrintInfo(format string, args ...interface{}) {
	fmt.Printf("%s→ %s%s\n", ColorYellow, fmt.Sprintf(format, args...), ColorReset)
}

// PrintKeyValue prints a key-value pair
func PrintKeyValue(key string, value interface{}) {
	fmt.Printf("%s%s:%s %v\n", ColorGray, key, ColorReset, value)
}

// PrintBatchOperation prints a batch operation
func PrintBatchOperation(op string, key []byte, value []byte) {
	if value != nil {
		fmt.Printf("  %s[%s]%s %s → %s\n", ColorPurple, op, ColorReset, key, value)
	} else {
		fmt.Printf("  %s[%s]%s %s\n", ColorPurple, op, ColorReset, key)
	}
}

// CreateTempStorage creates temporary storage for examples
func CreateTempStorage(name string) (storage.Storage, func(), error) {
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("proofbox-example-%s", name))
	
	// Clean up any existing directory
	os.RemoveAll(tmpDir)
	
	// Create new directory
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, nil, err
	}
	
	// Create PebbleDB storage using the proper driver
	opts := &pebblestorage.Options{
		EnableMetrics: false,
	}
	
	store, err := pebblestorage.NewStorage(tmpDir, opts)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, nil, err
	}
	
	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}
	
	return store, cleanup, nil
}


// CreateExampleTree creates a tree with some initial data
func CreateExampleTree(store storage.Storage) (*tree.Tree, error) {
	config := tree.DefaultTreeConfig()
	keyEncoder := storage.NewDefaultKeyEncoder()
	jmt, err := tree.NewTree(store, keyEncoder, config)
	if err != nil {
		return nil, err
	}
	
	// Add some initial data
	initialData := map[string]string{
		"apple":  "fruit",
		"banana": "fruit",
		"carrot": "vegetable",
		"dog":    "animal",
		"eagle":  "bird",
	}
	
	PrintInfo("Creating tree with initial data:")
	for k, v := range initialData {
		key := types.KeyHash([]byte(k))
		value := []byte(v)
		version, err := jmt.Put(key, value)
		if err != nil {
			return nil, err
		}
		PrintKeyValue(fmt.Sprintf("  v%d", version), fmt.Sprintf("%s = %s", k, v))
	}
	
	return jmt, nil
}

// PrintTreeStats prints statistics about the tree
func PrintTreeStats(jmt *tree.Tree) {
	latestVersion := jmt.GetLatestVersion()
	
	PrintSubSection("Tree Statistics")
	PrintKeyValue("Latest Version", latestVersion)
	
	// In a real implementation, we'd have a method to iterate keys
	// For now, we'll just show the version info
	
	rootHash, err := jmt.GetRootHash(latestVersion)
	if err == nil {
		PrintKeyValue("Root Hash", fmt.Sprintf("%x", rootHash[:8])+"...")
	}
}

// PrintVersionComparison shows differences between versions
func PrintVersionComparison(jmt *tree.Tree, v1, v2 types.Version, keys []string) {
	PrintSubSection(fmt.Sprintf("Comparing Version %d and Version %d", v1, v2))
	
	for _, k := range keys {
		key := types.KeyHash([]byte(k))
		
		val1, err1 := jmt.GetAtVersion(v1, key)
		val2, err2 := jmt.GetAtVersion(v2, key)
		
		if err1 == nil && err2 == nil {
			if string(val1) == string(val2) {
				fmt.Printf("  %s: %s%s%s (unchanged)\n", k, ColorGray, val1, ColorReset)
			} else {
				fmt.Printf("  %s: %s%s%s → %s%s%s\n", k, ColorRed, val1, ColorReset, ColorGreen, val2, ColorReset)
			}
		} else if err1 != nil && err2 == nil {
			fmt.Printf("  %s: %s(none)%s → %s%s%s\n", k, ColorGray, ColorReset, ColorGreen, val2, ColorReset)
		} else if err1 == nil && err2 != nil {
			fmt.Printf("  %s: %s%s%s → %s(deleted)%s\n", k, ColorRed, val1, ColorReset, ColorGray, ColorReset)
		}
	}
}

// FormatBytes formats byte size in human readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// RepeatString repeats a string n times
func RepeatString(s string, n int) string {
	return strings.Repeat(s, n)
}

// CheckError checks if an error occurred and prints it
func CheckError(err error, context string) bool {
	if err != nil {
		PrintError("%s: %v", context, err)
		return true
	}
	return false
}
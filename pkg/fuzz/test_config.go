package fuzz

import (
	"os"
	"strconv"
	"testing"
)

// TestLevel represents the testing level
type TestLevel string

const (
	TestLevelQuick         TestLevel = "quick"
	TestLevelStandard      TestLevel = "standard"
	TestLevelComprehensive TestLevel = "comprehensive"
	TestLevelStress        TestLevel = "stress"
)

// TestConfig holds configuration for fuzzing tests
type TestConfig struct {
	Level      TestLevel
	Iterations int
}

// GetTestConfig returns the test configuration based on environment variables
func GetTestConfig() TestConfig {
	// Check for explicit iteration count first
	if iterStr := os.Getenv("FUZZ_ITERATIONS"); iterStr != "" {
		if iter, err := strconv.Atoi(iterStr); err == nil && iter > 0 {
			return TestConfig{
				Level:      TestLevelCustom,
				Iterations: iter,
			}
		}
	}

	// Check for test level
	level := TestLevel(os.Getenv("FUZZ_LEVEL"))
	switch level {
	case TestLevelQuick:
		return TestConfig{Level: TestLevelQuick, Iterations: 20}
	case TestLevelComprehensive:
		return TestConfig{Level: TestLevelComprehensive, Iterations: 1000}
	case TestLevelStress:
		return TestConfig{Level: TestLevelStress, Iterations: 10000}
	default:
		// Standard is the default
		return TestConfig{Level: TestLevelStandard, Iterations: 100}
	}
}

// GetTestConfigWithShort returns test configuration respecting testing.Short()
func GetTestConfigWithShort(t *testing.T) TestConfig {
	config := GetTestConfig()

	// If testing.Short() is set, reduce iterations
	if testing.Short() {
		switch config.Level {
		case TestLevelStandard:
			config.Iterations = 20
		case TestLevelComprehensive:
			config.Iterations = 50
		case TestLevelStress:
			config.Iterations = 100
		}
		// Quick level stays the same (20)
	}

	return config
}

// ApplyToRapid configures rapid testing with the appropriate iteration count
// Note: Rapid uses the -rapid.checks flag for iteration count, so we primarily
// use this for logging and conditional test execution
func (tc TestConfig) ApplyToRapid(t *testing.T) {
	// In rapid, we use the -rapid.checks flag, but we can also skip tests
	// if they're too heavy for the current level
	if testing.Short() && tc.Level != TestLevelQuick {
		t.Skip("Skipping non-quick test in short mode")
	}

	// Log the configuration for visibility
	t.Logf("Running with test level: %s, target iterations: %d (use -rapid.checks=%d to set)", tc.Level, tc.Iterations, tc.Iterations)
}

const TestLevelCustom TestLevel = "custom"

// ShouldRunStressTest returns true if stress tests should run
func (tc TestConfig) ShouldRunStressTest() bool {
	return tc.Level == TestLevelStress || tc.Level == TestLevelComprehensive
}

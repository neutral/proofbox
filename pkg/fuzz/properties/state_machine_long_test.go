//go:build !short
// +build !short

package properties

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/neutral/proofbox/pkg/fuzz"
	"github.com/neutral/proofbox/pkg/types"
	"pgregory.net/rapid"
)

// TestTreeStateMachineVersionIsolationExtended is an extended version that runs more iterations
// This test is skipped in short mode
func TestTreeStateMachineVersionIsolationExtended(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping extended test in short mode")
	}

	// Force comprehensive testing for this extended test
	config := fuzz.TestConfig{
		Level:      fuzz.TestLevelComprehensive,
		Iterations: 500,
	}
	config.ApplyToRapid(t)

	rapid.Check(t, func(t *rapid.T) {
		m := &treeStateMachine{}
		m.Init(t)

		// Run more operations for comprehensive testing
		numOps := rapid.IntRange(100, 200).Draw(t, "num_operations")
		for i := 0; i < numOps; i++ {
			// Mostly puts with some gets
			if rapid.Float64Range(0, 1).Draw(t, "op_choice") < 0.7 {
				m.Put(t)
			} else {
				m.Get(t)
			}
		}

		// Now verify version isolation across all versions
		for version, expectedState := range m.versionStates {
			for keyStr, expectedValue := range expectedState {
				// keyStr is hex-encoded
				keyBytes, err := hex.DecodeString(keyStr)
				if err != nil || len(keyBytes) != 32 {
					continue
				}
				key, err := types.KeyFromBytes(keyBytes)
				if err != nil {
					continue
				}

				actualValue, err := m.tree.GetAtVersion(version, key)
				if err != nil {
					t.Fatalf("failed to get at version %d: %v", version, err)
				}

				if !bytes.Equal(actualValue, expectedValue) {
					t.Fatalf("version isolation violated at version %d: key %s has value %v, expected %v",
						version, keyStr, actualValue, expectedValue)
				}
			}
		}
	})
}

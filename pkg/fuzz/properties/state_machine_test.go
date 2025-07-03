package properties

import (
	"bytes"
	"testing"

	"github.com/neutral/proofbox/pkg/fuzz"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
	"pgregory.net/rapid"
)

// treeStateMachine models the expected behavior of the tree
type treeStateMachine struct {
	tree           *tree.Tree
	expectedState  map[string][]byte                   // current state (key -> value)
	versionStates  map[types.Version]map[string][]byte // state at each version
	currentVersion types.Version
}

// Init initializes the state machine
func (m *treeStateMachine) Init(t *rapid.T) {
	tr, err := fuzz.NewTestTree()
	if err != nil {
		t.Fatalf("failed to create tree: %v", err)
	}

	m.tree = tr
	m.expectedState = make(map[string][]byte)
	m.versionStates = make(map[types.Version]map[string][]byte)
	m.currentVersion = 0

	// Save initial empty state
	m.versionStates[0] = make(map[string][]byte)
}

// Put models a put operation
func (m *treeStateMachine) Put(t *rapid.T) {
	// Generate key and value
	key := rapid.SliceOfN(rapid.Byte(), 1, 32).Draw(t, "key")
	value := rapid.SliceOfN(rapid.Byte(), 1, 256).Draw(t, "value")

	// Execute on real tree
	k := types.KeyHash(key)
	newVersion, err := m.tree.Put(k, value)
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	// Update model state
	m.expectedState[string(key)] = value
	m.currentVersion = newVersion

	// Save snapshot of current state
	snapshot := make(map[string][]byte)
	for k, v := range m.expectedState {
		snapshot[k] = v
	}
	m.versionStates[newVersion] = snapshot

	// Verify current state matches
	m.checkCurrentState(t)
}

// Get models a get operation
func (m *treeStateMachine) Get(t *rapid.T) {
	if len(m.expectedState) == 0 && m.currentVersion == 0 {
		// Nothing to get from empty tree
		return
	}

	// Sometimes get an existing key, sometimes a non-existent one
	var key []byte
	if len(m.expectedState) > 0 && rapid.Bool().Draw(t, "get_existing") {
		// Pick an existing key
		keys := make([]string, 0, len(m.expectedState))
		for k := range m.expectedState {
			keys = append(keys, k)
		}
		keyStr := rapid.SampledFrom(keys).Draw(t, "existing_key")
		key = []byte(keyStr)
	} else {
		// Generate a random key
		key = rapid.SliceOfN(rapid.Byte(), 1, 32).Draw(t, "random_key")
	}

	// Get from current version
	k := types.KeyHash(key)
	value, err := m.tree.Get(m.currentVersion, k)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	// Verify against expected state
	expectedValue := m.expectedState[string(key)]
	if !bytes.Equal(value, expectedValue) {
		t.Fatalf("get mismatch: expected %v, got %v", expectedValue, value)
	}

	// Also test historical gets
	if m.currentVersion > 0 {
		version := types.Version(rapid.Uint64Range(0, uint64(m.currentVersion)).Draw(t, "historical_version"))
		historicalValue, err := m.tree.GetAtVersion(version, k)
		if err != nil {
			t.Fatalf("historical get failed: %v", err)
		}

		// Check against historical state
		if historicalState, ok := m.versionStates[version]; ok {
			expectedHistorical := historicalState[string(key)]
			if !bytes.Equal(historicalValue, expectedHistorical) {
				t.Fatalf("historical get mismatch at version %d: expected %v, got %v",
					version, expectedHistorical, historicalValue)
			}
		}
	}
}

// Delete models a delete operation
func (m *treeStateMachine) Delete(t *rapid.T) {
	if len(m.expectedState) == 0 {
		// Nothing to delete
		return
	}

	// Pick a key to delete
	keys := make([]string, 0, len(m.expectedState))
	for k := range m.expectedState {
		keys = append(keys, k)
	}
	keyStr := rapid.SampledFrom(keys).Draw(t, "key_to_delete")

	// Execute delete
	k := types.KeyHash([]byte(keyStr))
	newVersion, err := m.tree.Delete(k)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Update model
	delete(m.expectedState, keyStr)
	m.currentVersion = newVersion

	// Save snapshot
	snapshot := make(map[string][]byte)
	for k, v := range m.expectedState {
		snapshot[k] = v
	}
	m.versionStates[newVersion] = snapshot

	// Verify state
	m.checkCurrentState(t)
}

// checkCurrentState verifies the tree matches our model
func (m *treeStateMachine) checkCurrentState(t *rapid.T) {
	// Check all keys in our model are in the tree
	for keyStr, expectedValue := range m.expectedState {
		k := types.KeyHash([]byte(keyStr))
		value, err := m.tree.Get(m.currentVersion, k)
		if err != nil {
			t.Fatalf("failed to get key %x: %v", []byte(keyStr), err)
		}
		if !bytes.Equal(value, expectedValue) {
			t.Fatalf("value mismatch for key %x: expected %v, got %v",
				[]byte(keyStr), expectedValue, value)
		}
	}
}

// Check runs the state machine test
func (m *treeStateMachine) Check(t *rapid.T) {
	t.Repeat(map[string]func(*rapid.T){
		"Put":    m.Put,
		"Get":    m.Get,
		"Delete": m.Delete,
	})
}

func TestTreeStateMachine(t *testing.T) {
	config := fuzz.GetTestConfigWithShort(t)
	config.ApplyToRapid(t)
	rapid.Check(t, func(t *rapid.T) {
		m := &treeStateMachine{}
		m.Init(t)
		m.Check(t)
	})
}

// TestTreeStateMachineVersionIsolation focuses on version isolation in state machine
func TestTreeStateMachineVersionIsolation(t *testing.T) {
	config := fuzz.GetTestConfigWithShort(t)
	config.ApplyToRapid(t)
	rapid.Check(t, func(t *rapid.T) {
		m := &treeStateMachine{}
		m.Init(t)

		// Run some operations
		numOps := rapid.IntRange(10, 50).Draw(t, "num_operations")
		for i := 0; i < numOps; i++ {
			// Mostly puts with some gets
			if rapid.Float64Range(0, 1).Draw(t, "op_choice") < 0.7 {
				m.Put(t)
			} else {
				m.Get(t)
			}
		}

		// Now verify version isolation across all versions
		for version, state := range m.versionStates {
			// Check that all keys in this version's state are visible
			for keyStr, expectedValue := range state {
				k := types.KeyHash([]byte(keyStr))
				value, err := m.tree.GetAtVersion(version, k)
				if err != nil {
					t.Fatalf("failed to get key at version %d: %v", version, err)
				}
				if !bytes.Equal(value, expectedValue) {
					t.Fatalf("version %d: expected %v, got %v", version, expectedValue, value)
				}
			}

			// Check that keys not in this version's state are not visible
			for keyStr := range m.expectedState {
				if _, exists := state[keyStr]; !exists {
					k := types.KeyHash([]byte(keyStr))
					value, err := m.tree.GetAtVersion(version, k)
					if err != nil {
						t.Fatalf("failed to get non-existent key at version %d: %v", version, err)
					}
					if value != nil {
						t.Fatalf("key %x should not exist at version %d but got value: %v",
							[]byte(keyStr), version, value)
					}
				}
			}
		}
	})
}

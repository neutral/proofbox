package types

import (
	"bytes"
	"testing"
)

func TestKeyFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "valid 32-byte key",
			input:   make([]byte, 32),
			wantErr: false,
		},
		{
			name:    "too short",
			input:   make([]byte, 31),
			wantErr: true,
		},
		{
			name:    "too long",
			input:   make([]byte, 33),
			wantErr: true,
		},
		{
			name:    "empty slice",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "nil slice",
			input:   nil,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key, err := KeyFromBytes(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !bytes.Equal(key[:], tc.input) {
					t.Errorf("key mismatch: got %x, want %x", key[:], tc.input)
				}
			}
		})
	}
}

func TestKeyHash(t *testing.T) {
	// Test that KeyHash produces consistent results
	data1 := []byte("test data")
	data2 := []byte("test data")
	data3 := []byte("different data")

	key1 := KeyHash(data1)
	key2 := KeyHash(data2)
	key3 := KeyHash(data3)

	if key1 != key2 {
		t.Errorf("KeyHash not consistent: same input produced different hashes")
	}

	if key1 == key3 {
		t.Errorf("KeyHash collision: different inputs produced same hash")
	}

	// Test empty input
	emptyKey := KeyHash([]byte{})
	if emptyKey.IsEmpty() {
		t.Errorf("KeyHash of empty slice should not produce empty key")
	}
}

func TestKeyString(t *testing.T) {
	key := Key{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}
	expected := "0123456789abcdef0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	result := key.String()
	if len(result) != 64 { // 32 bytes * 2 hex chars
		t.Errorf("Key.String() wrong length: got %d, want 64", len(result))
	}
	if result[:16] != expected[:16] {
		t.Errorf("Key.String() mismatch: got %s..., want %s...", result[:16], expected[:16])
	}
}

func TestKeyIsEmpty(t *testing.T) {
	emptyKey := Key{}
	if !emptyKey.IsEmpty() {
		t.Errorf("zero key should be empty")
	}

	nonEmptyKey := Key{0x01}
	if nonEmptyKey.IsEmpty() {
		t.Errorf("non-zero key should not be empty")
	}
}

func TestValidateKey(t *testing.T) {
	emptyKey := Key{}
	if err := ValidateKey(emptyKey); err == nil {
		t.Errorf("ValidateKey should reject empty key")
	}

	validKey := Key{0x01}
	if err := ValidateKey(validKey); err != nil {
		t.Errorf("ValidateKey should accept non-empty key: %v", err)
	}
}

func TestKeyExtractNibble(t *testing.T) {
	key := Key{0x12, 0x34, 0x56, 0x78} // First 4 bytes

	tests := []struct {
		depth    int
		expected Nibble
		wantErr  bool
	}{
		{0, 0x1, false},  // High nibble of first byte
		{1, 0x2, false},  // Low nibble of first byte
		{2, 0x3, false},  // High nibble of second byte
		{3, 0x4, false},  // Low nibble of second byte
		{4, 0x5, false},  // High nibble of third byte
		{5, 0x6, false},  // Low nibble of third byte
		{6, 0x7, false},  // High nibble of fourth byte
		{7, 0x8, false},  // Low nibble of fourth byte
		{63, 0x0, false}, // Last valid nibble
		{64, 0x0, true},  // Out of range
		{-1, 0x0, true},  // Negative
	}

	for _, tc := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			nibble, err := key.ExtractNibble(tc.depth)
			if tc.wantErr {
				if err == nil {
					t.Errorf("depth %d: expected error, got nil", tc.depth)
				}
			} else {
				if err != nil {
					t.Errorf("depth %d: unexpected error: %v", tc.depth, err)
				}
				if nibble != tc.expected {
					t.Errorf("depth %d: got nibble %x, want %x", tc.depth, nibble, tc.expected)
				}
			}
		})
	}
}

func TestKeyToNibblePath(t *testing.T) {
	key := Key{0xAB, 0xCD} // First 2 bytes set
	path := key.ToNibblePath()

	if path.Length != MaxTreeDepth {
		t.Errorf("ToNibblePath length: got %d, want %d", path.Length, MaxTreeDepth)
	}

	// Check first 4 nibbles
	expected := []Nibble{0xA, 0xB, 0xC, 0xD}
	for i, exp := range expected {
		if path.Nibbles[i] != exp {
			t.Errorf("nibble %d: got %x, want %x", i, path.Nibbles[i], exp)
		}
	}

	// Rest should be zeros
	for i := 4; i < int(path.Length); i++ {
		if path.Nibbles[i] != 0 {
			t.Errorf("nibble %d: got %x, want 0", i, path.Nibbles[i])
		}
	}
}

func TestValidateNibble(t *testing.T) {
	tests := []struct {
		nibble  Nibble
		wantErr bool
	}{
		{0, false},
		{15, false},
		{16, true},
		{255, true},
	}

	for _, tc := range tests {
		err := ValidateNibble(tc.nibble)
		if tc.wantErr && err == nil {
			t.Errorf("nibble %d: expected error, got nil", tc.nibble)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("nibble %d: unexpected error: %v", tc.nibble, err)
		}
	}
}

func TestNibblePathString(t *testing.T) {
	tests := []struct {
		path     NibblePath
		expected string
	}{
		{
			path:     NibblePath{Nibbles: []Nibble{}, Length: 0},
			expected: "<empty>",
		},
		{
			path:     NibblePath{Nibbles: []Nibble{0x1, 0x2, 0xA, 0xB}, Length: 4},
			expected: "12AB",
		},
		{
			path:     NibblePath{Nibbles: []Nibble{0xF, 0xE, 0xD, 0xC}, Length: 4},
			expected: "FEDC",
		},
	}

	for _, tc := range tests {
		result := tc.path.String()
		if result != tc.expected {
			t.Errorf("NibblePath.String(): got %s, want %s", result, tc.expected)
		}
	}
}

func TestNibblePathCompare(t *testing.T) {
	tests := []struct {
		name     string
		path1    NibblePath
		path2    NibblePath
		expected int
	}{
		{
			name:     "equal paths",
			path1:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			path2:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			expected: 0,
		},
		{
			name:     "first nibble differs",
			path1:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			path2:    NibblePath{Nibbles: []Nibble{2, 2, 3}, Length: 3},
			expected: -1,
		},
		{
			name:     "shorter path comes first",
			path1:    NibblePath{Nibbles: []Nibble{1, 2}, Length: 2},
			path2:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			expected: -1,
		},
		{
			name:     "longer path comes second",
			path1:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			path2:    NibblePath{Nibbles: []Nibble{1, 2}, Length: 2},
			expected: 1,
		},
		{
			name:     "empty paths",
			path1:    NibblePath{Nibbles: []Nibble{}, Length: 0},
			path2:    NibblePath{Nibbles: []Nibble{}, Length: 0},
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.path1.Compare(tc.path2)
			if result != tc.expected {
				t.Errorf("got %d, want %d", result, tc.expected)
			}
		})
	}
}

func BenchmarkKeyExtractNibble(b *testing.B) {
	key := KeyHash([]byte("benchmark key"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = key.ExtractNibble(i % MaxTreeDepth)
	}
}

func BenchmarkKeyToNibblePath(b *testing.B) {
	key := KeyHash([]byte("benchmark key"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = key.ToNibblePath()
	}
}


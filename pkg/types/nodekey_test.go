package types

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestRootNodeKey(t *testing.T) {
	version := Version(42)
	root := RootNodeKey(version)

	if root.Version != version {
		t.Errorf("version mismatch: got %d, want %d", root.Version, version)
	}
	if root.NibblePath.Length != 0 {
		t.Errorf("root should have empty path, got length %d", root.NibblePath.Length)
	}
	if !root.IsRoot() {
		t.Errorf("RootNodeKey should return a root node")
	}
}

func TestNodeKeyChild(t *testing.T) {
	parent := NodeKey{
		Version: 10,
		NibblePath: NibblePath{
			Nibbles: []Nibble{0x1, 0x2},
			Length:  2,
		},
	}

	nibble := Nibble(0xA)
	childVersion := Version(15)
	child := parent.Child(nibble, childVersion)

	// Check version
	if child.Version != childVersion {
		t.Errorf("child version: got %d, want %d", child.Version, childVersion)
	}

	// Check path length
	if child.NibblePath.Length != 3 {
		t.Errorf("child path length: got %d, want 3", child.NibblePath.Length)
	}

	// Check path contents
	expectedNibbles := []Nibble{0x1, 0x2, 0xA}
	for i, expected := range expectedNibbles {
		if child.NibblePath.Nibbles[i] != expected {
			t.Errorf("nibble %d: got %x, want %x", i, child.NibblePath.Nibbles[i], expected)
		}
	}

	// Ensure child is not root
	if child.IsRoot() {
		t.Errorf("child should not be root")
	}
}

func TestNodeKeyString(t *testing.T) {
	tests := []struct {
		key      NodeKey
		expected string
	}{
		{
			key:      RootNodeKey(100),
			expected: "v100:",
		},
		{
			key: NodeKey{
				Version: 1000,
				NibblePath: NibblePath{
					Nibbles: []Nibble{0x1, 0x2, 0xA, 0xB, 0xC},
					Length:  5,
				},
			},
			expected: "v1000:12abc",
		},
	}

	for _, tc := range tests {
		result := tc.key.String()
		if result != tc.expected {
			t.Errorf("NodeKey.String(): got %s, want %s", result, tc.expected)
		}
	}
}

func TestNodeKeyEncoding(t *testing.T) {
	tests := []struct {
		name string
		key  NodeKey
	}{
		{
			name: "root node",
			key:  RootNodeKey(100),
		},
		{
			name: "leaf with odd nibbles",
			key: NodeKey{
				Version: 1000,
				NibblePath: NibblePath{
					Nibbles: []Nibble{0x1, 0x2, 0xA, 0xB, 0xC},
					Length:  5,
				},
			},
		},
		{
			name: "deep node with even nibbles",
			key: NodeKey{
				Version: 50000,
				NibblePath: NibblePath{
					Nibbles: []Nibble{0xF, 0xF, 0xE, 0xE, 0xD, 0xD},
					Length:  6,
				},
			},
		},
		{
			name: "max depth",
			key: NodeKey{
				Version: MaxVersion,
				NibblePath: NibblePath{
					Nibbles: make([]Nibble, MaxTreeDepth),
					Length:  MaxTreeDepth,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			encoded := EncodeNodeKey(tc.key)

			// Verify minimum size
			if len(encoded) < 10 {
				t.Errorf("encoded size too small: %d", len(encoded))
			}

			// Decode
			decoded, err := DecodeNodeKey(encoded)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			// Compare
			if decoded.Version != tc.key.Version {
				t.Errorf("version mismatch: got %d, want %d", decoded.Version, tc.key.Version)
			}
			if decoded.NibblePath.Length != tc.key.NibblePath.Length {
				t.Errorf("path length mismatch: got %d, want %d", decoded.NibblePath.Length, tc.key.NibblePath.Length)
			}
			for i := uint16(0); i < tc.key.NibblePath.Length; i++ {
				if decoded.NibblePath.Nibbles[i] != tc.key.NibblePath.Nibbles[i] {
					t.Errorf("nibble %d mismatch: got %x, want %x", i, decoded.NibblePath.Nibbles[i], tc.key.NibblePath.Nibbles[i])
				}
			}
		})
	}
}

func TestNodeKeyEncodingFormat(t *testing.T) {
	// Test specific encoding format from the specification
	key := NodeKey{
		Version: 1000,
		NibblePath: NibblePath{
			Nibbles: []Nibble{0x1, 0x2, 0xA, 0xB, 0xC},
			Length:  5,
		},
	}

	encoded := EncodeNodeKey(key)

	// Check version encoding (big-endian)
	version := binary.BigEndian.Uint64(encoded[0:8])
	if version != 1000 {
		t.Errorf("version encoding: got %d, want 1000", version)
	}

	// Check length encoding
	length := binary.BigEndian.Uint16(encoded[8:10])
	if length != 5 {
		t.Errorf("length encoding: got %d, want 5", length)
	}

	// Check nibble packing
	expectedNibbleBytes := []byte{0x12, 0xAB, 0xC0}
	actualNibbleBytes := encoded[10:]
	if !bytes.Equal(actualNibbleBytes, expectedNibbleBytes) {
		t.Errorf("nibble packing: got %x, want %x", actualNibbleBytes, expectedNibbleBytes)
	}
}

func TestDecodeNodeKeyErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "too short",
			input:   make([]byte, 9),
			wantErr: true,
		},
		{
			name:    "invalid nibble count",
			input:   makeInvalidNodeKey(100, MaxTreeDepth+1),
			wantErr: true,
		},
		{
			name:    "truncated nibbles",
			input:   makeInvalidNodeKey(100, 10)[:12], // Missing nibble bytes
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeNodeKey(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNodeKeyStorageKey(t *testing.T) {
	key := NodeKey{
		Version: 42,
		NibblePath: NibblePath{
			Nibbles: []Nibble{0xA, 0xB},
			Length:  2,
		},
	}

	storageKey := key.StorageKey()

	// Should start with prefix
	if !bytes.HasPrefix(storageKey, []byte(NodeKeyPrefix)) {
		t.Errorf("storage key should start with prefix %q", NodeKeyPrefix)
	}

	// Should contain encoded node key
	encoded := EncodeNodeKey(key)
	if !bytes.HasSuffix(storageKey, encoded) {
		t.Errorf("storage key should end with encoded node key")
	}
}

func TestNodeKeyCompare(t *testing.T) {
	tests := []struct {
		name     string
		key1     NodeKey
		key2     NodeKey
		expected int
	}{
		{
			name:     "equal keys",
			key1:     RootNodeKey(100),
			key2:     RootNodeKey(100),
			expected: 0,
		},
		{
			name:     "different versions",
			key1:     RootNodeKey(100),
			key2:     RootNodeKey(200),
			expected: -1,
		},
		{
			name:     "same version, different paths",
			key1:     NodeKey{Version: 1, NibblePath: NibblePath{[]Nibble{0x1}, 1}},
			key2:     NodeKey{Version: 1, NibblePath: NibblePath{[]Nibble{0x2}, 1}},
			expected: -1,
		},
		{
			name:     "same version, different path lengths",
			key1:     NodeKey{Version: 1, NibblePath: NibblePath{[]Nibble{0x1}, 1}},
			key2:     NodeKey{Version: 1, NibblePath: NibblePath{[]Nibble{0x1, 0x2}, 2}},
			expected: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.key1.Compare(tc.key2)
			if result != tc.expected {
				t.Errorf("got %d, want %d", result, tc.expected)
			}

			// Test symmetry
			reverseResult := tc.key2.Compare(tc.key1)
			if tc.expected != 0 && reverseResult != -tc.expected {
				t.Errorf("comparison not symmetric: %d vs %d", result, reverseResult)
			}
		})
	}
}

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		version Version
		wantErr bool
	}{
		{0, false},
		{MaxVersion, false},
		{MaxVersion + 1, true}, // This is 0 due to overflow, but represents invalid input
	}

	for _, tc := range tests {
		err := ValidateVersion(tc.version)
		if tc.wantErr && err == nil {
			t.Errorf("version %d: expected error, got nil", tc.version)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("version %d: unexpected error: %v", tc.version, err)
		}
	}
}

func TestVersionEncoding(t *testing.T) {
	tests := []Version{0, 1, 100, 1000, MaxVersion}

	for _, version := range tests {
		encoded := EncodeVersion(version)
		decoded := DecodeVersion(encoded)
		if decoded != version {
			t.Errorf("version roundtrip failed: got %d, want %d", decoded, version)
		}
	}
}

// Helper function for tests
func makeInvalidNodeKey(version Version, nibbleCount uint16) []byte {
	buf := make([]byte, 10+(nibbleCount+1)/2)
	binary.BigEndian.PutUint64(buf[0:8], uint64(version))
	binary.BigEndian.PutUint16(buf[8:10], nibbleCount)
	return buf
}

// Fuzz test for NodeKey encoding/decoding
func FuzzNodeKeyEncoding(f *testing.F) {
	// Add seed corpus
	f.Add(uint64(0), []byte{})
	f.Add(uint64(100), []byte{0x12, 0x34})
	f.Add(uint64(MaxVersion), []byte{0xFF, 0xEE, 0xDD, 0xCC, 0xBB, 0xAA})

	f.Fuzz(func(t *testing.T, version uint64, nibbleData []byte) {
		// Limit nibble data to reasonable size
		if len(nibbleData) > MaxTreeDepth {
			nibbleData = nibbleData[:MaxTreeDepth]
		}

		// Convert to nibbles
		nibbles := make([]Nibble, len(nibbleData))
		for i, b := range nibbleData {
			nibbles[i] = Nibble(b & 0x0F) // Ensure valid nibble
		}

		original := NodeKey{
			Version: Version(version),
			NibblePath: NibblePath{
				Nibbles: nibbles,
				Length:  uint16(len(nibbles)),
			},
		}

		// Encode
		encoded := EncodeNodeKey(original)

		// Decode
		decoded, err := DecodeNodeKey(encoded)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}

		// Verify
		if decoded.Version != original.Version {
			t.Errorf("version mismatch: got %d, want %d", decoded.Version, original.Version)
		}
		if decoded.NibblePath.Length != original.NibblePath.Length {
			t.Errorf("length mismatch: got %d, want %d", decoded.NibblePath.Length, original.NibblePath.Length)
		}
		for i := uint16(0); i < original.NibblePath.Length; i++ {
			if decoded.NibblePath.Nibbles[i] != original.NibblePath.Nibbles[i] {
				t.Errorf("nibble %d mismatch: got %x, want %x", i, decoded.NibblePath.Nibbles[i], original.NibblePath.Nibbles[i])
			}
		}
	})
}

// Benchmarks
func BenchmarkNodeKeyEncoding(b *testing.B) {
	key := NodeKey{
		Version: 1000,
		NibblePath: NibblePath{
			Nibbles: []Nibble{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8},
			Length:  8,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeNodeKey(key)
	}
}

func BenchmarkNodeKeyDecoding(b *testing.B) {
	key := NodeKey{
		Version: 1000,
		NibblePath: NibblePath{
			Nibbles: []Nibble{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8},
			Length:  8,
		},
	}
	encoded := EncodeNodeKey(key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeNodeKey(encoded)
	}
}

func BenchmarkNodeKeyCompare(b *testing.B) {
	key1 := NodeKey{
		Version: 1000,
		NibblePath: NibblePath{
			Nibbles: []Nibble{0x1, 0x2, 0x3, 0x4},
			Length:  4,
		},
	}
	key2 := NodeKey{
		Version: 1000,
		NibblePath: NibblePath{
			Nibbles: []Nibble{0x1, 0x2, 0x3, 0x5},
			Length:  4,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = key1.Compare(key2)
	}
}

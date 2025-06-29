package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommonPrefixLength(t *testing.T) {
	testCases := []struct {
		name     string
		path1    NibblePath
		path2    NibblePath
		expected int
	}{
		{
			name:     "identical paths",
			path1:    NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4},
			path2:    NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4},
			expected: 4,
		},
		{
			name:     "different at first nibble",
			path1:    NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4},
			path2:    NibblePath{Nibbles: []Nibble{2, 2, 3, 4}, Length: 4},
			expected: 0,
		},
		{
			name:     "common prefix of 2",
			path1:    NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4},
			path2:    NibblePath{Nibbles: []Nibble{1, 2, 5, 6}, Length: 4},
			expected: 2,
		},
		{
			name:     "different lengths",
			path1:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			path2:    NibblePath{Nibbles: []Nibble{1, 2, 3, 4, 5}, Length: 5},
			expected: 3,
		},
		{
			name:     "empty paths",
			path1:    NibblePath{Nibbles: []Nibble{}, Length: 0},
			path2:    NibblePath{Nibbles: []Nibble{}, Length: 0},
			expected: 0,
		},
		{
			name:     "one empty path",
			path1:    NibblePath{Nibbles: []Nibble{}, Length: 0},
			path2:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.path1.CommonPrefixLength(tc.path2)
			assert.Equal(t, tc.expected, result)

			// Test commutativity
			result2 := tc.path2.CommonPrefixLength(tc.path1)
			assert.Equal(t, tc.expected, result2)
		})
	}
}

func TestGetNibble(t *testing.T) {
	path := NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4}

	// Valid indices
	for i := 0; i < 4; i++ {
		nibble, err := path.GetNibble(i)
		assert.NoError(t, err)
		assert.Equal(t, Nibble(i+1), nibble)
	}

	// Invalid indices
	_, err := path.GetNibble(-1)
	assert.Error(t, err)

	_, err = path.GetNibble(4)
	assert.Error(t, err)

	_, err = path.GetNibble(100)
	assert.Error(t, err)

	// Empty path
	emptyPath := NibblePath{Nibbles: []Nibble{}, Length: 0}
	_, err = emptyPath.GetNibble(0)
	assert.Error(t, err)
}

func TestPrefix(t *testing.T) {
	path := NibblePath{Nibbles: []Nibble{1, 2, 3, 4, 5}, Length: 5}

	// Normal prefix
	prefix := path.Prefix(3)
	assert.Equal(t, uint16(3), prefix.Length)
	assert.Equal(t, []Nibble{1, 2, 3}, prefix.Nibbles)

	// Full prefix
	prefix = path.Prefix(5)
	assert.Equal(t, uint16(5), prefix.Length)
	assert.Equal(t, []Nibble{1, 2, 3, 4, 5}, prefix.Nibbles)

	// Prefix longer than path
	prefix = path.Prefix(10)
	assert.Equal(t, path.Length, prefix.Length)
	assert.Equal(t, path.Nibbles, prefix.Nibbles)

	// Empty prefix
	prefix = path.Prefix(0)
	assert.Equal(t, uint16(0), prefix.Length)
	assert.Empty(t, prefix.Nibbles)

	// Negative prefix
	prefix = path.Prefix(-1)
	assert.Equal(t, uint16(0), prefix.Length)
	assert.Empty(t, prefix.Nibbles)
}

func TestEquals(t *testing.T) {
	testCases := []struct {
		name     string
		path1    NibblePath
		path2    NibblePath
		expected bool
	}{
		{
			name:     "identical paths",
			path1:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			path2:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			expected: true,
		},
		{
			name:     "different lengths",
			path1:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			path2:    NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4},
			expected: false,
		},
		{
			name:     "different content",
			path1:    NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			path2:    NibblePath{Nibbles: []Nibble{1, 2, 4}, Length: 3},
			expected: false,
		},
		{
			name:     "empty paths",
			path1:    NibblePath{Nibbles: []Nibble{}, Length: 0},
			path2:    NibblePath{Nibbles: []Nibble{}, Length: 0},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.path1.Equals(tc.path2)
			assert.Equal(t, tc.expected, result)

			// Test reflexivity
			assert.True(t, tc.path1.Equals(tc.path1))
			assert.True(t, tc.path2.Equals(tc.path2))

			// Test symmetry
			assert.Equal(t, result, tc.path2.Equals(tc.path1))
		})
	}
}

func TestAppend(t *testing.T) {
	// Test appending to non-empty path
	path := NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3}
	newPath := path.Append(4)

	assert.Equal(t, uint16(4), newPath.Length)
	assert.Equal(t, []Nibble{1, 2, 3, 4}, newPath.Nibbles)

	// Original path should be unchanged
	assert.Equal(t, uint16(3), path.Length)
	assert.Equal(t, []Nibble{1, 2, 3}, path.Nibbles)

	// Test appending to empty path
	emptyPath := NibblePath{Nibbles: []Nibble{}, Length: 0}
	newPath2 := emptyPath.Append(5)

	assert.Equal(t, uint16(1), newPath2.Length)
	assert.Equal(t, []Nibble{5}, newPath2.Nibbles)
}

func TestIsPrefix(t *testing.T) {
	testCases := []struct {
		name     string
		prefix   NibblePath
		path     NibblePath
		expected bool
	}{
		{
			name:     "valid prefix",
			prefix:   NibblePath{Nibbles: []Nibble{1, 2}, Length: 2},
			path:     NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4},
			expected: true,
		},
		{
			name:     "not a prefix",
			prefix:   NibblePath{Nibbles: []Nibble{1, 3}, Length: 2},
			path:     NibblePath{Nibbles: []Nibble{1, 2, 3, 4}, Length: 4},
			expected: false,
		},
		{
			name:     "prefix longer than path",
			prefix:   NibblePath{Nibbles: []Nibble{1, 2, 3, 4, 5}, Length: 5},
			path:     NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			expected: false,
		},
		{
			name:     "equal paths",
			prefix:   NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			path:     NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			expected: true,
		},
		{
			name:     "empty prefix",
			prefix:   NibblePath{Nibbles: []Nibble{}, Length: 0},
			path:     NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3},
			expected: true,
		},
		{
			name:     "both empty",
			prefix:   NibblePath{Nibbles: []Nibble{}, Length: 0},
			path:     NibblePath{Nibbles: []Nibble{}, Length: 0},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.prefix.IsPrefix(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSkip(t *testing.T) {
	path := NibblePath{Nibbles: []Nibble{1, 2, 3, 4, 5}, Length: 5}

	// Skip first 2 nibbles
	skipped := path.Skip(2)
	assert.Equal(t, uint16(3), skipped.Length)
	assert.Equal(t, []Nibble{3, 4, 5}, skipped.Nibbles)

	// Skip all nibbles
	skipped = path.Skip(5)
	assert.Equal(t, uint16(0), skipped.Length)
	assert.Empty(t, skipped.Nibbles)

	// Skip more than available
	skipped = path.Skip(10)
	assert.Equal(t, uint16(0), skipped.Length)
	assert.Empty(t, skipped.Nibbles)

	// Skip zero
	skipped = path.Skip(0)
	assert.Equal(t, path.Length, skipped.Length)
	assert.Equal(t, path.Nibbles, skipped.Nibbles)

	// Skip negative
	skipped = path.Skip(-5)
	assert.Equal(t, path.Length, skipped.Length)
	assert.Equal(t, path.Nibbles, skipped.Nibbles)
}

func TestNibblePathConversion(t *testing.T) {
	// Test byte to nibble conversion
	testCases := []struct {
		name     string
		data     []byte
		expected []Nibble
	}{
		{
			name:     "single byte",
			data:     []byte{0x12},
			expected: []Nibble{0x1, 0x2},
		},
		{
			name:     "multiple bytes",
			data:     []byte{0x12, 0x34, 0x56},
			expected: []Nibble{0x1, 0x2, 0x3, 0x4, 0x5, 0x6},
		},
		{
			name:     "all zeros",
			data:     []byte{0x00, 0x00},
			expected: []Nibble{0x0, 0x0, 0x0, 0x0},
		},
		{
			name:     "all ones",
			data:     []byte{0xFF, 0xFF},
			expected: []Nibble{0xF, 0xF, 0xF, 0xF},
		},
		{
			name:     "empty",
			data:     []byte{},
			expected: []Nibble{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := NewNibblePath(tc.data)
			assert.Equal(t, uint16(len(tc.expected)), path.Length)
			assert.Equal(t, tc.expected, path.Nibbles)

			// Test round-trip conversion for even-length paths
			if path.Length%2 == 0 {
				bytes, err := path.ToBytes()
				assert.NoError(t, err)
				assert.Equal(t, tc.data, bytes)
			}
		})
	}

	// Test nibble to byte conversion errors
	oddPath := NibblePath{Nibbles: []Nibble{1, 2, 3}, Length: 3}
	_, err := oddPath.ToBytes()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "odd-length")
}

func TestNewNibblePathFromKey(t *testing.T) {
	// Test conversion from Key type
	key := Key{}
	for i := range key {
		key[i] = byte(i)
	}

	path := key.ToNibblePath()
	assert.Equal(t, uint16(MaxTreeDepth), path.Length)

	// Verify first few nibbles
	assert.Equal(t, Nibble(0x0), path.Nibbles[0])
	assert.Equal(t, Nibble(0x0), path.Nibbles[1])
	assert.Equal(t, Nibble(0x0), path.Nibbles[2])
	assert.Equal(t, Nibble(0x1), path.Nibbles[3])
}

func BenchmarkCommonPrefixLength(b *testing.B) {
	// Create two paths that differ at position 32
	key1 := Key{}
	key2 := Key{}
	for i := range key1 {
		key1[i] = byte(i)
		key2[i] = byte(i)
	}
	key2[16] = 0xFF // Differ at byte 16 (nibble 32)

	path1 := key1.ToNibblePath()
	path2 := key2.ToNibblePath()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = path1.CommonPrefixLength(path2)
	}
}

func BenchmarkNibblePathConversion(b *testing.B) {
	key := Key{}
	for i := range key {
		key[i] = byte(i)
	}

	b.Run("ToNibblePath", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewNibblePath(key[:])
		}
	})

	b.Run("ToBytes", func(b *testing.B) {
		path := key.ToNibblePath()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = path.ToBytes()
		}
	})
}

func BenchmarkAppend(b *testing.B) {
	path := NibblePath{Nibbles: make([]Nibble, 32), Length: 32}
	nibble := Nibble(5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = path.Append(nibble)
	}
}

func BenchmarkPrefix(b *testing.B) {
	path := Key{}.ToNibblePath()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = path.Prefix(32)
	}
}

func BenchmarkSkip(b *testing.B) {
	path := Key{}.ToNibblePath()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = path.Skip(16)
	}
}
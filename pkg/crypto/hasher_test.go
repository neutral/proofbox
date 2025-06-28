package crypto

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/neutral/proofbox/pkg/types"
)

func TestDefaultSHA256_Hash(t *testing.T) {
	hasher := DefaultSHA256{}

	// Test empty input
	emptyHash := hasher.Hash(nil)
	expectedEmpty := sha256.Sum256(nil)
	if !bytes.Equal(emptyHash[:], expectedEmpty[:]) {
		t.Errorf("Hash(nil) = %x, want %x", emptyHash, expectedEmpty)
	}

	// Test known input
	input := []byte("hello world")
	hash := hasher.Hash(input)
	expected := sha256.Sum256(input)
	if !bytes.Equal(hash[:], expected[:]) {
		t.Errorf("Hash(%q) = %x, want %x", input, hash, expected)
	}
}

func TestDefaultSHA256_HashConcat(t *testing.T) {
	hasher := DefaultSHA256{}

	// Test concatenation
	parts := [][]byte{
		[]byte("hello"),
		[]byte(" "),
		[]byte("world"),
	}

	hash := hasher.HashConcat(parts...)

	// Compare with manual concatenation
	combined := bytes.Join(parts, nil)
	expected := sha256.Sum256(combined)

	if !bytes.Equal(hash[:], expected[:]) {
		t.Errorf("HashConcat() = %x, want %x", hash, expected)
	}

	// Test empty parts
	emptyHash := hasher.HashConcat()
	expectedEmpty := sha256.Sum256(nil)
	if !bytes.Equal(emptyHash[:], expectedEmpty[:]) {
		t.Error("HashConcat with no parts should equal hash of empty data")
	}
}

func TestDefaultSHA256_EmptyHash(t *testing.T) {
	hasher := DefaultSHA256{}

	emptyHash := hasher.EmptyHash()
	expected := sha256.Sum256(nil)

	if !bytes.Equal(emptyHash[:], expected[:]) {
		t.Errorf("EmptyHash() = %x, want %x", emptyHash, expected)
	}
}

func TestDefaultHasher(t *testing.T) {
	// Verify DefaultHasher is properly initialized
	if DefaultHasher == nil {
		t.Fatal("DefaultHasher should not be nil")
	}

	// Test it implements the interface
	var _ Hasher = DefaultHasher

	// Test basic functionality
	hash := DefaultHasher.Hash([]byte("test"))
	expected := sha256.Sum256([]byte("test"))
	if !bytes.Equal(hash[:], expected[:]) {
		t.Error("DefaultHasher should use SHA-256")
	}
}

func TestDefaultDigest(t *testing.T) {
	// DefaultDigest should be SHA-256 of empty string
	expected := sha256.Sum256(nil)
	if !bytes.Equal(DefaultDigest[:], expected[:]) {
		t.Errorf("DefaultDigest = %x, want %x", DefaultDigest, expected)
	}

	// Verify it matches the documented value
	expectedHex := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hex.EncodeToString(DefaultDigest[:]) != expectedHex {
		t.Errorf("DefaultDigest hex = %s, want %s", hex.EncodeToString(DefaultDigest[:]), expectedHex)
	}
}

func TestEmptyTreeHash(t *testing.T) {
	// Verify EmptyTreeHash matches the specification
	expectedHex := "5ba93c9db0cff93f52b521d7814c7fa08abe86135f746b4668b013bcd1c48e9b"
	if hex.EncodeToString(EmptyTreeHash[:]) != expectedHex {
		t.Errorf("EmptyTreeHash hex = %s, want %s", hex.EncodeToString(EmptyTreeHash[:]), expectedHex)
	}

	// Verify EmptyTreeHash is different from DefaultDigest
	if bytes.Equal(EmptyTreeHash[:], DefaultDigest[:]) {
		t.Error("EmptyTreeHash should be different from DefaultDigest")
	}
}

func TestHasherInterface(t *testing.T) {
	// Test that we can use different hasher implementations
	var hasher Hasher = DefaultSHA256{}

	// Basic operations should work
	data := []byte("test data")
	hash1 := hasher.Hash(data)
	hash2 := DefaultHasher.Hash(data)

	if !hash1.Equal(hash2) {
		t.Error("Different hasher instances should produce same results")
	}
}

func TestHashDeterminism(t *testing.T) {
	hasher := DefaultSHA256{}
	data := []byte("deterministic test")

	// Hash same data multiple times
	hash1 := hasher.Hash(data)
	hash2 := hasher.Hash(data)
	hash3 := hasher.Hash(data)

	if !hash1.Equal(hash2) || !hash2.Equal(hash3) {
		t.Error("Hash should be deterministic")
	}

	// Test concatenation determinism
	parts := [][]byte{[]byte("part1"), []byte("part2")}
	concat1 := hasher.HashConcat(parts...)
	concat2 := hasher.HashConcat(parts...)

	if !concat1.Equal(concat2) {
		t.Error("HashConcat should be deterministic")
	}
}

func TestHashAvalanche(t *testing.T) {
	hasher := DefaultSHA256{}

	// Small change in input should cause large change in output
	input1 := []byte("hello world")
	input2 := []byte("hello World") // One bit changed

	hash1 := hasher.Hash(input1)
	hash2 := hasher.Hash(input2)

	// Count differing bits
	differences := 0
	for i := 0; i < types.HashSize; i++ {
		xor := hash1[i] ^ hash2[i]
		for xor != 0 {
			differences += int(xor & 1)
			xor >>= 1
		}
	}

	// Should have significant bit differences (avalanche effect)
	// For good hash functions, expect ~50% of bits to differ
	minDifferences := types.HashSize * 8 / 4 // At least 25% of bits
	if differences < minDifferences {
		t.Errorf("Insufficient avalanche effect: only %d bits differ (min %d)", differences, minDifferences)
	}
}

// Benchmarks

func BenchmarkHash(b *testing.B) {
	hasher := DefaultSHA256{}
	data := make([]byte, 1024) // 1KB of data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.Hash(data)
	}
}

func BenchmarkHashConcat(b *testing.B) {
	hasher := DefaultSHA256{}
	parts := [][]byte{
		make([]byte, 32),
		make([]byte, 32),
		make([]byte, 32),
		make([]byte, 32),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.HashConcat(parts...)
	}
}

func BenchmarkEmptyHash(b *testing.B) {
	hasher := DefaultSHA256{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.EmptyHash()
	}
}

func BenchmarkHashSmall(b *testing.B) {
	hasher := DefaultSHA256{}
	data := make([]byte, 32) // 32 bytes

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.Hash(data)
	}
}

func BenchmarkHashLarge(b *testing.B) {
	hasher := DefaultSHA256{}
	data := make([]byte, 1<<20) // 1MB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.Hash(data)
	}
}

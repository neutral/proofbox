package types

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestHash_Basic(t *testing.T) {
	// Test EmptyHash
	empty := EmptyHash()
	if !empty.IsEmpty() {
		t.Error("EmptyHash should return zero hash")
	}

	// Test hash creation
	data := make([]byte, HashSize)
	copy(data, "hello world")
	var h Hash
	copy(h[:], data)

	// Test Bytes()
	if !bytes.Equal(h.Bytes(), h[:]) {
		t.Error("Bytes() should return underlying array")
	}

	// Test String()
	expected := hex.EncodeToString(h[:])
	if h.String() != expected {
		t.Errorf("String() = %s, want %s", h.String(), expected)
	}
}

func TestHash_Equal(t *testing.T) {
	h1 := Hash{1, 2, 3}
	h2 := Hash{1, 2, 3}
	h3 := Hash{4, 5, 6}

	if !h1.Equal(h2) {
		t.Error("Equal hashes should be equal")
	}

	if h1.Equal(h3) {
		t.Error("Different hashes should not be equal")
	}
}

func TestHashFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "valid 32 bytes",
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
			name:    "nil",
			input:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := HashFromBytes(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashFromBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !bytes.Equal(h[:], tt.input) {
				t.Error("HashFromBytes should copy input bytes")
			}
		})
	}
}

func TestHashFromHex(t *testing.T) {
	// Test valid hex
	hexStr := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	h, err := HashFromHex(hexStr)
	if err != nil {
		t.Fatalf("HashFromHex failed: %v", err)
	}

	if h.String() != hexStr {
		t.Errorf("HashFromHex round trip failed: got %s, want %s", h.String(), hexStr)
	}

	// Test invalid hex
	_, err = HashFromHex("invalid hex")
	if err == nil {
		t.Error("HashFromHex should fail on invalid hex")
	}

	// Test wrong length
	_, err = HashFromHex("deadbeef")
	if err == nil {
		t.Error("HashFromHex should fail on wrong length")
	}
}

func TestHash_Deterministic(t *testing.T) {
	// Same data should produce same hash representation
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

	h1, err := HashFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := HashFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	if !h1.Equal(h2) {
		t.Error("Same data should produce equal hashes")
	}

	if h1.String() != h2.String() {
		t.Error("Same data should produce same string representation")
	}
}

func BenchmarkHash_String(b *testing.B) {
	h := Hash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.String()
	}
}

func BenchmarkHash_Equal(b *testing.B) {
	h1 := Hash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	h2 := h1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h1.Equal(h2)
	}
}

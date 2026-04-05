package delta

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestXOR_Roundtrip(t *testing.T) {
	// Generate random pattern and original data
	n := 128
	original := make([]byte, n)
	pattern := make([]byte, n)

	rand.Read(original)
	rand.Read(pattern)

	// original ^ pattern = delta
	deltaBytes := XOR(original, pattern)

	// pattern ^ delta = reconstructed
	reconstructed := Apply(pattern, deltaBytes)

	if !bytes.Equal(original, reconstructed) {
		t.Fatal("lossy reconstruction: reconstructed != original")
	}
}

func TestXOR_Identical(t *testing.T) {
	n := 64
	data := make([]byte, n)
	rand.Read(data)

	deltaBytes := XOR(data, data)

	// The XOR of identical slices should be all zeros
	for i, b := range deltaBytes {
		if b != 0 {
			t.Fatalf("delta[%d] expected 0, got %X", i, b)
		}
	}

	// Reconstruct
	reconstructed := Apply(data, deltaBytes)
	if !bytes.Equal(data, reconstructed) {
		t.Fatal("lossy reconstruction from zero-delta")
	}
}

func TestXOR_DifferentLengths(t *testing.T) {
	// XOR and Apply process exactly the length of 'orig'
	orig := []byte{1, 2, 3, 4}
	pat := []byte{0xFF, 0xFF}

	// Will process 4 bytes, borrowing the pattern for the first 2, and zeroes for the rest
	deltaBytes := XOR(orig, pat)
	if len(deltaBytes) != 4 {
		t.Fatalf("expected delta length 4, got %d", len(deltaBytes))
	}

	// Reconstruct
	reconstructed := Apply(pat, deltaBytes)
	if len(reconstructed) != 4 {
		t.Fatalf("expected reconstructed length 4, got %d", len(reconstructed))
	}
	if !bytes.Equal(reconstructed, orig) {
		t.Fatal("reconstruction of variable length slice failed")
	}
}

func TestCompressPool_Roundtrip(t *testing.T) {
	// Create a compressible pool (many zeros, typical of delta pool)
	pool := make([]byte, 1000)
	for i := 0; i < len(pool); i += 10 {
		pool[i] = 0xFF // Add some non-zero data
	}

	compressed, err := CompressPool(pool)
	if err != nil {
		t.Fatalf("compression failed: %v", err)
	}

	if len(compressed) >= len(pool) {
		t.Logf("note: compression didn't shrink data (pool=%d, comp=%d). Expected for very small/random data, but not here.", len(pool), len(compressed))
	}

	decompressed, err := DecompressPool(compressed)
	if err != nil {
		t.Fatalf("decompression failed: %v", err)
	}

	if !bytes.Equal(pool, decompressed) {
		t.Fatal("decompressed data does not match original pool")
	}
}

func TestCompressPool_RandomData(t *testing.T) {
	// Random data doesn't compress well, but it should still roundtrip safely
	pool := make([]byte, 500)
	rand.Read(pool)

	compressed, err := CompressPool(pool)
	if err != nil {
		t.Fatal(err)
	}

	decompressed, err := DecompressPool(compressed)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(pool, decompressed) {
		t.Fatal("random data roundtrip failed")
	}
}

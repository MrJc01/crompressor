package search

import (
	"bytes"
	"math/rand"
	"testing"
)

// Mock Codebook Reader using a simple wrapper
// For unit tests, we'll avoid the full mmap codebook and just
// test the linear logic using raw byte slices. Wait, the linear searcher
// depends directly on *codebook.Reader. We should use the real one
// or refactor to interfaces. Let's use the real codebook by creating a temp one.

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"

	"github.com/MrJc01/crompressor/internal/codebook"
)

func createTestCodebook(t *testing.T, patterns [][]byte) string {
	t.Helper()

	if len(patterns) == 0 {
		t.Fatal("no patterns provided")
	}
	cwSize := uint16(len(patterns[0]))

	dir := t.TempDir()
	path := filepath.Join(dir, "search_test.cromdb")

	var data bytes.Buffer
	for _, p := range patterns {
		if len(p) != int(cwSize) {
			t.Fatalf("all patterns must have the same size, expected %d got %d", cwSize, len(p))
		}
		data.Write(p)
	}
	codewordData := data.Bytes()
	buildHash := sha256.Sum256(codewordData)

	header := make([]byte, codebook.HeaderSize)
	copy(header[0:codebook.MagicSize], codebook.MagicString)
	binary.LittleEndian.PutUint16(header[6:8], codebook.Version1)
	binary.LittleEndian.PutUint16(header[8:10], cwSize)
	binary.LittleEndian.PutUint64(header[10:18], uint64(len(patterns)))
	binary.LittleEndian.PutUint64(header[18:26], codebook.HeaderSize)
	copy(header[26:58], buildHash[:])

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Write(header)
	f.Write(codewordData)
	f.Close()

	return path
}

func TestHammingDistance(t *testing.T) {
	tests := []struct {
		a, b []byte
		dist int
	}{
		{[]byte{1, 2, 3}, []byte{1, 2, 3}, 0},
		{[]byte{1, 2, 3}, []byte{1, 2, 4}, 3},  // 3 (00000011) vs 4 (00000100) = 3 bits
		{[]byte{1, 2, 3}, []byte{4, 5, 6}, 7},  // 1^4=5(2), 2^5=7(3), 3^6=5(2) = 7 bits
		{[]byte{1, 2}, []byte{1, 2, 3, 4}, 16}, // diff lengths -> missing 2 bytes = 16 bits
		{[]byte{}, []byte{1}, 8},               // diff length 1 byte = 8 bits
	}

	for _, tt := range tests {
		got := hammingDistance(tt.a, tt.b)
		if got != tt.dist {
			t.Errorf("hammingDistance(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.dist)
		}
	}
}

func TestLinearSearcher_FindBestMatch(t *testing.T) {
	patterns := [][]byte{
		{0, 0, 0, 0},     // ID 0
		{0xFF, 0, 0, 0},  // ID 1
		{1, 2, 3, 4},     // ID 2
		{10, 20, 30, 40}, // ID 3
	}
	path := createTestCodebook(t, patterns)

	cb, err := codebook.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer cb.Close()

	searcher := NewLinearSearcher(cb)

	// Perfect match test
	chunk := []byte{1, 2, 3, 4}
	res, err := searcher.FindBestMatch(chunk)
	if err != nil {
		t.Fatal(err)
	}
	if res.CodebookID != 2 {
		t.Errorf("expected ID 2, got %d", res.CodebookID)
	}
	if res.Distance != 0 {
		t.Errorf("expected distance 0, got %d", res.Distance)
	}

	// Partial match test
	chunk = []byte{1, 2, 3, 5} // 1 byte off from ID 2
	res, err = searcher.FindBestMatch(chunk)
	if err != nil {
		t.Fatal(err)
	}
	if res.CodebookID != 2 {
		t.Errorf("expected ID 2, got %d", res.CodebookID)
	}
	if res.Distance != 1 {
		t.Errorf("expected distance 1, got %d", res.Distance)
	}

	// Completely different chunk matching closest neighbor
	chunk = []byte{0xFE, 0, 0, 0} // Closest to ID 1
	res, err = searcher.FindBestMatch(chunk)
	if err != nil {
		t.Fatal(err)
	}
	if res.CodebookID != 1 {
		t.Errorf("expected ID 1, got %d", res.CodebookID)
	}
	if res.Distance != 1 {
		t.Errorf("expected distance 1, got %d", res.Distance)
	}
}

func BenchmarkLinearSearcher(b *testing.B) {
	// 4096 patterns of 128 bytes each
	numPatterns := 4096
	cwSize := 128
	patterns := make([][]byte, numPatterns)

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < numPatterns; i++ {
		patterns[i] = make([]byte, cwSize)
		rng.Read(patterns[i])
	}

	dir := b.TempDir()
	path := filepath.Join(dir, "bench.cromdb")
	codegen(path, patterns, cwSize)

	cb, _ := codebook.Open(path)
	defer cb.Close()

	searcher := NewLinearSearcher(cb)

	query := make([]byte, cwSize)
	rng.Read(query)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searcher.FindBestMatch(query)
	}
}

// Helper to write raw codebook for benchmark without going through temp test func
func codegen(path string, patterns [][]byte, cwSize int) string {
	codewordData := make([]byte, 0, len(patterns)*cwSize)
	for _, p := range patterns {
		codewordData = append(codewordData, p...)
	}
	buildHash := sha256.Sum256(codewordData)

	header := make([]byte, codebook.HeaderSize)
	copy(header[0:6], codebook.MagicString)
	binary.LittleEndian.PutUint16(header[6:8], codebook.Version1)
	binary.LittleEndian.PutUint16(header[8:10], uint16(cwSize))
	binary.LittleEndian.PutUint64(header[10:18], uint64(len(patterns)))
	binary.LittleEndian.PutUint64(header[18:26], codebook.HeaderSize)
	copy(header[26:58], buildHash[:])

	f, _ := os.Create(path)
	f.Write(header)
	f.Write(codewordData)
	f.Close()

	return path
}

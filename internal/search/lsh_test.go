package search

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/MrJc01/crompressor/internal/codebook"
)

func createLSHCodebook(t *testing.T, patterns [][]byte) string {
	t.Helper()
	cwSize := uint16(len(patterns[0]))
	dir := t.TempDir()
	path := filepath.Join(dir, "search_lsh.cromdb")

	codewordData := make([]byte, 0, int(cwSize)*len(patterns))
	for _, p := range patterns {
		codewordData = append(codewordData, p...)
	}
	buildHash := sha256.Sum256(codewordData)

	header := make([]byte, codebook.HeaderSize)
	copy(header[0:codebook.MagicSize], codebook.MagicString)
	binary.LittleEndian.PutUint16(header[6:8], codebook.Version1)
	binary.LittleEndian.PutUint16(header[8:10], cwSize)
	binary.LittleEndian.PutUint64(header[10:18], uint64(len(patterns)))
	binary.LittleEndian.PutUint64(header[18:26], codebook.HeaderSize)
	copy(header[26:58], buildHash[:])

	f, _ := os.Create(path)
	f.Write(header)
	f.Write(codewordData)
	f.Close()

	return path
}

func TestLSHSearcher_FindBestMatch(t *testing.T) {
	patterns := [][]byte{
		{0, 0, 0, 0},     // ID 0
		{0xFF, 0, 0, 0},  // ID 1
		{1, 2, 3, 4},     // ID 2
		{10, 20, 30, 40}, // ID 3
		{1, 2, 99, 99},   // ID 4 (same bucket as ID 2)
	}

	path := createLSHCodebook(t, patterns)
	cb, _ := codebook.Open(path)
	defer cb.Close()

	lsh := NewLSHSearcher(cb)

	// Bucket test: chunk{1, 2, ...} computes hash 0x0201
	// It should find ID 2 since it's a perfect match
	chunk := []byte{1, 2, 3, 4}
	res, _ := lsh.FindBestMatch(chunk)
	if res.CodebookID != 2 {
		t.Errorf("expected ID 2, got %d", res.CodebookID)
	}

	// Fallback test: bucket empty -> linear scan
	chunk = []byte{0, 0, 10, 10}
	res, _ = lsh.FindBestMatch(chunk)
	if res.CodebookID != 0 {
		t.Errorf("fallback linear expected ID 0, got %d", res.CodebookID)
	}
}

func BenchmarkLSHSearcher(b *testing.B) {
	numPatterns := 4096
	cwSize := 128
	patterns := make([][]byte, numPatterns)

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < numPatterns; i++ {
		patterns[i] = make([]byte, cwSize)
		rng.Read(patterns[i])
	}

	dir := b.TempDir()
	path := filepath.Join(dir, "bench_lsh.cromdb")

	// inline codegen
	codewordData := make([]byte, 0, int(cwSize)*numPatterns)
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

	cb, _ := codebook.Open(path)
	defer cb.Close()

	b.ResetTimer()
	b.StopTimer()
	lsh := NewLSHSearcher(cb)
	b.StartTimer()

	query := make([]byte, cwSize)
	copy(query, patterns[314])

	for i := 0; i < b.N; i++ {
		lsh.FindBestMatch(query)
	}
}

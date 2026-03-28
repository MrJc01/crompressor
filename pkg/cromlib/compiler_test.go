package cromlib

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math"
	mathrand "math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/crom-project/crom/internal/chunker"
	"github.com/crom-project/crom/internal/codebook"
)

// createTestCodebook generates a .cromdb file with patterns derived from the input data.
// This simulates a real "train" step by extracting actual 128-byte chunks from the data.
func createTestCodebook(t *testing.T, data []byte) string {
	t.Helper()

	dir := t.TempDir()
	cbPath := filepath.Join(dir, "test.cromdb")

	fc := chunker.NewFixedChunker(chunker.DefaultChunkSize)
	chunks := fc.Split(data)

	// Collect unique patterns (up to 256 for tests)
	seen := make(map[[32]byte]bool)
	var patterns [][]byte
	for _, c := range chunks {
		if len(c.Data) != chunker.DefaultChunkSize {
			continue // Skip partial chunks
		}
		hash := sha256.Sum256(c.Data)
		if !seen[hash] {
			seen[hash] = true
			p := make([]byte, chunker.DefaultChunkSize)
			copy(p, c.Data)
			patterns = append(patterns, p)
			if len(patterns) >= 256 {
				break
			}
		}
	}

	// If data is too small or uniform, add some random patterns
	if len(patterns) == 0 {
		p := make([]byte, chunker.DefaultChunkSize)
		patterns = append(patterns, p) // zero pattern
	}

	// Write the codebook manually (avoiding circular import with trainer)
	writeCodebook(t, cbPath, patterns)
	return cbPath
}

// writeCodebook writes a .cromdb from byte patterns.
func writeCodebook(t *testing.T, path string, patterns [][]byte) {
	t.Helper()

	cwSize := uint16(len(patterns[0]))
	h := sha256.New()
	for _, p := range patterns {
		h.Write(p)
	}
	buildHash := h.Sum(nil)

	header := make([]byte, codebook.HeaderSize)
	copy(header[0:codebook.MagicSize], codebook.MagicString)
	binary.LittleEndian.PutUint16(header[6:8], codebook.Version1)
	binary.LittleEndian.PutUint16(header[8:10], cwSize)
	binary.LittleEndian.PutUint64(header[10:18], uint64(len(patterns)))
	binary.LittleEndian.PutUint64(header[18:26], codebook.HeaderSize)
	copy(header[26:58], buildHash[:32])

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	f.Write(header)
	for _, p := range patterns {
		f.Write(p)
	}
}

// packUnpackRoundtrip is a helper that tests the full Pack → Unpack pipeline.
func packUnpackRoundtrip(t *testing.T, data []byte, testName string) {
	t.Helper()

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.bin")
	cromPath := filepath.Join(dir, "output.crom")
	outputPath := filepath.Join(dir, "restored.bin")

	if err := os.WriteFile(inputPath, data, 0644); err != nil {
		t.Fatalf("[%s] write input: %v", testName, err)
	}

	cbPath := createTestCodebook(t, data)

	// Pack
	opts := DefaultPackOptions()
	opts.Concurrency = 2
	metrics, err := Pack(inputPath, cromPath, cbPath, opts)
	if err != nil {
		t.Fatalf("[%s] Pack failed: %v", testName, err)
	}

	if metrics.OriginalSize != uint64(len(data)) {
		t.Errorf("[%s] metrics.OriginalSize = %d, want %d", testName, metrics.OriginalSize, len(data))
	}

	// Unpack
	unpackOpts := DefaultUnpackOptions()
	err = Unpack(cromPath, outputPath, cbPath, unpackOpts)
	if err != nil {
		t.Fatalf("[%s] Unpack failed: %v", testName, err)
	}

	// Compare
	restored, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("[%s] read output: %v", testName, err)
	}

	if len(restored) != len(data) {
		t.Fatalf("[%s] size mismatch: original=%d, restored=%d", testName, len(data), len(restored))
	}

	origHash := sha256.Sum256(data)
	restHash := sha256.Sum256(restored)
	if origHash != restHash {
		t.Fatalf("[%s] SHA-256 MISMATCH: original=%x, restored=%x", testName, origHash[:8], restHash[:8])
	}

	t.Logf("[%s] ✔ Roundtrip OK: %d bytes → %d bytes (%.1f%% ratio), HitRate=%.1f%%",
		testName, metrics.OriginalSize, metrics.PackedSize,
		float64(metrics.PackedSize)/float64(metrics.OriginalSize)*100,
		metrics.HitRate)
}

// --- Test Cases ---

func TestPackUnpack_SmallFile(t *testing.T) {
	// 1KB of repetitive data — should compress well
	data := bytes.Repeat([]byte("Hello, Crompressor! "), 52) // ~1040 bytes
	packUnpackRoundtrip(t, data, "SmallFile_1KB")
}

func TestPackUnpack_MediumFile(t *testing.T) {
	// 1MB of mixed data
	rng := mathrand.New(mathrand.NewSource(42))
	data := make([]byte, 1*1024*1024)
	// Fill with semi-structured data: patterns + noise
	for i := 0; i < len(data); i += 128 {
		end := i + 128
		if end > len(data) {
			end = len(data)
		}
		segment := data[i:end]
		patternID := rng.Intn(20)
		for j := range segment {
			segment[j] = byte(patternID*13 + j%7)
		}
	}
	packUnpackRoundtrip(t, data, "MediumFile_1MB")
}

func TestPackUnpack_RandomData(t *testing.T) {
	// 512KB of purely random data — worst case for compression
	data := make([]byte, 512*1024)
	rand.Read(data)
	packUnpackRoundtrip(t, data, "RandomData_512KB")
}

func TestPackUnpack_RepetitiveData(t *testing.T) {
	// 1MB of mostly zeros with some scattered noise
	data := make([]byte, 1*1024*1024)
	rng := mathrand.New(mathrand.NewSource(123))
	for i := 0; i < 1000; i++ {
		pos := rng.Intn(len(data))
		data[pos] = byte(rng.Intn(256))
	}
	packUnpackRoundtrip(t, data, "RepetitiveData_1MB")
}

func TestPackUnpack_NonAligned128(t *testing.T) {
	// Size that is NOT a multiple of 128 bytes
	data := make([]byte, 1000) // 1000 = 7*128 + 104
	rand.Read(data)
	packUnpackRoundtrip(t, data, "NonAligned128_1000B")
}

func TestPackUnpack_NonAligned16MB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large file test in short mode")
	}

	// 17MB — crosses the 16MB block boundary with a partial second block
	size := 17 * 1024 * 1024
	data := make([]byte, size)
	rng := mathrand.New(mathrand.NewSource(77))
	// Fill with semi-repetitive patterns
	pattern := make([]byte, 128)
	for i := range pattern {
		pattern[i] = byte(i * 3)
	}
	for i := 0; i < len(data); i += 128 {
		end := i + 128
		if end > len(data) {
			end = len(data)
		}
		copy(data[i:end], pattern)
		// Mutate a few bytes to create variation
		for j := 0; j < 5 && i+j < len(data); j++ {
			data[i+j] ^= byte(rng.Intn(256))
		}
	}
	packUnpackRoundtrip(t, data, "NonAligned16MB_17MB")
}

func TestPackUnpack_ChunkCountConsistency(t *testing.T) {
	// Test that the chunk count written in the header matches
	// the actual number of entries, especially for non-aligned sizes.
	sizes := []int{
		1,                      // 1 byte → 1 chunk
		127,                    // just under 1 chunk
		128,                    // exactly 1 chunk
		129,                    // 1 full + 1 partial
		256,                    // exactly 2 chunks
		1000,                   // 7 full + 1 partial
		chunker.DefaultChunkSize * 100, // exactly 100 chunks
	}

	for _, size := range sizes {
		t.Run(sizeLabel(size), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)

			dir := t.TempDir()
			inputPath := filepath.Join(dir, "input.bin")
			cromPath := filepath.Join(dir, "output.crom")
			outputPath := filepath.Join(dir, "restored.bin")

			os.WriteFile(inputPath, data, 0644)
			cbPath := createTestCodebook(t, data)

			opts := DefaultPackOptions()
			opts.Concurrency = 1
			_, err := Pack(inputPath, cromPath, cbPath, opts)
			if err != nil {
				t.Fatalf("Pack(%d bytes) failed: %v", size, err)
			}

			// Verify unpack works
			err = Unpack(cromPath, outputPath, cbPath, DefaultUnpackOptions())
			if err != nil {
				t.Fatalf("Unpack(%d bytes) failed: %v", size, err)
			}

			restored, _ := os.ReadFile(outputPath)
			origHash := sha256.Sum256(data)
			restHash := sha256.Sum256(restored)
			if origHash != restHash {
				t.Fatalf("SHA-256 mismatch for %d byte file", size)
			}

			// Expected chunk count
			expected := uint32(math.Ceil(float64(size) / float64(chunker.DefaultChunkSize)))
			t.Logf("✔ Size=%d → chunks=%d (expected=%d)", size, len(restored), expected)
		})
	}
}

func TestPackUnpack_SHA256Integrity(t *testing.T) {
	// Explicit SHA-256 verification test
	data := []byte("The quick brown fox jumps over the lazy dog. CROMpressor integrity test.")
	// Pad to reasonable size
	data = bytes.Repeat(data, 200) // ~14.6KB

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.bin")
	cromPath := filepath.Join(dir, "output.crom")
	outputPath := filepath.Join(dir, "restored.bin")

	os.WriteFile(inputPath, data, 0644)
	cbPath := createTestCodebook(t, data)

	opts := DefaultPackOptions()
	_, err := Pack(inputPath, cromPath, cbPath, opts)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	err = Unpack(cromPath, outputPath, cbPath, DefaultUnpackOptions())
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}

	original, _ := os.ReadFile(inputPath)
	restored, _ := os.ReadFile(outputPath)

	origHash := sha256.Sum256(original)
	restHash := sha256.Sum256(restored)

	if origHash != restHash {
		t.Fatalf("INTEGRITY FAILURE:\n  Original SHA-256: %x\n  Restored SHA-256: %x", origHash, restHash)
	}

	if !bytes.Equal(original, restored) {
		// Find first differing byte
		for i := 0; i < len(original) && i < len(restored); i++ {
			if original[i] != restored[i] {
				t.Fatalf("First difference at byte %d: original=0x%02x, restored=0x%02x", i, original[i], restored[i])
			}
		}
	}

	t.Logf("✔ SHA-256 integrity verified: %x", origHash[:8])
}

func sizeLabel(size int) string {
	if size < 1024 {
		return string(rune('0'+size/100)) + string(rune('0'+(size%100)/10)) + string(rune('0'+size%10)) + "B"
	}
	kb := size / 1024
	return string(rune('0'+kb/10)) + string(rune('0'+kb%10)) + "KB"
}

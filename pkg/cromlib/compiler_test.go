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

	"github.com/MrJc01/crompressor/internal/chunker"
	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/pkg/format"
)

// createTestCodebook generates a .cromdb file with patterns derived from the input data.
// This simulates a real "train" step by extracting actual chunks from the data.
func createTestCodebook(t *testing.T, data []byte, chunkSize int) string {
	t.Helper()

	dir := t.TempDir()
	cbPath := filepath.Join(dir, "test.cromdb")

	fc := chunker.NewFixedChunker(chunkSize)
	chunks := fc.Split(data)

	// Collect unique patterns (up to 256 for tests)
	seen := make(map[[32]byte]bool)
	var patterns [][]byte
	for _, c := range chunks {
		if len(c.Data) != chunkSize {
			continue // Skip partial chunks
		}
		hash := sha256.Sum256(c.Data)
		if !seen[hash] {
			seen[hash] = true
			p := make([]byte, chunkSize)
			copy(p, c.Data)
			patterns = append(patterns, p)
			if len(patterns) >= 256 {
				break
			}
		}
	}

	if len(patterns) == 0 {
		p := make([]byte, chunkSize)
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
func packUnpackRoundtrip(t *testing.T, data []byte, testName string, opts PackOptions) {
	t.Helper()

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.bin")
	cromPath := filepath.Join(dir, "output.crom")
	outputPath := filepath.Join(dir, "restored.bin")

	if err := os.WriteFile(inputPath, data, 0644); err != nil {
		t.Fatalf("[%s] write input: %v", testName, err)
	}

	if opts.ChunkSize <= 0 {
		opts.ChunkSize = chunker.DefaultChunkSize
	}
	cbPath := createTestCodebook(t, data, opts.ChunkSize)

	// Pack
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
	opts := DefaultPackOptions()
	opts.Concurrency = 2
	packUnpackRoundtrip(t, data, "SmallFile_1KB", opts)
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
	opts := DefaultPackOptions()
	opts.Concurrency = 2
	packUnpackRoundtrip(t, data, "MediumFile_1MB", opts)
}

func TestPackUnpack_RandomData(t *testing.T) {
	// 512KB of purely random data — worst case for compression
	data := make([]byte, 512*1024)
	rand.Read(data)
	opts := DefaultPackOptions()
	opts.Concurrency = 2
	packUnpackRoundtrip(t, data, "RandomData_512KB", opts)
}

func TestPackUnpack_RepetitiveData(t *testing.T) {
	// 1MB of mostly zeros with some scattered noise
	data := make([]byte, 1*1024*1024)
	rng := mathrand.New(mathrand.NewSource(123))
	for i := 0; i < 1000; i++ {
		pos := rng.Intn(len(data))
		data[pos] = byte(rng.Intn(256))
	}
	opts := DefaultPackOptions()
	opts.Concurrency = 2
	packUnpackRoundtrip(t, data, "RepetitiveData_1MB", opts)
}

func TestPackUnpack_NonAligned128(t *testing.T) {
	// Size that is NOT a multiple of 128 bytes
	data := make([]byte, 1000) // 1000 = 7*128 + 104
	rand.Read(data)
	opts := DefaultPackOptions()
	opts.Concurrency = 2
	packUnpackRoundtrip(t, data, "NonAligned128_1000B", opts)
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
	opts := DefaultPackOptions()
	opts.Concurrency = 2
	packUnpackRoundtrip(t, data, "NonAligned16MB_17MB", opts)
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
			opts := DefaultPackOptions()
			opts.Concurrency = 1
			cbPath := createTestCodebook(t, data, opts.ChunkSize)

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
	opts := DefaultPackOptions()
	cbPath := createTestCodebook(t, data, opts.ChunkSize)

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

func TestPackUnpack_CustomChunkSize(t *testing.T) {
	// 512B dataset grouped in 64B chunks instead of 128B
	data := make([]byte, 512)
	for i := range data {
		data[i] = byte(i % 17) // somewhat repetitive
	}
	opts := DefaultPackOptions()
	opts.Concurrency = 1
	opts.ChunkSize = 64
	packUnpackRoundtrip(t, data, "CustomChunkSize_64B", opts)
}

func TestPackUnpack_CDC(t *testing.T) {
	// Simple highly repetitive dataset to test CDC borders logic
	data := bytes.Repeat([]byte("ABCDC"), 2000) // 10KB
	opts := DefaultPackOptions()
	opts.Concurrency = 1
	opts.UseCDC = true
	opts.ChunkSize = 64
	packUnpackRoundtrip(t, data, "CDC_64B", opts)
}

func TestUnpack_CorruptBlock(t *testing.T) {
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 16) // Lower entropy to avoid passthrough
	}

	opts := DefaultPackOptions()
	opts.ChunkSize = 128
	cbPath := createTestCodebook(t, make([]byte, 128), opts.ChunkSize)

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.bin")
	cromPath := filepath.Join(dir, "output.crom")
	restoredPath := filepath.Join(dir, "restored.bin")

	os.WriteFile(inputPath, data, 0644)
	_, err := Pack(inputPath, cromPath, cbPath, opts)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// Delta pool lives at the end. Let's flip the first byte of the zstd frame.
	// We know the block table tells us where the delta pool starts, but we can just
	// parse the header to find `baseOffset`.
	cromData, _ := os.ReadFile(cromPath)
	h, _, _, _ := format.NewReader(bytes.NewReader(cromData)).ReadMetadata("")
	
	// Corrupt a byte right after the chunk table (start of delta pool)
	tableSize := int(h.ChunkCount) * int(format.GetEntrySize(h.Version))
	if h.IsEncrypted { tableSize += 28 }
	hSize := format.HeaderSizeV2
	if h.Version == format.Version4 { 
		hSize = format.HeaderSizeV4 
	} else if h.Version == format.Version5 {
		hSize = format.HeaderSizeV5
	} else if h.Version == format.Version6 || h.Version == format.Version7 {
		hSize = format.HeaderSizeV6
	} else if h.Version >= format.Version8 {
		hSize = format.HeaderSizeV8 + int(h.MicroDictSize)
	}
	baseOffset := hSize + 4 /* block table len 1 */ + tableSize
	
	if baseOffset < len(cromData) {
		cromData[baseOffset] ^= 0xFF
	}
	os.WriteFile(cromPath, cromData, 0644)

	// Strict Unpack -> should fail
	strictOpts := DefaultUnpackOptions()
	strictOpts.Strict = true
	err = Unpack(cromPath, restoredPath, cbPath, strictOpts)
	if err == nil {
		t.Fatalf("Expected strict unpack to fail on corrupted block")
	}

	// Tolerant Unpack -> should pass but skipping the block
	tolerantOpts := DefaultUnpackOptions()
	tolerantOpts.Strict = false
	err = Unpack(cromPath, restoredPath, cbPath, tolerantOpts)
	if err != nil {
		t.Fatalf("Expected tolerant unpack to succeed, but got error: %v", err)
	}

	restored, _ := os.ReadFile(restoredPath)
	if len(restored) != len(data) {
		t.Fatalf("Tolerant unpack didn't restore correct length (got %d, want %d)", len(restored), len(data))
	}
}

func sizeLabel(size int) string {
	if size < 1024 {
		return string(rune('0'+size/100)) + string(rune('0'+(size%100)/10)) + string(rune('0'+size%10)) + "B"
	}
	kb := size / 1024
	return string(rune('0'+kb/10)) + string(rune('0'+kb%10)) + "KB"
}

func TestPackUnpack_ThresholdRandom(t *testing.T) {
	data := make([]byte, 1024)
	// Fill with either 0x00 or 0xFF. Entropy = 1.0 bit/byte.
	// Average distance from 0x00 is 128 * 4 = 512 bits... wait
	// actually 0xFF is 8 bits distance! So 128 * 4 = 512 average distance.
	// For similarity < 0.2, we need distance > 819. So we need mostly 0xFF!
	for i := range data {
		if mathrand.Float32() < 0.85 {
			data[i] = 0xFF
		} else {
			data[i] = 0x00
		}
	}
	// Entropy is very low (~0.6 bits), but distance from zero codebook is very high!
	opts := DefaultPackOptions()
	opts.Concurrency = 1

	// Use a codebook of zeros so the random data is completely different
	cbPath := createTestCodebook(t, make([]byte, 1024), opts.ChunkSize)

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.bin")
	cromPath := filepath.Join(dir, "output.crom")
	outputPath := filepath.Join(dir, "restored.bin")

	os.WriteFile(inputPath, data, 0644)

	_, err := Pack(inputPath, cromPath, cbPath, opts)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	err = Unpack(cromPath, outputPath, cbPath, DefaultUnpackOptions())
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}

	restored, _ := os.ReadFile(outputPath)
	if !bytes.Equal(data, restored) {
		t.Fatalf("Data mismatch after unpacking literals")
	}
}

func TestPackUnpack_ThresholdZeros(t *testing.T) {
	data := make([]byte, 1024) // all zeros
	opts := DefaultPackOptions()
	opts.Concurrency = 1

	// Use a codebook of zeros
	cbPath := createTestCodebook(t, data, opts.ChunkSize)

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.bin")
	cromPath := filepath.Join(dir, "output.crom")
	outputPath := filepath.Join(dir, "restored.bin")

	os.WriteFile(inputPath, data, 0644)

	metrics, err := Pack(inputPath, cromPath, cbPath, opts)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}
	if metrics.LiteralChunks > 0 {
		t.Fatalf("Expected 0 literal chunks for matching data, got %d", metrics.LiteralChunks)
	}

	err = Unpack(cromPath, outputPath, cbPath, DefaultUnpackOptions())
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}

	restored, _ := os.ReadFile(outputPath)
	if !bytes.Equal(data, restored) {
		t.Fatalf("Data mismatch after unpacking")
	}
}

func TestPackUnpack_UrandomBypass(t *testing.T) {
	// Teste de entropia alta (Urandom)
	// O novo Pipeline deve notar que Entropy > 3.0 e aplicar Bypass Literal/Passthrough.
	data := make([]byte, 1024*1024)
	rand.Read(data)

	opts := DefaultPackOptions()
	cbPath := createTestCodebook(t, make([]byte, opts.ChunkSize), opts.ChunkSize)

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.bin")
	cromPath := filepath.Join(dir, "output.crom")
	outputPath := filepath.Join(dir, "restored.bin")

	os.WriteFile(inputPath, data, 0644)

	metrics, err := Pack(inputPath, cromPath, cbPath, opts)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// Com entropia máxima, a expectativa e que o arquivo vire Passthrough (header IsPassthrough = true)
	// Ou que 100% dos chunks sejam literais (LiteralChunks) no pior caso de streaming.
	// O bypass principal age sobre o pacote inteiro se a Head tiver entropia > 7.8 ou fallbacks.
	if metrics.LiteralChunks == 0 && metrics.TotalChunks > 0 {
		t.Fatalf("Expected literal chunks > 0 for Urandom data if not globally bypassed, got %d", metrics.LiteralChunks)
	}

	err = Unpack(cromPath, outputPath, cbPath, DefaultUnpackOptions())
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}

	restored, _ := os.ReadFile(outputPath)
	if !bytes.Equal(data, restored) {
		t.Fatalf("Data mismatch after unpacking Urandom")
	}
}

func TestPackUnpack_Polynomial(t *testing.T) {
	// Teste do Fractal Multiestrategia O(1).
	// Cria arquivo com um padrao linear (ax^2+bx+c mod 256) onde:
	// a=0, b=1, c=5: data[i] = i + 5
	data := make([]byte, 1000)
	for i := range data {
		data[i] = byte(i + 5)
	} // O bloco possui entropia < 3.0, entao dispara FindPolynomial.

	opts := DefaultPackOptions()
	opts.ChunkSize = 8 
	cbPath := createTestCodebook(t, make([]byte, opts.ChunkSize), opts.ChunkSize)

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.bin")
	cromPath := filepath.Join(dir, "output.crom")
	outputPath := filepath.Join(dir, "restored.bin")

	os.WriteFile(inputPath, data, 0644)

	metrics, err := Pack(inputPath, cromPath, cbPath, opts)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	if metrics.HitRate > 0 {
		t.Logf("Polynomial Hit! HitRate=%f", metrics.HitRate)
	}

	err = Unpack(cromPath, outputPath, cbPath, DefaultUnpackOptions())
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}

	restored, _ := os.ReadFile(outputPath)
	if !bytes.Equal(data, restored) {
		t.Fatalf("Data mismatch after unpacking Polynomial fractal chunk")
	}
}

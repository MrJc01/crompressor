package vfs

import (
	"bytes"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/pkg/cromlib"
	"github.com/MrJc01/crompressor/pkg/format"
)

// TestRandomAccessStress packs synthetic data, then performs hundreds of
// random-offset reads via RandomReader and validates each fragment against
// the original data. This validates the entire chain: format parsing,
// block offset calculation, LRU cache, Zstd decompression, AES decryption
// (when enabled), and XOR delta reconstruction.
func TestRandomAccessStress(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	codebookPath := findCodebook(t)

	// Generate synthetic data: repeating pattern to get good codebook hits
	const dataSize = 256 * 1024 // 256 KB
	original := makeSyntheticData(dataSize)

	// Pack to a temp .crom file
	cromFile := packToTemp(t, original, codebookPath, "")

	// Open and create RandomReader
	rr := openRandomReader(t, cromFile, codebookPath, "")

	// Stress: 500 random reads
	const numReads = 500
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	var totalLatency time.Duration
	var latencies []time.Duration

	for i := 0; i < numReads; i++ {
		maxOff := int64(dataSize - 1)
		off := rng.Int63n(maxOff)
		maxLen := int64(dataSize) - off
		readLen := rng.Int63n(min64(maxLen, 4096)) + 1

		buf := make([]byte, readLen)

		start := time.Now()
		n, err := rr.ReadAt(buf, off)
		elapsed := time.Since(start)

		if err != nil && err.Error() != "EOF" {
			t.Fatalf("read #%d at off=%d len=%d failed: %v", i, off, readLen, err)
		}

		if n == 0 && off < int64(dataSize) {
			t.Fatalf("read #%d at off=%d returned 0 bytes", i, off)
		}

		// Compare with original
		expected := original[off : off+int64(n)]
		if !bytes.Equal(buf[:n], expected) {
			t.Fatalf("read #%d MISMATCH at off=%d len=%d:\n  got:  %x\n  want: %x",
				i, off, n, buf[:minInt(n, 32)], expected[:minInt(len(expected), 32)])
		}

		totalLatency += elapsed
		latencies = append(latencies, elapsed)
	}

	// Report P50 and P99
	sortDurations(latencies)
	p50 := latencies[len(latencies)*50/100]
	p99 := latencies[len(latencies)*99/100]

	t.Logf("✔ %d random reads passed", numReads)
	t.Logf("  Total:  %v", totalLatency)
	t.Logf("  Avg:    %v", totalLatency/time.Duration(numReads))
	t.Logf("  P50:    %v", p50)
	t.Logf("  P99:    %v", p99)
}

// TestRandomAccessEncrypted is the same stress test but with AES-256-GCM encryption.
func TestRandomAccessEncrypted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping encrypted stress test in short mode")
	}

	codebookPath := findCodebook(t)
	const password = "SoberaniaStress2026"
	const dataSize = 128 * 1024

	original := makeSyntheticData(dataSize)
	cromFile := packToTemp(t, original, codebookPath, password)
	rr := openRandomReader(t, cromFile, codebookPath, password)

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 200; i++ {
		off := rng.Int63n(int64(dataSize - 1))
		readLen := rng.Int63n(min64(int64(dataSize)-off, 2048)) + 1

		buf := make([]byte, readLen)
		n, err := rr.ReadAt(buf, off)
		if err != nil && err.Error() != "EOF" {
			t.Fatalf("encrypted read #%d at off=%d failed: %v", i, off, err)
		}

		expected := original[off : off+int64(n)]
		if !bytes.Equal(buf[:n], expected) {
			t.Fatalf("encrypted read #%d MISMATCH at off=%d", i, off)
		}
	}

	t.Log("✔ 200 encrypted random reads passed")
}

// --- Helpers ---

func findCodebook(t *testing.T) string {
	t.Helper()
	paths := []string{
		"../../testdata/trained.cromdb",
		"testdata/trained.cromdb",
		"../../testdata/mini.cromdb",
		"testdata/mini.cromdb",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("no codebook found; run 'make gen-codebook' or 'make train-standard' first")
	return ""
}

func makeSyntheticData(size int) []byte {
	data := make([]byte, size)
	// Create a repeated pattern to get codebook hits, with some variation
	pattern := []byte("package main\n\nfunc main() {\n\tfmt.Println(\"Hello, CROM World!\")\n}\n\n// This is a synthetic test file for stress testing.\n// It repeats enough patterns to get good codebook coverage.\n\n")
	for i := 0; i < size; i++ {
		data[i] = pattern[i%len(pattern)]
	}
	// Add some distinct regions
	rng := rand.New(rand.NewSource(12345))
	for i := size / 4; i < size/4+1024; i++ {
		data[i] = byte(rng.Intn(256))
	}
	return data
}

func packToTemp(t *testing.T, data []byte, codebookPath, password string) string {
	t.Helper()

	// Write original data to temp file
	tmpIn, err := os.CreateTemp("", "crom_stress_in_*.dat")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpIn.Write(data); err != nil {
		t.Fatal(err)
	}
	tmpIn.Close()
	t.Cleanup(func() { os.Remove(tmpIn.Name()) })

	// Prepare output temp file
	tmpOut, err := os.CreateTemp("", "crom_stress_out_*.crom")
	if err != nil {
		t.Fatal(err)
	}
	tmpOut.Close()
	t.Cleanup(func() { os.Remove(tmpOut.Name()) })

	// Pack
	opts := cromlib.DefaultPackOptions()
	if password != "" {
		opts.EncryptionKey = password
	}
	_, err = cromlib.Pack(tmpIn.Name(), tmpOut.Name(), codebookPath, opts)
	if err != nil {
		t.Fatalf("pack failed: %v", err)
	}

	return tmpOut.Name()
}

func openRandomReader(t *testing.T, cromFile, codebookPath, password string) *RandomReader {
	t.Helper()

	f, err := os.Open(cromFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })

	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}

	cb, err := codebook.Open(codebookPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cb.Close() })

	reader := format.NewReader(f)
	header, blockTable, entries, err := reader.ReadMetadata(password)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}

	rr, err := NewRandomReader(f, info.Size(), header, blockTable, entries, cb, password)
	if err != nil {
		t.Fatalf("new random reader: %v", err)
	}

	return rr
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sortDurations(d []time.Duration) {
	for i := 1; i < len(d); i++ {
		for j := i; j > 0 && d[j] < d[j-1]; j-- {
			d[j], d[j-1] = d[j-1], d[j]
		}
	}
}


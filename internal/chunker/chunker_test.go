package chunker

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"testing"
)

func TestFixedChunker_Basic(t *testing.T) {
	// 256 bytes should produce exactly 2 chunks of 128 bytes each.
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i % 256)
	}

	fc := NewFixedChunker(DefaultChunkSize)
	chunks := fc.Split(data)

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	for i, c := range chunks {
		if c.Size != 128 {
			t.Errorf("chunk[%d]: expected size 128, got %d", i, c.Size)
		}
		if c.Offset != uint64(i*128) {
			t.Errorf("chunk[%d]: expected offset %d, got %d", i, i*128, c.Offset)
		}
		if len(c.Data) != 128 {
			t.Errorf("chunk[%d]: expected data length 128, got %d", i, len(c.Data))
		}
		if c.Hash == 0 {
			t.Errorf("chunk[%d]: hash should not be zero", i)
		}
	}
}

func TestFixedChunker_NonAligned(t *testing.T) {
	// 200 bytes: first chunk 128 bytes, second chunk 72 bytes.
	data := make([]byte, 200)
	rand.Read(data)

	fc := NewFixedChunker(DefaultChunkSize)
	chunks := fc.Split(data)

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	if chunks[0].Size != 128 {
		t.Errorf("chunk[0]: expected size 128, got %d", chunks[0].Size)
	}
	if chunks[1].Size != 72 {
		t.Errorf("chunk[1]: expected size 72, got %d", chunks[1].Size)
	}
	if chunks[1].Offset != 128 {
		t.Errorf("chunk[1]: expected offset 128, got %d", chunks[1].Offset)
	}
}

func TestFixedChunker_Empty(t *testing.T) {
	fc := NewFixedChunker(DefaultChunkSize)
	chunks := fc.Split(nil)

	if chunks != nil {
		t.Fatalf("expected nil chunks for empty data, got %d chunks", len(chunks))
	}

	chunks = fc.Split([]byte{})
	if chunks != nil {
		t.Fatalf("expected nil chunks for zero-length data, got %d chunks", len(chunks))
	}
}

func TestFixedChunker_SingleByte(t *testing.T) {
	data := []byte{0xFF}

	fc := NewFixedChunker(DefaultChunkSize)
	chunks := fc.Split(data)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Size != 1 {
		t.Errorf("expected size 1, got %d", chunks[0].Size)
	}
	if chunks[0].Offset != 0 {
		t.Errorf("expected offset 0, got %d", chunks[0].Offset)
	}
	if !bytes.Equal(chunks[0].Data, data) {
		t.Errorf("chunk data mismatch")
	}
}

func TestFixedChunker_Reassemble(t *testing.T) {
	// Generate random data and verify SHA-256 roundtrip.
	data := make([]byte, 1000)
	rand.Read(data)

	originalHash := sha256.Sum256(data)

	fc := NewFixedChunker(DefaultChunkSize)
	chunks := fc.Split(data)

	reassembled := Reassemble(chunks)
	reassembledHash := sha256.Sum256(reassembled)

	if originalHash != reassembledHash {
		t.Fatalf("SHA-256 mismatch: original=%x reassembled=%x", originalHash[:8], reassembledHash[:8])
	}

	if !bytes.Equal(data, reassembled) {
		t.Fatal("reassembled data does not match original")
	}
}

func TestFixedChunker_LargeData(t *testing.T) {
	// 1MB of random data.
	data := make([]byte, 1024*1024)
	rand.Read(data)

	fc := NewFixedChunker(DefaultChunkSize)
	chunks := fc.Split(data)

	expectedChunks := (1024 * 1024) / DefaultChunkSize
	if len(chunks) != expectedChunks {
		t.Fatalf("expected %d chunks, got %d", expectedChunks, len(chunks))
	}

	// Roundtrip check
	reassembled := Reassemble(chunks)
	if !bytes.Equal(data, reassembled) {
		t.Fatal("roundtrip failed for 1MB data")
	}
}

func TestFixedChunker_CustomSize(t *testing.T) {
	data := make([]byte, 300)
	rand.Read(data)

	fc := NewFixedChunker(64) // 64-byte chunks
	chunks := fc.Split(data)

	// 300 / 64 = 4 full + 1 partial (44 bytes) = 5 chunks
	if len(chunks) != 5 {
		t.Fatalf("expected 5 chunks with size 64, got %d", len(chunks))
	}

	if chunks[4].Size != 44 {
		t.Errorf("last chunk: expected size 44, got %d", chunks[4].Size)
	}

	reassembled := Reassemble(chunks)
	if !bytes.Equal(data, reassembled) {
		t.Fatal("roundtrip failed for custom chunk size")
	}
}

func TestFixedChunker_DefaultSize(t *testing.T) {
	fc := NewFixedChunker(0) // Should default to 128
	if fc.ChunkSize != DefaultChunkSize {
		t.Errorf("expected default chunk size %d, got %d", DefaultChunkSize, fc.ChunkSize)
	}

	fc = NewFixedChunker(-1) // Negative should also default
	if fc.ChunkSize != DefaultChunkSize {
		t.Errorf("expected default chunk size %d for negative input, got %d", DefaultChunkSize, fc.ChunkSize)
	}
}

func TestFixedChunker_HashConsistency(t *testing.T) {
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}

	fc := NewFixedChunker(DefaultChunkSize)

	// Split twice; hashes must be identical (deterministic).
	chunks1 := fc.Split(data)
	chunks2 := fc.Split(data)

	for i := range chunks1 {
		if chunks1[i].Hash != chunks2[i].Hash {
			t.Errorf("chunk[%d]: hash not deterministic: %d vs %d", i, chunks1[i].Hash, chunks2[i].Hash)
		}
	}
}

func BenchmarkFixedChunker_1MB(b *testing.B) {
	data := make([]byte, 1024*1024)
	rand.Read(data)
	fc := NewFixedChunker(DefaultChunkSize)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		fc.Split(data)
	}
}

func TestCDCInsertion(t *testing.T) {
	// Generate 100KB of random data
	data := make([]byte, 100*1024)
	rand.Read(data)

	fc := NewFastCDCChunker(128)
	chunks1 := fc.Split(data)

	// Insert 1 byte at position 500
	mutated := make([]byte, 0, len(data)+1)
	mutated = append(mutated, data[:500]...)
	mutated = append(mutated, 0xFF)
	mutated = append(mutated, data[500:]...)

	chunks2 := fc.Split(mutated)

	// We expect the vast majority of chunks to remain identical
	// Let's count matching chunk hashes
	hashes1 := make(map[uint64]bool)
	for _, c := range chunks1 {
		hashes1[c.Hash] = true
	}

	matchCount := 0
	for _, c := range chunks2 {
		if hashes1[c.Hash] {
			matchCount++
		}
	}

	// Calculate percentage of matching chunks
	matchRatio := float64(matchCount) / float64(len(chunks1))
	
	// FastCDC should preserve at least 95% of the chunks after a single byte insertion
	if matchRatio < 0.95 {
		t.Fatalf("FastCDC failed to resist byte shifting. Match ratio: %.2f%% (Expected >95%%)", matchRatio*100)
	}
	
	t.Logf("FastCDC byte-shift resistance: %.2f%% chunks intact", matchRatio*100)
}


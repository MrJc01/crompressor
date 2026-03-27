package format

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestFormat_Roundtrip(t *testing.T) {
	// Create sample data
	origHash := sha256.Sum256([]byte("hello world"))

	header := &Header{
		Version:      Version1,
		OriginalHash: origHash,
		OriginalSize: 1024,
		ChunkCount:   2,
	}

	entries := []ChunkEntry{
		{CodebookID: 42, DeltaOffset: 0, DeltaSize: 64, OriginalSize: 128},
		{CodebookID: 99, DeltaOffset: 64, DeltaSize: 10, OriginalSize: 128},
	}

	compDeltaPool := []byte{0xDE, 0xAD, 0xBE, 0xEF} // mock compressed data

	// Write
	var buf bytes.Buffer
	writer := NewWriter(&buf)
	err := writer.Write(header, entries, compDeltaPool)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read
	reader := NewReader(&buf)
	h2, entries2, compDeltaPool2, err := reader.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Verify Header
	if h2.Version != header.Version {
		t.Errorf("Header Version mismatch: got %d, want %d", h2.Version, header.Version)
	}
	if h2.OriginalHash != header.OriginalHash {
		t.Errorf("Header OriginalHash mismatch")
	}
	if h2.OriginalSize != header.OriginalSize {
		t.Errorf("Header OriginalSize mismatch: got %d, want %d", h2.OriginalSize, header.OriginalSize)
	}
	if h2.ChunkCount != header.ChunkCount {
		t.Errorf("Header ChunkCount mismatch: got %d, want %d", h2.ChunkCount, header.ChunkCount)
	}

	// Verify Entries
	if len(entries2) != len(entries) {
		t.Fatalf("Entries length mismatch: got %d, want %d", len(entries2), len(entries))
	}
	for i := range entries {
		if entries2[i] != entries[i] {
			t.Errorf("Entry %d mismatch:\ngot  %+v\nwant %+v", i, entries2[i], entries[i])
		}
	}

	// Verify Delta Pool
	if !bytes.Equal(compDeltaPool2, compDeltaPool) {
		t.Errorf("Delta Pool mismatch: got %x, want %x", compDeltaPool2, compDeltaPool)
	}
}

func TestFormat_InvalidMagic(t *testing.T) {
	buf := make([]byte, HeaderSize)
	copy(buf[0:4], "BAD!") // Corrupt magic

	reader := NewReader(bytes.NewReader(buf))
	_, _, _, err := reader.Read()
	if err == nil {
		t.Fatal("expected error for invalid magic")
	}
}

func TestFormat_TruncatedHeader(t *testing.T) {
	buf := make([]byte, 20) // Much smaller than HeaderSize
	reader := NewReader(bytes.NewReader(buf))
	_, _, _, err := reader.Read()
	if err == nil {
		t.Fatal("expected error for truncated header")
	}
}

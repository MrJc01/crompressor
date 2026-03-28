package format

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestFormat_V1_Roundtrip(t *testing.T) {
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

	compDeltaPool := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	// Write V1 (no block table)
	var buf bytes.Buffer
	writer := NewWriter(&buf)
	err := writer.Write(header, nil, entries, compDeltaPool)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read V1
	reader := NewReader(&buf)
	h2, _, entries2, compDeltaPool2, err := reader.Read("")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

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

	if len(entries2) != len(entries) {
		t.Fatalf("Entries length mismatch: got %d, want %d", len(entries2), len(entries))
	}
	for i := range entries {
		if entries2[i] != entries[i] {
			t.Errorf("Entry %d mismatch:\ngot  %+v\nwant %+v", i, entries2[i], entries[i])
		}
	}

	if !bytes.Equal(compDeltaPool2, compDeltaPool) {
		t.Errorf("Delta Pool mismatch: got %x, want %x", compDeltaPool2, compDeltaPool)
	}
}

func TestFormat_V2_Roundtrip(t *testing.T) {
	origHash := sha256.Sum256([]byte("crompressor v2"))

	header := &Header{
		Version:      Version2,
		OriginalHash: origHash,
		OriginalSize: 2048,
		ChunkCount:   4,
	}

	entries := []ChunkEntry{
		{CodebookID: 1, DeltaOffset: 0, DeltaSize: 128, OriginalSize: 128},
		{CodebookID: 2, DeltaOffset: 128, DeltaSize: 128, OriginalSize: 128},
		{CodebookID: 3, DeltaOffset: 256, DeltaSize: 128, OriginalSize: 128},
		{CodebookID: 4, DeltaOffset: 384, DeltaSize: 64, OriginalSize: 64},
	}

	blockTable := []uint32{350} // One compressed block (NumBlocks = ceil(4/8192) = 1)
	compDeltaPool := make([]byte, 350)
	for i := range compDeltaPool {
		compDeltaPool[i] = byte(i % 256)
	}

	var buf bytes.Buffer
	writer := NewWriter(&buf)
	err := writer.Write(header, blockTable, entries, compDeltaPool)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	reader := NewReader(&buf)
	h2, bt2, entries2, pool2, err := reader.Read("")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Header checks
	if h2.Version != Version2 {
		t.Errorf("Version: got %d, want %d", h2.Version, Version2)
	}
	if h2.OriginalHash != origHash {
		t.Errorf("OriginalHash mismatch")
	}
	if h2.OriginalSize != 2048 {
		t.Errorf("OriginalSize: got %d, want 2048", h2.OriginalSize)
	}
	if h2.ChunkCount != 4 {
		t.Errorf("ChunkCount: got %d, want 4", h2.ChunkCount)
	}

	// Block table checks
	if len(bt2) != len(blockTable) {
		t.Fatalf("BlockTable length mismatch: got %d, want %d", len(bt2), len(blockTable))
	}
	for i, v := range blockTable {
		if bt2[i] != v {
			t.Errorf("BlockTable[%d]: got %d, want %d", i, bt2[i], v)
		}
	}

	// Entries checks
	if len(entries2) != len(entries) {
		t.Fatalf("Entries length mismatch: got %d, want %d", len(entries2), len(entries))
	}
	for i := range entries {
		if entries2[i] != entries[i] {
			t.Errorf("Entry %d mismatch:\ngot  %+v\nwant %+v", i, entries2[i], entries[i])
		}
	}

	// Delta pool
	if !bytes.Equal(pool2, compDeltaPool) {
		t.Errorf("Delta Pool mismatch")
	}
}

func TestFormat_V2_ChunkEntrySerialization(t *testing.T) {
	entries := []ChunkEntry{
		{CodebookID: 0, DeltaOffset: 0, DeltaSize: 0, OriginalSize: 128},
		{CodebookID: ^uint64(0), DeltaOffset: ^uint64(0), DeltaSize: ^uint32(0), OriginalSize: ^uint32(0)},
		{CodebookID: 42, DeltaOffset: 1000, DeltaSize: 512, OriginalSize: 128},
	}

	data := SerializeChunkTable(entries)
	parsed, err := ParseChunkTable(data, uint32(len(entries)))
	if err != nil {
		t.Fatalf("ParseChunkTable failed: %v", err)
	}

	for i := range entries {
		if parsed[i] != entries[i] {
			t.Errorf("ChunkEntry[%d] roundtrip mismatch:\ngot  %+v\nwant %+v", i, parsed[i], entries[i])
		}
	}
}

func TestFormat_InvalidMagic(t *testing.T) {
	buf := make([]byte, HeaderSizeV2+100)
	copy(buf[0:4], "BAD!")

	reader := NewReader(bytes.NewReader(buf))
	_, _, _, _, err := reader.Read("")
	if err == nil {
		t.Fatal("expected error for invalid magic")
	}
}

func TestFormat_TruncatedHeader(t *testing.T) {
	buf := make([]byte, 20)
	reader := NewReader(bytes.NewReader(buf))
	_, _, _, _, err := reader.Read("")
	if err == nil {
		t.Fatal("expected error for truncated header")
	}
}

func TestFormat_V2_HeaderSerializeRoundtrip(t *testing.T) {
	origHash := sha256.Sum256([]byte("integrity"))
	salt := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	h := &Header{
		Version:      Version2,
		IsEncrypted:  true,
		Salt:         salt,
		OriginalHash: origHash,
		OriginalSize: 1_000_000,
		ChunkCount:   7813,
	}

	data := h.Serialize()
	if len(data) != HeaderSizeV2 {
		t.Fatalf("serialized V2 header size: got %d, want %d", len(data), HeaderSizeV2)
	}

	h2, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("ParseHeader failed: %v", err)
	}

	if h2.Version != Version2 {
		t.Errorf("Version mismatch")
	}
	if !h2.IsEncrypted {
		t.Errorf("IsEncrypted should be true")
	}
	if h2.Salt != salt {
		t.Errorf("Salt mismatch")
	}
	if h2.OriginalHash != origHash {
		t.Errorf("OriginalHash mismatch")
	}
	if h2.OriginalSize != 1_000_000 {
		t.Errorf("OriginalSize mismatch")
	}
	if h2.ChunkCount != 7813 {
		t.Errorf("ChunkCount mismatch")
	}
}

package sync

import (
	"testing"
)

func TestManifestJSONRoundTrip(t *testing.T) {
	orig := &ChunkManifest{
		Version:      ManifestVersion,
		OriginalHash: [32]byte{0xAA, 0xBB, 0xCC},
		OriginalSize: 1024 * 1024,
		ChunkCount:   3,
		Entries: []ManifestEntry{
			{CodebookID: 100, DeltaHash: 0xDEADBEEF, ChunkSize: 128},
			{CodebookID: 200, DeltaHash: 0xCAFEBABE, ChunkSize: 128},
			{CodebookID: 300, DeltaHash: 0xFACEFEED, ChunkSize: 64},
		},
	}

	data, err := orig.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	assertManifestsEqual(t, orig, restored)
}

func TestManifestBinaryRoundTrip(t *testing.T) {
	orig := &ChunkManifest{
		Version:      ManifestVersion,
		OriginalHash: [32]byte{0x11, 0x22, 0x33, 0x44},
		OriginalSize: 2 * 1024 * 1024,
		ChunkCount:   4,
		Entries: []ManifestEntry{
			{CodebookID: 10, DeltaHash: 0x1111111111111111, ChunkSize: 128},
			{CodebookID: 20, DeltaHash: 0x2222222222222222, ChunkSize: 128},
			{CodebookID: 30, DeltaHash: 0x3333333333333333, ChunkSize: 128},
			{CodebookID: 40, DeltaHash: 0x4444444444444444, ChunkSize: 96},
		},
	}

	data := orig.ToBinary()
	restored, err := FromBinary(data)
	if err != nil {
		t.Fatalf("FromBinary failed: %v", err)
	}

	assertManifestsEqual(t, orig, restored)
}

func TestManifestDiff(t *testing.T) {
	local := &ChunkManifest{
		Entries: []ManifestEntry{
			{CodebookID: 1, DeltaHash: 0xAAA, ChunkSize: 128},
			{CodebookID: 2, DeltaHash: 0xBBB, ChunkSize: 128},
			{CodebookID: 3, DeltaHash: 0xCCC, ChunkSize: 128},
		},
	}

	remote := &ChunkManifest{
		Entries: []ManifestEntry{
			{CodebookID: 2, DeltaHash: 0xBBB, ChunkSize: 128}, // shared
			{CodebookID: 4, DeltaHash: 0xDDD, ChunkSize: 128}, // missing locally
			{CodebookID: 5, DeltaHash: 0xEEE, ChunkSize: 64},  // missing locally
		},
	}

	result := Diff(local, remote)

	// Missing: 4+5 (in remote, not local)
	if len(result.Missing) != 2 {
		t.Errorf("expected 2 missing, got %d", len(result.Missing))
	}

	// Extra: 1+3 (in local, not remote)
	if len(result.Extra) != 2 {
		t.Errorf("expected 2 extra, got %d", len(result.Extra))
	}

	// Verify the missing entries are correct
	missingIDs := map[uint64]bool{}
	for _, e := range result.Missing {
		missingIDs[e.CodebookID] = true
	}
	if !missingIDs[4] || !missingIDs[5] {
		t.Errorf("missing entries should contain CodebookIDs 4 and 5, got %v", result.Missing)
	}

	// Verify the extra entries are correct
	extraIDs := map[uint64]bool{}
	for _, e := range result.Extra {
		extraIDs[e.CodebookID] = true
	}
	if !extraIDs[1] || !extraIDs[3] {
		t.Errorf("extra entries should contain CodebookIDs 1 and 3, got %v", result.Extra)
	}
}

func TestManifestBinaryTruncated(t *testing.T) {
	_, err := FromBinary([]byte{0x01, 0x02})
	if err == nil {
		t.Fatal("expected error for truncated binary, got nil")
	}
}

func assertManifestsEqual(t *testing.T, a, b *ChunkManifest) {
	t.Helper()

	if a.Version != b.Version {
		t.Errorf("version mismatch: %d vs %d", a.Version, b.Version)
	}
	if a.OriginalHash != b.OriginalHash {
		t.Errorf("hash mismatch: %x vs %x", a.OriginalHash[:8], b.OriginalHash[:8])
	}
	if a.OriginalSize != b.OriginalSize {
		t.Errorf("size mismatch: %d vs %d", a.OriginalSize, b.OriginalSize)
	}
	if a.ChunkCount != b.ChunkCount {
		t.Errorf("chunkCount mismatch: %d vs %d", a.ChunkCount, b.ChunkCount)
	}
	if len(a.Entries) != len(b.Entries) {
		t.Fatalf("entry count mismatch: %d vs %d", len(a.Entries), len(b.Entries))
	}
	for i := range a.Entries {
		if a.Entries[i] != b.Entries[i] {
			t.Errorf("entry %d mismatch: %+v vs %+v", i, a.Entries[i], b.Entries[i])
		}
	}
}

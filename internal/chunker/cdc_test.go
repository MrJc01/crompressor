package chunker

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestCDC_Split_Basic(t *testing.T) {
	data := []byte("Hello, Content-Defined Chunking World! Let's see some boundaries.")
	c := NewCDCChunker(128)
	chunks := c.Split(data)

	if len(chunks) == 0 {
		t.Fatal("Expected chunks, got 0")
	}

	reassembled := Reassemble(chunks)
	if !bytes.Equal(data, reassembled) {
		t.Fatalf("Reassembled data doesn't match original. Len: orig=%d, reas=%d", len(data), len(reassembled))
	}
}

// TestCDC_ShiftingResistance proves that inserting a single byte
// at the beginning of a file only changes the first chunk, and the
// subsequent chunks remain identical, unlike fixed-size chunking.
func TestCDC_ShiftingResistance(t *testing.T) {
	// Generate 10KB of random pseudo-text data
	rng := rand.New(rand.NewSource(42))
	orig := make([]byte, 10*1024)
	for i := range orig {
		orig[i] = byte(rng.Intn(256))
	}

	// File A: Original
	// File B: Shifted (1 byte inserted at index 0)
	shifted := make([]byte, len(orig)+1)
	shifted[0] = 0xAA // Inserted byte
	copy(shifted[1:], orig)

	cdc := NewCDCChunker(128)
	chunksA := cdc.Split(orig)
	chunksB := cdc.Split(shifted)

	if len(chunksA) < 10 {
		t.Fatalf("Expected multiple chunks for 10KB data, got %d", len(chunksA))
	}

	// Compare hashes. We expect the first chunk(s) to differ, but the rest to synchronize and match.
	// Find the first matching hash in B for each hash in A (after the first few)
	
	matchCount := 0
	
	// Create a map of A hashes for easy lookup
	hashesA := make(map[uint64]bool)
	for _, c := range chunksA {
		hashesA[c.Hash] = true
	}

	for _, c := range chunksB {
		if hashesA[c.Hash] {
			matchCount++
		}
	}

	// A good CDC algorithm should synchronize quickly.
	// Out of ~100 chunks, we expect > 90% to match exactly despite the 1 byte shift.
	matchRatio := float64(matchCount) / float64(len(chunksA))

	if matchRatio < 0.90 {
		t.Errorf("CDC failed shifting resistance test. Match ratio: %.2f%% (Expected > 90%%)", matchRatio*100)
	} else {
		t.Logf("CDC Shifting Resistance Success! Match ratio: %.2f%%", matchRatio*100)
	}

	// For comparison, let's see how FixedChunker performs:
	fixed := NewFixedChunker(128)
	fixedA := fixed.Split(orig)
	fixedB := fixed.Split(shifted)

	fixedMatchCount := 0
	fixedHashesA := make(map[uint64]bool)
	for _, c := range fixedA {
		fixedHashesA[c.Hash] = true
	}

	for _, c := range fixedB {
		if fixedHashesA[c.Hash] {
			fixedMatchCount++
		}
	}

	fixedMatchRatio := float64(fixedMatchCount) / float64(len(fixedA))
	t.Logf("Fixed Chunker Match ratio: %.2f%%", fixedMatchRatio*100)

	if fixedMatchRatio > 0.05 {
		// It shouldn't match anything except by pure random luck
		t.Errorf("Fixed chunker matched too much? Ratio: %.2f%%", fixedMatchRatio*100)
	}
}

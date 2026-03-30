package search

import (
	"encoding/binary"
	"math"
	"math/bits"

	"golang.org/x/sys/cpu"
)

// MatchResult represents the outcome of a search operation.
type MatchResult struct {
	// CodebookID is the index of the matching codeword in the Codebook.
	CodebookID uint64

	// Pattern is the actual byte content of the codeword.
	Pattern []byte

	// Distance is the quantitative difference between the chunk and the codeword.
	// For bitwise Hamming distance, 0 means perfect match.
	Distance int
}

// Similarity returns a 0.0-1.0 value representing how closely the match
// resembles the input chunk. 1.0 = perfect match (distance=0), 0.0 = completely different.
// chunkBits is len(chunk)*8 (total bits in the input).
func (m MatchResult) Similarity(chunkBits int) float64 {
	if chunkBits == 0 {
		return 0
	}
	s := 1.0 - float64(m.Distance)/float64(chunkBits)
	if s < 0 {
		return 0
	}
	return s
}

// Searcher defines the interface for finding patterns in a Codebook.
type Searcher interface {
	// FindBestMatch searches for the codeword that is most similar to the given chunk.
	FindBestMatch(chunk []byte) (MatchResult, error)
	Restrict(allowed []uint64)
}

// hammingDistance calculates the number of mismatching bits between two byte slices.
func hammingDistance(a, b []byte) int {
	// O(1) Branch para Hardware Capabilities:
	if cpu.X86.HasAVX2 || cpu.X86.HasAVX512 || cpu.ARM64.HasASIMD {
		return hammingDistanceSIMD(a, b) // 256-bit unrolled via pipeline
	}

	dist := 0
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	// Process 8 bytes (64 bits) at a time
	blocks := minLen / 8
	for i := 0; i < blocks; i++ {
		offset := i * 8
		v1 := binary.LittleEndian.Uint64(a[offset:])
		v2 := binary.LittleEndian.Uint64(b[offset:])
		dist += bits.OnesCount64(v1 ^ v2)
	}

	// Process remaining bytes
	for i := blocks * 8; i < minLen; i++ {
		dist += bits.OnesCount8(a[i] ^ b[i])
	}

	// If lengths are different, missing bytes count as entirely mismatched
	if len(a) != len(b) {
		dist += int(math.Abs(float64(len(a)-len(b)))) * 8
	}

	return dist
}

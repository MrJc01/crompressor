// Package search provides mechanisms to find the closest matching codeword
// for a given chunk of data within a Codebook.
package search

import (
	"encoding/binary"
	"math"
	"math/bits"
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

// Searcher defines the interface for finding patterns in a Codebook.
type Searcher interface {
	// FindBestMatch searches for the codeword that is most similar to the given chunk.
	FindBestMatch(chunk []byte) (MatchResult, error)
}

// hammingDistance calculates the number of mismatching bits between two byte slices.
// Optimized to compare 64-bit blocks for massive performance gains.
func hammingDistance(a, b []byte) int {
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

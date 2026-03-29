package search

import (
	"errors"

	"github.com/MrJc01/crompressor/internal/codebook"
)

// LinearSearcher implements a brute-force exact matcher optimized for the MVP.
// It scans all codewords in the given codebook and calculates exact Hamming distance.
// While O(N) per chunk is slow for large codebooks, it is perfectly viable for a 1MB
// mini-codebook and guarantees finding the mathematically closest match without HNSW overhead.
type LinearSearcher struct {
	cb      *codebook.Reader
	allowed []uint64
}

// NewLinearSearcher creates a new LinearSearcher using the provided Codebook.
func NewLinearSearcher(cb *codebook.Reader) *LinearSearcher {
	return &LinearSearcher{cb: cb, allowed: nil}
}

// Restrict limits the linear search space to only the specified CodebookIDs.
func (ls *LinearSearcher) Restrict(allowed []uint64) {
	ls.allowed = allowed
}

// FindBestMatch sequentially searches the entire codebook for the closest match.
func (ls *LinearSearcher) FindBestMatch(chunk []byte) (MatchResult, error) {
	if ls.cb == nil {
		return MatchResult{}, errors.New("search: nil codebook")
	}

	count := ls.cb.CodewordCount()
	if count == 0 {
		return MatchResult{}, errors.New("search: empty codebook")
	}

	var bestMatchedID uint64
	var bestPattern []byte
	bestDistance := int(^uint(0) >> 1) // Max int

	if ls.allowed != nil {
		for _, id := range ls.allowed {
			pattern := ls.cb.LookupUnsafe(id)
			dist := hammingDistance(chunk, pattern)

			if dist < bestDistance {
				bestDistance = dist
				bestPattern = pattern
				bestMatchedID = id

				if dist == 0 {
					break
				}
			}
		}
	} else {
		for id := uint64(0); id < count; id++ {
			// Fast unprotected lookup since we know id < count
			pattern := ls.cb.LookupUnsafe(id)

			dist := hammingDistance(chunk, pattern)

			if dist < bestDistance {
				bestDistance = dist
				bestPattern = pattern
				bestMatchedID = id

				// Early exit on perfect match
				if dist == 0 {
					break
				}
			}
		}
	}

	return MatchResult{
		CodebookID: bestMatchedID,
		Pattern:    bestPattern,
		Distance:   bestDistance,
	}, nil
}

package search

import (
	"errors"
)

// MultiSearcher iterates through a hierarchy of Codebooks (L3->L2->L1).
// This enables "Transfer Learning" where a specific local codebook is
// consulted first, falling back to a universal codebook if no good match is found.
type MultiSearcher struct {
	searchers []*LSHSearcher
}

// NewMultiSearcher initializes a hierarchical searcher across multiple codebooks.
func NewMultiSearcher(searchers []*LSHSearcher) *MultiSearcher {
	return &MultiSearcher{
		searchers: searchers,
	}
}

// Restrict applies the vocabulary restriction to all active searchers.
func (m *MultiSearcher) Restrict(allowed []uint64) {
	for _, s := range m.searchers {
		s.Restrict(allowed)
	}
}

// FindBestMatch searches the tiers sequentially.
// It returns the first match that exceeds a strong similarity threshold (e.g. 50%),
// or the absolute best match across all tiers if none exceed the threshold.
func (m *MultiSearcher) FindBestMatch(chunk []byte) (MatchResult, error) {
	if len(m.searchers) == 0 {
		return MatchResult{}, errors.New("multi_search: no searchers provided")
	}

	var bestMatch MatchResult
	bestMatch.Distance = int(^uint(0) >> 1) // Max int
	
	// Pre-calculate bits for thresholding
	chunkBits := len(chunk) * 8
	// Target similarity to short-circuit the tier search: 50%
	// If a match is >= 50% similar, we consider it "good enough" to stop exploring lower tiers
	// since XOR deltas compress well above this threshold.
	targetDistance := chunkBits / 2 

	for tierIdx, s := range m.searchers {
		match, err := s.FindBestMatch(chunk)
		if err != nil {
			continue // Gracefully skip failed tiers
		}

		if match.Distance < bestMatch.Distance {
			bestMatch = match
			// Inject the Tier ID into the upper bits of the CodebookID
			// so the Unpacker knows WHICH codebook this ID belongs to!
			// We shift the Tier Index (0, 1, 2) to the highest 2 bits of the 64-bit ID.
			bestMatch.CodebookID = match.CodebookID | (uint64(tierIdx) << 62)
		}

		// Short-circuit: if we found an excellent match in an upper tier, don't waste CPU on L1
		if bestMatch.Distance <= targetDistance {
			break
		}
	}

	if bestMatch.Distance == int(^uint(0)>>1) {
		return MatchResult{}, errors.New("multi_search: failed to find any match across all tiers")
	}

	return bestMatch, nil
}

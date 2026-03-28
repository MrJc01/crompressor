package search

import (
	"errors"

	"github.com/MrJc01/crompressor/internal/codebook"
)

// LSHSearcher implements Locality Sensitive Hashing (LSH) for sub-linear search.
// Instead of O(N) linear scans, it groups codewords into buckets using a locality
// preserving hash. During search, it only scans codewords that mapped to the same bucket.
type LSHSearcher struct {
	cb      *codebook.Reader
	buckets map[uint16][]uint64
	// Fallback to linear if a bucket is empty (for the MVP to guarantee a result)
	linear *LinearSearcher
}

// NewLSHSearcher builds the spatial index over the Codebook in memory.
// This O(N) initialization cost is paid once and amortized over millions of chunks.
func NewLSHSearcher(cb *codebook.Reader) *LSHSearcher {
	ls := &LSHSearcher{
		cb:      cb,
		buckets: make(map[uint16][]uint64),
		linear:  NewLinearSearcher(cb),
	}

	ls.buildIndex()
	return ls
}

// buildIndex clusters all codewords into buckets based on the LSH function.
func (ls *LSHSearcher) buildIndex() {
	count := ls.cb.CodewordCount()
	for id := uint64(0); id < count; id++ {
		pattern := ls.cb.LookupUnsafe(id)
		hash := computeLSH(pattern)
		ls.buckets[hash] = append(ls.buckets[hash], id)
	}
}

// computeLSH generates a 16-bit locality sensitive hash.
// For binary Hamming space, an exact projection (e.g., first 16 bits)
// serves as an extremely fast and perfectly sensitive spatial clustering
// mechanism for exact and near-exact matches in those coordinates.
//
// In a full production model, multiple hash tables (forests) would be used.
func computeLSH(data []byte) uint16 {
	if len(data) >= 2 {
		// Use the first 2 bytes as the projection vector
		return uint16(data[0]) | uint16(data[1])<<8
	}
	return 0
}

// FindBestMatch finds the closest pattern by only scanning the target bucket.
// If the bucket is empty, it falls back to linear search to ensure a match.
func (ls *LSHSearcher) FindBestMatch(chunk []byte) (MatchResult, error) {
	if ls.cb == nil {
		return MatchResult{}, errors.New("search: nil codebook")
	}

	hash := computeLSH(chunk)
	candidates, ok := ls.buckets[hash]

	// MVP Fallback: if no patterns exist in this exact bucket, do a linear scan.
	// In HNSW or Multi-Probe LSH, we would check neighboring buckets instead.
	if !ok || len(candidates) == 0 {
		return ls.linear.FindBestMatch(chunk)
	}

	var bestMatchedID uint64
	var bestPattern []byte
	bestDistance := int(^uint(0) >> 1) // Max int

	for _, id := range candidates {
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

	return MatchResult{
		CodebookID: bestMatchedID,
		Pattern:    bestPattern,
		Distance:   bestDistance,
	}, nil
}

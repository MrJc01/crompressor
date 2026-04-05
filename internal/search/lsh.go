package search

import (
	"errors"
	"math"
	"sync"

	"github.com/MrJc01/crompressor/internal/codebook"
)

const lshCacheSize = 65536 // 64K entry LRU cache

// LSHSearcher implements Locality Sensitive Hashing (LSH) for sub-linear search.
// Instead of O(N) linear scans, it groups codewords into buckets using a locality
// preserving hash. During search, it only scans codewords that mapped to the same bucket.
type LSHSearcher struct {
	cb      *codebook.Reader
	buckets map[uint16][]uint64
	// Fallback to linear if a bucket is empty (for the MVP to guarantee a result)
	linear *LinearSearcher
	// V16: LRU Cache for O(1) repeated chunk matching
	cacheMu    sync.RWMutex
	cache      map[uint64]MatchResult
	cacheOrder []uint64 // simple ring buffer for eviction
	cacheIdx   int
}

// NewLSHSearcher builds the spatial index over the Codebook in memory.
// This O(N) initialization cost is paid once and amortized over millions of chunks.
func NewLSHSearcher(cb *codebook.Reader) *LSHSearcher {
	ls := &LSHSearcher{
		cb:         cb,
		buckets:    make(map[uint16][]uint64),
		linear:     NewLinearSearcher(cb),
		cache:      make(map[uint64]MatchResult, lshCacheSize),
		cacheOrder: make([]uint64, lshCacheSize),
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

// Restrict prunes the search space to only allowed CodebookIDs, ignoring the rest.
func (ls *LSHSearcher) Restrict(allowed []uint64) {
	allowedMap := make(map[uint64]bool, len(allowed))
	for _, id := range allowed {
		allowedMap[id] = true
	}

	for bucket, ids := range ls.buckets {
		var filtered []uint64
		for _, id := range ids {
			if allowedMap[id] {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) > 0 {
			ls.buckets[bucket] = filtered
		} else {
			delete(ls.buckets, bucket)
		}
	}
	if ls.linear != nil {
		ls.linear.Restrict(allowed)
	}
	// Invalidate cache after restriction
	ls.cacheMu.Lock()
	ls.cache = make(map[uint64]MatchResult, lshCacheSize)
	ls.cacheMu.Unlock()
}

// computeLSH generates a 16-bit locality sensitive hash.
// Uses the first 2 bytes as a fast projection vector for bucket assignment.
func computeLSH(data []byte) uint16 {
	if len(data) >= 2 {
		return uint16(data[0]) | uint16(data[1])<<8
	}
	return 0
}

// chunkHash computes a fast FNV-1a hash for cache lookup.
func chunkHash(data []byte) uint64 {
	var h uint64 = 14695981039346656037
	for _, b := range data {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}

// isHighEntropy checks if a chunk has entropy > 7.5 bits/byte (incompressible).
// Used as a Bloom-style pre-filter to skip LSH search entirely for random data.
func isHighEntropy(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	var freq [256]int
	for _, b := range data {
		freq[b]++
	}
	n := float64(len(data))
	var entropy float64
	for _, f := range freq {
		if f > 0 {
			p := float64(f) / n
			entropy -= p * math.Log2(p)
		}
	}
	return entropy > 7.5
}

// FindBestMatch finds the closest pattern by only scanning the target bucket.
// If the bucket is empty, it falls back to linear search to ensure a match.
// V16: Uses LRU cache for O(1) lookup on repeated chunks and entropy pre-filter.
func (ls *LSHSearcher) FindBestMatch(chunk []byte) (MatchResult, error) {
	if ls.cb == nil {
		return MatchResult{}, errors.New("search: nil codebook")
	}

	// V16: Check cache first (O(1))
	h := chunkHash(chunk)
	ls.cacheMu.RLock()
	if cached, ok := ls.cache[h]; ok {
		ls.cacheMu.RUnlock()
		return cached, nil
	}
	ls.cacheMu.RUnlock()

	// V16: Entropy pre-filter — skip LSH for incompressible data
	if isHighEntropy(chunk) {
		// Return a "worst possible" match so the compiler treats it as literal
		result := MatchResult{
			CodebookID: 0,
			Pattern:    ls.cb.LookupUnsafe(0),
			Distance:   len(chunk) * 8, // Maximum Hamming distance
		}
		ls.cacheStore(h, result)
		return result, nil
	}

	hash := computeLSH(chunk)
	candidates, ok := ls.buckets[hash]

	// MVP Fallback: if no patterns exist in this exact bucket, do a linear scan.
	// In HNSW or Multi-Probe LSH, we would check neighboring buckets instead.
	if !ok || len(candidates) == 0 {
		result, err := ls.linear.FindBestMatch(chunk)
		if err == nil {
			ls.cacheStore(h, result)
		}
		return result, err
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

	result := MatchResult{
		CodebookID: bestMatchedID,
		Pattern:    bestPattern,
		Distance:   bestDistance,
	}

	// Store in cache
	ls.cacheStore(h, result)

	return result, nil
}

// cacheStore adds a result to the LRU cache with ring-buffer eviction.
func (ls *LSHSearcher) cacheStore(h uint64, result MatchResult) {
	ls.cacheMu.Lock()
	defer ls.cacheMu.Unlock()

	if len(ls.cache) >= lshCacheSize {
		// Evict oldest entry
		delete(ls.cache, ls.cacheOrder[ls.cacheIdx])
	}
	ls.cache[h] = result
	ls.cacheOrder[ls.cacheIdx] = h
	ls.cacheIdx = (ls.cacheIdx + 1) % lshCacheSize
}


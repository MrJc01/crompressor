package trainer

import (
	"sort"
)

// SelectElite picks the top maxCodewords patterns by frequency,
// with LSH diversity filtering to avoid bucket saturation.
// maxPerBucket limits how many codewords share the same LSH bucket.
func SelectElite(table *FrequencyTable, maxCodewords int, maxPerBucket int) [][]byte {
	all := table.All()

	// Sort descending by frequency
	sort.Slice(all, func(i, j int) bool {
		return all[i].Count > all[j].Count
	})

	if maxPerBucket <= 0 {
		maxPerBucket = maxCodewords // No bucket limit
	}

	// Track how many codewords are in each LSH bucket
	bucketCount := make(map[uint16]int)
	selected := make([][]byte, 0, maxCodewords)

	for _, entry := range all {
		if len(selected) >= maxCodewords {
			break
		}

		bucket := computeLSHBucket(entry.Data)

		// Diversity filter: skip if this bucket is already saturated
		if bucketCount[bucket] >= maxPerBucket {
			continue
		}

		selected = append(selected, entry.Data)
		bucketCount[bucket]++
	}

	return selected
}

// computeLSHBucket generates a 16-bit locality hash (same algorithm as search/lsh.go).
func computeLSHBucket(data []byte) uint16 {
	if len(data) >= 2 {
		return uint16(data[0]) | uint16(data[1])<<8
	}
	return 0
}

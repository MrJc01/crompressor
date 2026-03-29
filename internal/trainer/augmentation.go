package trainer

import (
	"sort"
)

// AugmentPatterns takes a FrequencyTable and extracts the top 'limit' most frequent
// patterns. It generates slightly perturbed versions of these patterns (byte shifts,
// rotations) and injects them back into the table with a reduced weight.
// This prevents out-of-distribution performance drops ("memorization trap").
func AugmentPatterns(ft *FrequencyTable, limit int) {
	all := ft.All()

	// Sort descending by frequency
	sort.Slice(all, func(i, j int) bool {
		return all[i].Count > all[j].Count
	})

	if limit > len(all) {
		limit = len(all)
	}

	for i := 0; i < limit; i++ {
		entry := all[i]
		baseData := entry.Data
		baseCount := entry.Count
		if baseCount <= 1 {
			continue // Don't augment noise
		}

		// Reduced weight for generated patterns (e.g., half of base)
		augCount := baseCount / 2
		if augCount == 0 {
			augCount = 1
		}

		// 1. Shift Left by 1 byte (Pad with 0 on right)
		sL := make([]byte, len(baseData))
		copy(sL, baseData[1:])
		sL[len(baseData)-1] = 0
		ft.RecordWithCount(sL, augCount)

		// 2. Shift Right by 1 byte (Pad with 0 on left)
		sR := make([]byte, len(baseData))
		copy(sR[1:], baseData[:len(baseData)-1])
		sR[0] = 0
		ft.RecordWithCount(sR, augCount)

		// 3. Circular Rotation (+1 / -1) Byte
		rL := make([]byte, len(baseData))
		copy(rL, baseData[1:])
		rL[len(baseData)-1] = baseData[0]
		ft.RecordWithCount(rL, augCount)
	}
}

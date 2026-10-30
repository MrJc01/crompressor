package trainer

import (
	"sync"

	"github.com/cespare/xxhash/v2"
)

// PatternEntry holds a unique 128-byte pattern and how many times it was seen.
type PatternEntry struct {
	Hash  uint64
	Count uint32
	Data  []byte
	First uint64 // Position of the first occurrence, to number the brain in reading order
	Bucket uint16 // LSH bucket; Data may be nil until the selected patterns are read back
}

// FrequencyTable tracks the occurrence count of every unique chunk pattern.
// It is designed to be fed from a single collector goroutine (not concurrent writes).
type FrequencyTable struct {
	mu      sync.Mutex
	entries map[uint64]*PatternEntry
}

// NewFrequencyTable creates an empty frequency table.
func NewFrequencyTable() *FrequencyTable {
	return &FrequencyTable{
		entries: make(map[uint64]*PatternEntry),
	}
}

// Record registers a chunk pattern. If seen before, increments its count.
// Uses xxhash for O(1) lookups. The full 128B data is stored on first encounter.
func (ft *FrequencyTable) Record(data []byte) {
	h := xxhash.Sum64(data)

	ft.mu.Lock()
	defer ft.mu.Unlock()

	if entry, ok := ft.entries[h]; ok {
		entry.Count++
	} else {
		cp := make([]byte, len(data))
		copy(cp, data)
		ft.entries[h] = &PatternEntry{
			Hash:  h,
			Count: 1,
			Data:  cp,
		}
	}
}

// RecordAt is Record for a pattern at a known position in the training data. It keeps
// no bytes, so memory depends on the unique patterns and not on their size.
func (ft *FrequencyTable) RecordAt(data []byte, pos uint64) {
	h := xxhash.Sum64(data)

	ft.mu.Lock()
	defer ft.mu.Unlock()

	if entry, ok := ft.entries[h]; ok {
		entry.Count++
		if pos < entry.First {
			entry.First = pos
		}
	} else {
		ft.entries[h] = &PatternEntry{Hash: h, Count: 1, First: pos, Bucket: computeLSHBucket(data)}
	}
}

// RecordWithCount registers a chunk pattern with a specific initial count.
// Used by incremental training to seed existing patterns with a boost.
func (ft *FrequencyTable) RecordWithCount(data []byte, count uint32) {
	h := xxhash.Sum64(data)

	ft.mu.Lock()
	defer ft.mu.Unlock()

	if entry, ok := ft.entries[h]; ok {
		entry.Count += count
	} else {
		cp := make([]byte, len(data))
		copy(cp, data)
		ft.entries[h] = &PatternEntry{
			Hash:  h,
			Count: count,
			Data:  cp,
		}
	}
}

// Len returns the number of unique patterns recorded.
func (ft *FrequencyTable) Len() int {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	return len(ft.entries)
}

// All returns all pattern entries (unordered).
func (ft *FrequencyTable) All() []*PatternEntry {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	result := make([]*PatternEntry, 0, len(ft.entries))
	for _, e := range ft.entries {
		result = append(result, e)
	}
	return result
}

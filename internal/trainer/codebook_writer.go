package trainer

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"sort"

	"github.com/MrJc01/crompressor/internal/codebook"
)

// WriteCodebook generates a .cromdb file from the selected patterns.
// Patterns are sorted by LSH bucket for optimal mmap locality during search.
func WriteCodebook(path string, patterns [][]byte) error {
	if len(patterns) == 0 {
		return fmt.Errorf("trainer: no patterns to write")
	}

	cwSize := len(patterns[0])

	// Sort patterns by LSH bucket for spatial locality in mmap
	sort.SliceStable(patterns, func(i, j int) bool {
		return computeLSHBucket(patterns[i]) < computeLSHBucket(patterns[j])
	})

	// Compute build hash over all pattern data
	h := sha256.New()
	for _, p := range patterns {
		h.Write(p)
	}
	buildHash := h.Sum(nil)

	// Build header
	header := make([]byte, codebook.HeaderSize)
	copy(header[0:codebook.MagicSize], codebook.MagicString)
	binary.LittleEndian.PutUint16(header[6:8], codebook.Version1)
	binary.LittleEndian.PutUint16(header[8:10], uint16(cwSize))
	binary.LittleEndian.PutUint64(header[10:18], uint64(len(patterns)))
	binary.LittleEndian.PutUint64(header[18:26], codebook.HeaderSize)
	copy(header[26:58], buildHash[:32])

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("trainer: create codebook: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(header); err != nil {
		return err
	}

	for _, p := range patterns {
		if _, err := f.Write(p); err != nil {
			return err
		}
	}

	return nil
}
